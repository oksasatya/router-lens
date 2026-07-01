package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"router-lens/internal/domain/event"
	"router-lens/internal/domain/project"
)

// seedEvents inserts n events for proj, evenly spaced one hour apart starting
// at base, alternating priced/unpriced and one error, so every aggregate
// (cost NULL-handling, error count, latency) has non-trivial data to check.
func seedEvents(t *testing.T, ctx context.Context, pool *pgxpool.Pool, proj *project.Project, base time.Time, n int) {
	t.Helper()
	repo := NewEventRepository(pool)
	for i := 0; i < n; i++ {
		lat := 100 + i*10
		status := 200
		e := &event.Event{
			ProjectID: proj.ID, Provider: "anthropic", Model: "claude",
			InputTokens: 1000, OutputTokens: 500, LatencyMs: &lat, StatusCode: &status,
			RequestStartedAt: base.Add(time.Duration(i) * time.Hour), ReceivedAt: time.Now().UTC(),
		}
		if i%2 == 0 { // half the events are priced
			cost := decimal.NewFromFloat(0.01)
			e.CostUSD, e.InputPrice1M, e.OutputPrice1M = &cost, &cost, &cost
		}
		if i == 0 { // exactly one error
			bad := 500
			e.StatusCode = &bad
			e.IsError = true
		}
		if _, err := repo.Insert(ctx, e); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
}

func TestAnalyticsRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)
	proj := seedProject(t, ctx, pool)
	base := time.Now().UTC().Add(-24 * time.Hour)
	seedEvents(t, ctx, pool, proj, base, 4)
	repo := NewAnalyticsRepository(pool)
	rng := event.AnalyticsFilter{ProjectID: proj.ID, From: base.Add(-time.Minute), To: base.Add(24 * time.Hour)}

	t.Run("Overview totals + unpriced count + one error", func(t *testing.T) {
		got, err := repo.Overview(ctx, rng)
		if err != nil {
			t.Fatalf("overview: %v", err)
		}
		if got.TotalRequests != 4 || got.ErrorCount != 1 || got.UnpricedCount != 2 {
			t.Fatalf("got %+v", got)
		}
		if got.TotalInputTokens != 4000 || got.TotalOutputTokens != 2000 {
			t.Fatalf("wrong token sums: got %+v", got)
		}
		if got.TotalCostUSD == nil {
			t.Fatal("expected a non-nil total cost (2 of 4 events priced)")
		}
	})

	t.Run("Series buckets by hour and sums each bucket independently", func(t *testing.T) {
		f := rng
		f.Interval = "hour"
		points, err := repo.Series(ctx, f)
		if err != nil {
			t.Fatalf("series: %v", err)
		}
		if len(points) != 4 {
			t.Fatalf("expected 4 hourly buckets, got %d", len(points))
		}
		for i, p := range points {
			if p.InputTokens != 1000 || p.OutputTokens != 500 {
				t.Fatalf("bucket %d: wrong token totals, got %+v", i, p)
			}
			wantErrors := int64(0)
			if i == 0 { // seedEvents marks index 0 as the sole error
				wantErrors = 1
			}
			if p.ErrorCount != wantErrors {
				t.Fatalf("bucket %d: expected ErrorCount=%d, got %d", i, wantErrors, p.ErrorCount)
			}
			if i%2 == 0 { // seedEvents prices indices 0,2
				if p.CostUSD == nil {
					t.Fatalf("bucket %d: expected priced (non-nil) cost, got nil", i)
				}
			} else if p.CostUSD != nil { // indices 1,3 are unpriced
				t.Fatalf("bucket %d: expected nil cost (unpriced), got %v", i, *p.CostUSD)
			}
		}
	})

	t.Run("Providers groups by provider", func(t *testing.T) {
		stats, err := repo.Providers(ctx, rng)
		if err != nil || len(stats) != 1 || stats[0].Provider != "anthropic" || stats[0].RequestCount != 4 {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	})

	t.Run("Models groups by provider+model", func(t *testing.T) {
		stats, err := repo.Models(ctx, rng)
		if err != nil || len(stats) != 1 || stats[0].Model != "claude" {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	})

	t.Run("TopProjects ranks by request count across all projects", func(t *testing.T) {
		stats, err := repo.TopProjects(ctx, event.AnalyticsFilter{From: rng.From, To: rng.To}, 5)
		if err != nil || len(stats) == 0 || stats[0].ProjectID != proj.ID {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	})
}

// TestAnalyticsRepository_AllUnpriced covers the review-flagged gap: an
// overview built entirely from unpriced events must report TotalCostUSD as
// nil, never a silently-wrong 0 (decision 10 — zero would be a lie about cost).
func TestAnalyticsRepository_AllUnpriced(t *testing.T) {
	ctx, pool := setupTestDB(t)
	proj := seedProject(t, ctx, pool)
	base := time.Now().UTC().Add(-24 * time.Hour)

	repo := NewEventRepository(pool)
	for i := 0; i < 3; i++ {
		lat := 100 + i*10
		status := 200
		e := &event.Event{
			ProjectID: proj.ID, Provider: "anthropic", Model: "claude",
			InputTokens: 1000, OutputTokens: 500, LatencyMs: &lat, StatusCode: &status,
			RequestStartedAt: base.Add(time.Duration(i) * time.Hour), ReceivedAt: time.Now().UTC(),
		} // no CostUSD/InputPrice1M/OutputPrice1M set — every event stays unpriced
		if _, err := repo.Insert(ctx, e); err != nil {
			t.Fatalf("seed unpriced event %d: %v", i, err)
		}
	}

	analytics := NewAnalyticsRepository(pool)
	rng := event.AnalyticsFilter{ProjectID: proj.ID, From: base.Add(-time.Minute), To: base.Add(24 * time.Hour)}
	got, err := analytics.Overview(ctx, rng)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if got.TotalRequests != 3 || got.UnpricedCount != 3 {
		t.Fatalf("got %+v", got)
	}
	if got.TotalCostUSD != nil {
		t.Fatalf("expected nil TotalCostUSD for an all-unpriced project, got %v", *got.TotalCostUSD)
	}
}
