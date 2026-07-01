// Package event (analytics.go): the read-side aggregate types and the
// AnalyticsRepository port. Same domain-purity rule as entity.go — stdlib +
// shopspring/decimal only, no echo/pgx/adapter imports.
package event

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// AnalyticsFilter bounds every analytics query. Interval is only meaningful
// for Series ("" is fine for Overview/Providers/Models/TopProjects, which are
// not bucketed by time). ProjectID == "" means "all projects".
type AnalyticsFilter struct {
	ProjectID string
	From      time.Time
	To        time.Time
	Interval  string // "hour" | "day" | "week", validated by the caller
}

// OverviewTotals is the raw single-row aggregate for a date range. It carries
// no derived fields (error rate, most-used provider) — those are computed in
// the usecase layer from this plus Providers/Models/TopProjects, keeping SQL
// dumb and the business interpretation testable in Go.
type OverviewTotals struct {
	TotalRequests     int64
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalCostUSD      *decimal.Decimal // nil when every event in range is unpriced (or zero rows)
	UnpricedCount     int64
	AvgLatencyMs      *float64 // nil when no event in range carries a latency_ms
	P95LatencyMs      *float64
	ErrorCount        int64
}

// SeriesPoint is one time-bucketed row. It backs FOUR endpoints (tokens,
// cost, latency, errors) from ONE query — each endpoint's handler projects
// out only the fields it needs (see Task 2/3).
type SeriesPoint struct {
	Bucket       time.Time
	InputTokens  int64
	OutputTokens int64
	CostUSD      *decimal.Decimal
	RequestCount int64
	ErrorCount   int64
	AvgLatencyMs *float64
	P95LatencyMs *float64
}

// ProviderStat is one provider's distribution over the range.
type ProviderStat struct {
	Provider     string
	RequestCount int64
	InputTokens  int64
	OutputTokens int64
	CostUSD      *decimal.Decimal
}

// ModelStat is one (provider, model) pair's distribution over the range.
type ModelStat struct {
	Provider     string
	Model        string
	RequestCount int64
	InputTokens  int64
	OutputTokens int64
	CostUSD      *decimal.Decimal
}

// ProjectStat is one project's request count over the range, used by
// Overview's "top projects by usage".
type ProjectStat struct {
	ProjectID    string
	ProjectName  string
	RequestCount int64
}

// AnalyticsRepository is the read-only port over llm_events aggregates.
// Every method is bounded by AnalyticsFilter.From/To (enforced by the caller
// via datetime.ParseRange) — none of these ever scans the whole table.
type AnalyticsRepository interface {
	Overview(ctx context.Context, f AnalyticsFilter) (*OverviewTotals, error)
	// Series buckets by f.Interval ("hour"/"day"/"week") — backs tokens/cost/latency/errors.
	Series(ctx context.Context, f AnalyticsFilter) ([]SeriesPoint, error)
	Providers(ctx context.Context, f AnalyticsFilter) ([]ProviderStat, error)
	Models(ctx context.Context, f AnalyticsFilter) ([]ModelStat, error)
	// TopProjects ignores f.ProjectID (it exists to compare projects) and returns
	// at most limit rows, ordered by request count descending.
	TopProjects(ctx context.Context, f AnalyticsFilter, limit int) ([]ProjectStat, error)
}
