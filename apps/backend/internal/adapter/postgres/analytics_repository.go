package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/event"
)

type AnalyticsRepository struct{ pool *pgxpool.Pool }

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

var _ event.AnalyticsRepository = (*AnalyticsRepository)(nil)

// bucketExpr validates interval against a fixed allow-list and returns the
// date_trunc SQL fragment. This is the ONLY place a non-parameterized value
// is interpolated into a query in this file — interval can only ever be one
// of the three literal strings below, never raw user input (gosec).
func bucketExpr(interval string) (string, error) {
	switch interval {
	case "hour", "day", "week":
		return fmt.Sprintf("date_trunc('%s', request_started_at)", interval), nil
	default:
		return "", fmt.Errorf("postgres: invalid analytics interval %q", interval)
	}
}

// analyticsWhere builds the mandatory bounded-range WHERE clause + optional
// project scope, shared by every method below. Returns the clause and its
// positional args; the caller appends any further $n placeholders after it.
func analyticsWhere(f event.AnalyticsFilter) (string, []any) {
	if f.ProjectID == "" {
		return "WHERE request_started_at BETWEEN $1 AND $2", []any{f.From, f.To}
	}
	return "WHERE request_started_at BETWEEN $1 AND $2 AND project_id = $3", []any{f.From, f.To, f.ProjectID}
}

func (r *AnalyticsRepository) Overview(ctx context.Context, f event.AnalyticsFilter) (*event.OverviewTotals, error) {
	where, args := analyticsWhere(f)
	q := `SELECT
			COUNT(*),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			SUM(cost_usd),
			COUNT(*) FILTER (WHERE cost_usd IS NULL),
			AVG(latency_ms)::float8,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms)::float8,
			COUNT(*) FILTER (WHERE is_error)
		FROM llm_events ` + where
	var t event.OverviewTotals
	err := r.pool.QueryRow(ctx, q, args...).Scan(
		&t.TotalRequests, &t.TotalInputTokens, &t.TotalOutputTokens, &t.TotalCostUSD,
		&t.UnpricedCount, &t.AvgLatencyMs, &t.P95LatencyMs, &t.ErrorCount,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *AnalyticsRepository) Series(ctx context.Context, f event.AnalyticsFilter) ([]event.SeriesPoint, error) {
	bucket, err := bucketExpr(f.Interval)
	if err != nil {
		return nil, err
	}
	where, args := analyticsWhere(f)
	q := `SELECT ` + bucket + ` AS bucket,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			SUM(cost_usd),
			COUNT(*),
			COUNT(*) FILTER (WHERE is_error),
			AVG(latency_ms)::float8,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms)::float8
		FROM llm_events ` + where + `
		GROUP BY bucket ORDER BY bucket`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.SeriesPoint, 0)
	for rows.Next() {
		var p event.SeriesPoint
		if err := rows.Scan(&p.Bucket, &p.InputTokens, &p.OutputTokens, &p.CostUSD,
			&p.RequestCount, &p.ErrorCount, &p.AvgLatencyMs, &p.P95LatencyMs); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) Providers(ctx context.Context, f event.AnalyticsFilter) ([]event.ProviderStat, error) {
	where, args := analyticsWhere(f)
	q := `SELECT provider, COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), SUM(cost_usd)
		FROM llm_events ` + where + `
		GROUP BY provider ORDER BY 2 DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.ProviderStat, 0)
	for rows.Next() {
		var s event.ProviderStat
		if err := rows.Scan(&s.Provider, &s.RequestCount, &s.InputTokens, &s.OutputTokens, &s.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) Models(ctx context.Context, f event.AnalyticsFilter) ([]event.ModelStat, error) {
	where, args := analyticsWhere(f)
	q := `SELECT provider, model, COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), SUM(cost_usd)
		FROM llm_events ` + where + `
		GROUP BY provider, model ORDER BY 3 DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.ModelStat, 0)
	for rows.Next() {
		var s event.ModelStat
		if err := rows.Scan(&s.Provider, &s.Model, &s.RequestCount, &s.InputTokens, &s.OutputTokens, &s.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) TopProjects(ctx context.Context, f event.AnalyticsFilter, limit int) ([]event.ProjectStat, error) {
	q := `SELECT e.project_id, p.name, COUNT(*)
		FROM llm_events e
		JOIN projects p ON p.id = e.project_id
		WHERE e.request_started_at BETWEEN $1 AND $2
		GROUP BY e.project_id, p.name
		ORDER BY 3 DESC
		LIMIT $3`
	rows, err := r.pool.Query(ctx, q, f.From, f.To, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]event.ProjectStat, 0, limit)
	for rows.Next() {
		var s event.ProjectStat
		if err := rows.Scan(&s.ProjectID, &s.ProjectName, &s.RequestCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
