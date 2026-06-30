package datetime

import (
	"testing"
	"time"
)

func TestParseRange(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)

	t.Run("default is last 24h", func(t *testing.T) {
		r, err := ParseRange("", "", "", now)
		if err != nil || !r.To.Equal(now) || !r.From.Equal(now.Add(-24*time.Hour)) {
			t.Fatalf("default wrong: %+v err=%v", r, err)
		}
	})

	t.Run("preset 7d", func(t *testing.T) {
		r, err := ParseRange("", "", "7d", now)
		if err != nil {
			t.Fatalf("preset 7d: %v", err)
		}
		if !r.From.Equal(now.AddDate(0, 0, -7)) {
			t.Fatalf("7d wrong: %+v", r)
		}
	})

	t.Run("explicit range", func(t *testing.T) {
		from := "2026-06-01T00:00:00Z"
		to := "2026-06-10T00:00:00Z"
		r, err := ParseRange(from, to, "", now)
		if err != nil || r.From.Day() != 1 || r.To.Day() != 10 {
			t.Fatalf("explicit wrong: %+v err=%v", r, err)
		}
	})

	t.Run("rejects inverted and over-wide ranges", func(t *testing.T) {
		if _, err := ParseRange("2026-06-10T00:00:00Z", "2026-06-01T00:00:00Z", "", now); err == nil {
			t.Fatal("expected error for from>to")
		}
		if _, err := ParseRange("2026-01-01T00:00:00Z", "2026-06-01T00:00:00Z", "", now); err == nil {
			t.Fatal("expected error for >90d window")
		}
	})
}
