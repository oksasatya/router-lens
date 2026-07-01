package event

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
)

type fakeAnalyticsRepo struct {
	overview    *eventdomain.OverviewTotals
	series      []eventdomain.SeriesPoint
	providers   []eventdomain.ProviderStat
	models      []eventdomain.ModelStat
	topProjects []eventdomain.ProjectStat
}

func (f *fakeAnalyticsRepo) Overview(context.Context, eventdomain.AnalyticsFilter) (*eventdomain.OverviewTotals, error) {
	return f.overview, nil
}
func (f *fakeAnalyticsRepo) Series(context.Context, eventdomain.AnalyticsFilter) ([]eventdomain.SeriesPoint, error) {
	return f.series, nil
}
func (f *fakeAnalyticsRepo) Providers(context.Context, eventdomain.AnalyticsFilter) ([]eventdomain.ProviderStat, error) {
	return f.providers, nil
}
func (f *fakeAnalyticsRepo) Models(context.Context, eventdomain.AnalyticsFilter) ([]eventdomain.ModelStat, error) {
	return f.models, nil
}
func (f *fakeAnalyticsRepo) TopProjects(context.Context, eventdomain.AnalyticsFilter, int) ([]eventdomain.ProjectStat, error) {
	return f.topProjects, nil
}

func TestAnalyticsServiceOverview(t *testing.T) {
	t.Run("assembles most-used provider/model, most-expensive model, error rate", func(t *testing.T) {
		cheapCost := decimal.NewFromFloat(1)
		pricyCost := decimal.NewFromFloat(99)
		repo := &fakeAnalyticsRepo{
			overview: &eventdomain.OverviewTotals{TotalRequests: 10, ErrorCount: 2},
			providers: []eventdomain.ProviderStat{
				{Provider: "anthropic", RequestCount: 7},
				{Provider: "openai", RequestCount: 3},
			},
			models: []eventdomain.ModelStat{
				{Provider: "anthropic", Model: "claude", RequestCount: 7, CostUSD: &cheapCost},
				{Provider: "openai", Model: "gpt", RequestCount: 3, CostUSD: &pricyCost},
			},
		}
		got, err := NewAnalyticsService(repo).Overview(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if got.MostUsedProvider != "anthropic" || got.MostUsedModel != "claude" {
			t.Fatalf("most-used: provider=%q model=%q", got.MostUsedProvider, got.MostUsedModel)
		}
		if got.MostExpensiveModel != "gpt" {
			t.Fatalf("most-expensive model = %q, want gpt", got.MostExpensiveModel)
		}
		if got.ErrorRate != 0.2 {
			t.Fatalf("error rate = %v, want 0.2", got.ErrorRate)
		}
	})
	t.Run("zero requests -> error rate 0, no panic, empty most-used fields", func(t *testing.T) {
		repo := &fakeAnalyticsRepo{overview: &eventdomain.OverviewTotals{}}
		got, err := NewAnalyticsService(repo).Overview(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if got.ErrorRate != 0 || got.MostUsedProvider != "" || got.MostUsedModel != "" || got.MostExpensiveModel != "" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("no priced model -> most-expensive model stays empty, not the first row", func(t *testing.T) {
		repo := &fakeAnalyticsRepo{
			overview: &eventdomain.OverviewTotals{TotalRequests: 1},
			models:   []eventdomain.ModelStat{{Provider: "p", Model: "unpriced-model", RequestCount: 1, CostUSD: nil}},
		}
		got, err := NewAnalyticsService(repo).Overview(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if got.MostExpensiveModel != "" {
			t.Fatalf("most-expensive model = %q, want empty (nothing priced)", got.MostExpensiveModel)
		}
	})
}

func TestAnalyticsServiceSeriesProjections(t *testing.T) {
	lat := 42.0
	series := []eventdomain.SeriesPoint{
		{InputTokens: 10, OutputTokens: 20, RequestCount: 5, ErrorCount: 1, AvgLatencyMs: &lat, P95LatencyMs: &lat},
	}
	repo := &fakeAnalyticsRepo{series: series}
	svc := NewAnalyticsService(repo)

	t.Run("TokensSeries projects input/output tokens only", func(t *testing.T) {
		got, err := svc.TokensSeries(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil || len(got) != 1 || got[0].InputTokens != 10 || got[0].OutputTokens != 20 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("ErrorSeries computes rate per bucket, zero requests -> rate 0", func(t *testing.T) {
		zero := []eventdomain.SeriesPoint{{RequestCount: 0, ErrorCount: 0}}
		got, err := NewAnalyticsService(&fakeAnalyticsRepo{series: zero}).ErrorSeries(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil || len(got) != 1 || got[0].ErrorRate != 0 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("CostSeries projects cost only", func(t *testing.T) {
		got, err := svc.CostSeries(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil || len(got) != 1 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
		// the shared fixture's SeriesPoint has no CostUSD set (nil) — confirm that
		// nil-ness survives the projection untouched, not coerced to a zero value.
		if got[0].CostUSD != nil {
			t.Fatalf("cost = %v, want nil (fixture has no CostUSD)", got[0].CostUSD)
		}
	})
	t.Run("CostSeries preserves a non-nil cost value", func(t *testing.T) {
		cost := decimal.NewFromFloat(12.5)
		priced := []eventdomain.SeriesPoint{{CostUSD: &cost}}
		got, err := NewAnalyticsService(&fakeAnalyticsRepo{series: priced}).CostSeries(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil || len(got) != 1 || got[0].CostUSD == nil || !got[0].CostUSD.Equal(cost) {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("LatencySeries projects avg/p95 latency only", func(t *testing.T) {
		got, err := svc.LatencySeries(context.Background(), eventdomain.AnalyticsFilter{})
		if err != nil || len(got) != 1 || got[0].AvgLatencyMs == nil || *got[0].AvgLatencyMs != 42.0 || got[0].P95LatencyMs == nil || *got[0].P95LatencyMs != 42.0 {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
}
