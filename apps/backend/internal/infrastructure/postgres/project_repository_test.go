package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/project"
)

// setupTestDB opens a pool against TEST_DATABASE_URL (skips when unset), applies
// migrations, and truncates all tables so each integration test starts clean
// (re-running must not fail on unique constraints). Shared by every postgres
// integration test in this package.
func setupTestDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE llm_events, api_keys, pricing_rules, sessions, projects, users CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return ctx, pool
}

func TestProjectRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)
	repo := NewProjectRepository(pool)

	// An owner user is required by the FK. Insert one directly.
	var ownerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		"proj-owner@test.local", "x", "Owner").Scan(&ownerID); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("create then find", func(t *testing.T) {
		p := &project.Project{OwnerUserID: ownerID, Name: "Alpha", Slug: "alpha", Description: "first"}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("create: %v", err)
		}
		if p.ID == "" {
			t.Fatal("id not set")
		}
		got, err := repo.FindByID(ctx, p.ID)
		if err != nil || got.Name != "Alpha" || got.Slug != "alpha" {
			t.Fatalf("find: %+v err=%v", got, err)
		}
	})

	t.Run("duplicate slug for same owner -> ErrSlugTaken", func(t *testing.T) {
		if err := repo.Create(ctx, &project.Project{OwnerUserID: ownerID, Name: "Beta", Slug: "dup", Description: ""}); err != nil {
			t.Fatalf("seed Beta: %v", err)
		}
		err := repo.Create(ctx, &project.Project{OwnerUserID: ownerID, Name: "Beta2", Slug: "dup", Description: ""})
		if err != project.ErrSlugTaken {
			t.Fatalf("want ErrSlugTaken, got %v", err)
		}
	})

	t.Run("update changes name, keeps slug", func(t *testing.T) {
		p := &project.Project{OwnerUserID: ownerID, Name: "Gamma", Slug: "gamma", Description: ""}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("create Gamma: %v", err)
		}
		p.Name = "Gamma Renamed"
		if err := repo.Update(ctx, p); err != nil {
			t.Fatalf("update: %v", err)
		}
		got, err := repo.FindByID(ctx, p.ID)
		if err != nil {
			t.Fatalf("find Gamma: %v", err)
		}
		if got.Name != "Gamma Renamed" || got.Slug != "gamma" {
			t.Fatalf("update result: %+v", got)
		}
	})

	t.Run("delete missing -> ErrNotFound", func(t *testing.T) {
		if err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000000"); err != project.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
