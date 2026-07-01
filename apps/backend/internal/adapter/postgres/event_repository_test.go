package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/event"
	"router-lens/internal/domain/project"
)

func seedProject(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *project.Project {
	t.Helper()
	var ownerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1,$2,$3) RETURNING id`,
		"event-owner@test.local", "x", "Owner").Scan(&ownerID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	p := &project.Project{OwnerUserID: ownerID, Name: "Ev", Slug: "ev", Description: ""}
	if err := NewProjectRepository(pool).Create(ctx, p); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return p
}

func TestEventRepositoryInsert(t *testing.T) {
	ctx, pool := setupTestDB(t)
	proj := seedProject(t, ctx, pool)
	repo := NewEventRepository(pool)

	mk := func() *event.Event {
		return &event.Event{
			ProjectID: proj.ID, EventID: "evt-1", Provider: "anthropic", Model: "claude",
			InputTokens: 100, OutputTokens: 50, IsError: false,
			Metadata:         json.RawMessage(`{"workspace":"nuvora"}`),
			RequestStartedAt: time.Now().UTC().Add(-time.Minute), ReceivedAt: time.Now().UTC(),
		}
	}

	t.Run("first insert stores + metadata round-trips, duplicate deduplicates", func(t *testing.T) {
		e := mk()
		inserted, err := repo.Insert(ctx, e)
		if err != nil || !inserted {
			t.Fatalf("first insert: inserted=%v err=%v", inserted, err)
		}
		got, err := repo.FindByID(ctx, e.ID)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		if string(got.Metadata) == "" || got.Provider != "anthropic" {
			t.Fatalf("round-trip mismatch: %+v", got)
		}
		inserted2, err := repo.Insert(ctx, mk())
		if err != nil {
			t.Fatalf("dup insert: %v", err)
		}
		if inserted2 {
			t.Fatal("duplicate event_id should not insert again")
		}
	})

	t.Run("empty event_id always inserts", func(t *testing.T) {
		e1, e2 := mk(), mk()
		e1.EventID, e2.EventID = "", ""
		i1, err := repo.Insert(ctx, e1)
		if err != nil {
			t.Fatalf("insert e1: %v", err)
		}
		i2, err := repo.Insert(ctx, e2)
		if err != nil {
			t.Fatalf("insert e2: %v", err)
		}
		if !i1 || !i2 {
			t.Fatalf("null event_id must always insert: %v %v", i1, i2)
		}
		got, err := repo.FindByID(ctx, e1.ID)
		if err != nil {
			t.Fatalf("find e1 with null event_id: %v", err)
		}
		if got.EventID != "" {
			t.Fatalf("want EventID == \"\" for a NULL event_id row, got %q", got.EventID)
		}
	})
}

func TestEventRepositoryList(t *testing.T) {
	ctx, pool := setupTestDB(t)
	proj := seedProject(t, ctx, pool)
	repo := NewEventRepository(pool)

	base := time.Now().UTC().Add(-time.Hour)
	for i := range 3 {
		e := &event.Event{
			ProjectID: proj.ID, Provider: "p", Model: "m", InputTokens: 1, OutputTokens: 1,
			RequestStartedAt: base.Add(time.Duration(i) * time.Minute), ReceivedAt: time.Now().UTC(),
		}
		if _, err := repo.Insert(ctx, e); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}

	t.Run("newest first, project-scoped", func(t *testing.T) {
		list, err := repo.List(ctx, event.Filter{ProjectID: proj.ID, Limit: 10})
		if err != nil || len(list) != 3 {
			t.Fatalf("list: len=%d err=%v", len(list), err)
		}
		if !list[0].RequestStartedAt.After(list[1].RequestStartedAt) {
			t.Fatal("expected newest first")
		}
	})

	t.Run("findbyid missing -> ErrNotFound", func(t *testing.T) {
		if _, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); err != event.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("keyset paging: limit 2 + cursor returns the next page", func(t *testing.T) {
		page1, err := repo.List(ctx, event.Filter{ProjectID: proj.ID, Limit: 2})
		if err != nil || len(page1) != 2 {
			t.Fatalf("page1: len=%d err=%v", len(page1), err)
		}
		last := page1[len(page1)-1]
		page2, err := repo.List(ctx, event.Filter{
			ProjectID: proj.ID, Limit: 2,
			CursorTime: last.RequestStartedAt, CursorID: last.ID,
		})
		if err != nil {
			t.Fatalf("page2: %v", err)
		}
		for _, e := range page2 {
			if e.ID == last.ID {
				t.Fatalf("page2 must not repeat the cursor row %s", last.ID)
			}
		}
	})
}
