// Package event (analytics.go): the AnalyticsService use case. Orchestrates
// AnalyticsRepository calls into the seven endpoint shapes and computes every
// derived number (rates, "most used") here in Go — the repository only ever
// returns raw grouped totals, never a business interpretation.
package event

import (
	"context"

	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
)

const topProjectsLimit = 5

type AnalyticsService struct {
	repo eventdomain.AnalyticsRepository
}

func NewAnalyticsService(repo eventdomain.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// OverviewResult is Overview's fully-assembled response shape.
type OverviewResult struct {
	Totals             *eventdomain.OverviewTotals
	ErrorRate          float64
	MostUsedProvider   string
	MostUsedModel      string
	MostExpensiveModel string
	TopProjects        []eventdomain.ProjectStat
}

func (s *AnalyticsService) Overview(ctx context.Context, f eventdomain.AnalyticsFilter) (*OverviewResult, error) {
	totals, err := s.repo.Overview(ctx, f)
	if err != nil {
		return nil, err
	}
	providers, err := s.repo.Providers(ctx, f)
	if err != nil {
		return nil, err
	}
	models, err := s.repo.Models(ctx, f)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopProjects(ctx, f, topProjectsLimit)
	if err != nil {
		return nil, err
	}
	return &OverviewResult{
		Totals:             totals,
		ErrorRate:          safeRate(totals.ErrorCount, totals.TotalRequests),
		MostUsedProvider:   mostUsedProvider(providers),
		MostUsedModel:      mostUsedModel(models),
		MostExpensiveModel: mostExpensiveModel(models),
		TopProjects:        top,
	}, nil
}

// safeRate divides part/whole, returning 0 (not NaN/Inf) when whole is zero.
func safeRate(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

// mostUsedProvider returns the provider with the highest request count, or ""
// when there are none. Providers is already the full grouped result — small
// (a handful of distinct providers), so a linear scan is the right tool.
func mostUsedProvider(stats []eventdomain.ProviderStat) string {
	var best eventdomain.ProviderStat
	for _, s := range stats {
		if s.RequestCount > best.RequestCount {
			best = s
		}
	}
	return best.Provider
}

func mostUsedModel(stats []eventdomain.ModelStat) string {
	var best eventdomain.ModelStat
	for _, s := range stats {
		if s.RequestCount > best.RequestCount {
			best = s
		}
	}
	return best.Model
}

// mostExpensiveModel returns the model with the highest total cost, skipping
// unpriced models entirely (nil CostUSD) so an unpriced model never wins by
// virtue of being compared against a zero value.
func mostExpensiveModel(stats []eventdomain.ModelStat) string {
	var best eventdomain.ModelStat
	var bestCost decimal.Decimal
	found := false
	for _, s := range stats {
		if s.CostUSD == nil {
			continue
		}
		if !found || s.CostUSD.GreaterThan(bestCost) {
			best, bestCost, found = s, *s.CostUSD, true
		}
	}
	return best.Model
}

// TokenPoint, CostPoint, LatencyPoint, and ErrorPoint are the four
// projections of eventdomain.SeriesPoint that the four series endpoints need.
type TokenPoint struct {
	Bucket       string
	InputTokens  int64
	OutputTokens int64
}
type CostPoint struct {
	Bucket  string
	CostUSD *decimal.Decimal
}
type LatencyPoint struct {
	Bucket       string
	AvgLatencyMs *float64
	P95LatencyMs *float64
}
type ErrorPoint struct {
	Bucket       string
	RequestCount int64
	ErrorCount   int64
	ErrorRate    float64
}

func (s *AnalyticsService) series(ctx context.Context, f eventdomain.AnalyticsFilter) ([]eventdomain.SeriesPoint, error) {
	return s.repo.Series(ctx, f)
}

func (s *AnalyticsService) TokensSeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]TokenPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]TokenPoint, 0, len(points))
	for _, p := range points {
		out = append(out, TokenPoint{Bucket: p.Bucket.UTC().Format(bucketLayout), InputTokens: p.InputTokens, OutputTokens: p.OutputTokens})
	}
	return out, nil
}

func (s *AnalyticsService) CostSeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]CostPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]CostPoint, 0, len(points))
	for _, p := range points {
		out = append(out, CostPoint{Bucket: p.Bucket.UTC().Format(bucketLayout), CostUSD: p.CostUSD})
	}
	return out, nil
}

func (s *AnalyticsService) LatencySeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]LatencyPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]LatencyPoint, 0, len(points))
	for _, p := range points {
		out = append(out, LatencyPoint{Bucket: p.Bucket.UTC().Format(bucketLayout), AvgLatencyMs: p.AvgLatencyMs, P95LatencyMs: p.P95LatencyMs})
	}
	return out, nil
}

func (s *AnalyticsService) ErrorSeries(ctx context.Context, f eventdomain.AnalyticsFilter) ([]ErrorPoint, error) {
	points, err := s.series(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]ErrorPoint, 0, len(points))
	for _, p := range points {
		out = append(out, ErrorPoint{
			Bucket: p.Bucket.UTC().Format(bucketLayout), RequestCount: p.RequestCount,
			ErrorCount: p.ErrorCount, ErrorRate: safeRate(p.ErrorCount, p.RequestCount),
		})
	}
	return out, nil
}

func (s *AnalyticsService) Providers(ctx context.Context, f eventdomain.AnalyticsFilter) ([]eventdomain.ProviderStat, error) {
	return s.repo.Providers(ctx, f)
}

func (s *AnalyticsService) Models(ctx context.Context, f eventdomain.AnalyticsFilter) ([]eventdomain.ModelStat, error) {
	return s.repo.Models(ctx, f)
}

// bucketLayout renders a series bucket timestamp on the wire. RFC3339 like
// every other timestamp in this API (dto/format.go's timeLayout, same value).
const bucketLayout = "2006-01-02T15:04:05Z07:00"
