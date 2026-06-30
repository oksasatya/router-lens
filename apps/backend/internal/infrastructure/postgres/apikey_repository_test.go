package postgres

import (
	"testing"

	"router-lens/internal/domain/apikey"
	"router-lens/internal/domain/project"
)

func TestAPIKeyRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)

	var ownerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		"key-owner@test.local", "x", "Owner").Scan(&ownerID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	proj := &project.Project{OwnerUserID: ownerID, Name: "Keys", Slug: "keys", Description: ""}
	if err := NewProjectRepository(pool).Create(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	repo := NewAPIKeyRepository(pool)

	t.Run("create, list, revoke", func(t *testing.T) {
		k := &apikey.APIKey{ProjectID: proj.ID, Name: "ci", KeyHash: "hash-abc", KeyPrefix: "rl_live_ab"}
		if err := repo.Create(ctx, k); err != nil {
			t.Fatalf("create: %v", err)
		}
		list, err := repo.ListByProject(ctx, proj.ID)
		if err != nil || len(list) != 1 || list[0].RevokedAt != nil {
			t.Fatalf("list: %+v err=%v", list, err)
		}
		if err := repo.Revoke(ctx, k.ID); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		list, err = repo.ListByProject(ctx, proj.ID)
		if err != nil || len(list) != 1 {
			t.Fatalf("relist: %+v err=%v", list, err)
		}
		if list[0].RevokedAt == nil {
			t.Fatal("revoked_at should be set")
		}
	})

	t.Run("revoke missing -> ErrNotFound", func(t *testing.T) {
		if err := repo.Revoke(ctx, "00000000-0000-0000-0000-000000000000"); err != apikey.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
