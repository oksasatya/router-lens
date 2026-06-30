package pagination

import (
	"testing"
	"time"
)

func TestOffset(t *testing.T) {
	t.Run("defaults and clamps", func(t *testing.T) {
		if o := ParseOffset("", ""); o.Page != 1 || o.Limit != 20 {
			t.Fatalf("defaults wrong: %+v", o)
		}
		if o := ParseOffset("3", "500"); o.Limit != 100 || o.SQLOffset() != 200 {
			t.Fatalf("clamp/offset wrong: %+v off=%d", o, o.SQLOffset())
		}
		if o := ParseOffset("0", "-5"); o.Page != 1 || o.Limit != 20 {
			t.Fatalf("invalid should fall back: %+v", o)
		}
	})
}

func TestCursor(t *testing.T) {
	t.Run("round-trips", func(t *testing.T) {
		now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
		c := Cursor{Time: now, ID: "abc-123"}
		got, err := DecodeCursor(EncodeCursor(c))
		if err != nil || !got.Time.Equal(now) || got.ID != "abc-123" {
			t.Fatalf("round-trip failed: %+v err=%v", got, err)
		}
	})

	t.Run("empty cursor is zero value", func(t *testing.T) {
		got, err := DecodeCursor("")
		if err != nil || !got.Time.IsZero() || got.ID != "" {
			t.Fatalf("empty cursor wrong: %+v err=%v", got, err)
		}
	})

	t.Run("garbage errors", func(t *testing.T) {
		if _, err := DecodeCursor("!!!not-base64!!!"); err == nil {
			t.Fatal("expected error for garbage cursor")
		}
	})
}
