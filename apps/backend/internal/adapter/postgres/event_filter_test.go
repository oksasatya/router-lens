package postgres

import (
	"testing"
	"time"

	"router-lens/internal/domain/event"
)

func TestBuildEventWhere(t *testing.T) {
	t.Run("project + range + cursor produce ordered placeholders", func(t *testing.T) {
		f := event.Filter{
			ProjectID:  "p1",
			From:       time.Unix(1000, 0).UTC(),
			To:         time.Unix(2000, 0).UTC(),
			CursorTime: time.Unix(1500, 0).UTC(),
			CursorID:   "c1",
			Limit:      20,
		}
		where, args := buildEventWhere(f)
		if where == "" || len(args) == 0 {
			t.Fatalf("expected conditions, got where=%q args=%v", where, args)
		}
		if args[0] != "p1" {
			t.Fatalf("first arg should be project id, got %v", args[0])
		}
		// last two args are the cursor tuple (time, id)
		if args[len(args)-1] != "c1" {
			t.Fatalf("last arg should be cursor id, got %v", args[len(args)-1])
		}
	})
	t.Run("no filters yields empty where and no args", func(t *testing.T) {
		where, args := buildEventWhere(event.Filter{})
		if where != "" || len(args) != 0 {
			t.Fatalf("expected empty, got where=%q args=%v", where, args)
		}
	})
	t.Run("single project filter produces exact where and args", func(t *testing.T) {
		where, args := buildEventWhere(event.Filter{ProjectID: "p1"})
		if where != "WHERE project_id = $1" {
			t.Fatalf("unexpected where: %q", where)
		}
		if len(args) != 1 || args[0] != "p1" {
			t.Fatalf("unexpected args: %v", args)
		}
	})
	t.Run("provider, model, is_error each get their own numbered placeholder", func(t *testing.T) {
		isErr := true
		where, args := buildEventWhere(event.Filter{Provider: "anthropic", Model: "claude", IsError: &isErr})
		want := "WHERE provider = $1 AND model = $2 AND is_error = $3"
		if where != want {
			t.Fatalf("where = %q, want %q", where, want)
		}
		if len(args) != 3 || args[0] != "anthropic" || args[1] != "claude" || args[2] != true {
			t.Fatalf("unexpected args: %v", args)
		}
	})
	t.Run("partial cursor (id only) is ignored", func(t *testing.T) {
		where, args := buildEventWhere(event.Filter{CursorID: "c1"})
		if where != "" || len(args) != 0 {
			t.Fatalf("expected empty (partial cursor must not build a predicate), got where=%q args=%v", where, args)
		}
	})
	t.Run("partial cursor (time only) is ignored", func(t *testing.T) {
		where, args := buildEventWhere(event.Filter{CursorTime: time.Unix(1500, 0).UTC()})
		if where != "" || len(args) != 0 {
			t.Fatalf("expected empty (partial cursor must not build a predicate), got where=%q args=%v", where, args)
		}
	})
}
