package dto

import (
	eventdomain "router-lens/internal/domain/event"
	eventapp "router-lens/internal/usecase/event"
)

// OverviewResponse is the wire shape of GET /analytics/overview.
type OverviewResponse struct {
	TotalRequests      int64         `json:"total_requests"`
	TotalInputTokens   int64         `json:"total_input_tokens"`
	TotalOutputTokens  int64         `json:"total_output_tokens"`
	TotalCostUSD       *string       `json:"total_cost_usd"`
	UnpricedCount      int64         `json:"unpriced_count"`
	AvgLatencyMs       *float64      `json:"avg_latency_ms"`
	P95LatencyMs       *float64      `json:"p95_latency_ms"`
	ErrorCount         int64         `json:"error_count"`
	ErrorRate          float64       `json:"error_rate"`
	MostUsedProvider   string        `json:"most_used_provider"`
	MostUsedModel      string        `json:"most_used_model"`
	MostExpensiveModel string        `json:"most_expensive_model"`
	TopProjects        []ProjectStat `json:"top_projects"`
}

type ProjectStat struct {
	ProjectID    string `json:"project_id"`
	ProjectName  string `json:"project_name"`
	RequestCount int64  `json:"request_count"`
}

func FromOverviewResult(r *eventapp.OverviewResult) OverviewResponse {
	top := make([]ProjectStat, 0, len(r.TopProjects))
	for _, p := range r.TopProjects {
		top = append(top, ProjectStat{ProjectID: p.ProjectID, ProjectName: p.ProjectName, RequestCount: p.RequestCount})
	}
	return OverviewResponse{
		TotalRequests: r.Totals.TotalRequests, TotalInputTokens: r.Totals.TotalInputTokens,
		TotalOutputTokens: r.Totals.TotalOutputTokens, TotalCostUSD: decimalPtrString(r.Totals.TotalCostUSD),
		UnpricedCount: r.Totals.UnpricedCount, AvgLatencyMs: r.Totals.AvgLatencyMs, P95LatencyMs: r.Totals.P95LatencyMs,
		ErrorCount: r.Totals.ErrorCount, ErrorRate: r.ErrorRate,
		MostUsedProvider: r.MostUsedProvider, MostUsedModel: r.MostUsedModel, MostExpensiveModel: r.MostExpensiveModel,
		TopProjects: top,
	}
}

type TokenPointResponse struct {
	Bucket       string `json:"bucket"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

func FromTokenPoints(points []eventapp.TokenPoint) []TokenPointResponse {
	out := make([]TokenPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, TokenPointResponse{Bucket: p.Bucket, InputTokens: p.InputTokens, OutputTokens: p.OutputTokens})
	}
	return out
}

type CostPointResponse struct {
	Bucket  string  `json:"bucket"`
	CostUSD *string `json:"cost_usd"`
}

func FromCostPoints(points []eventapp.CostPoint) []CostPointResponse {
	out := make([]CostPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, CostPointResponse{Bucket: p.Bucket, CostUSD: decimalPtrString(p.CostUSD)})
	}
	return out
}

type LatencyPointResponse struct {
	Bucket       string   `json:"bucket"`
	AvgLatencyMs *float64 `json:"avg_latency_ms"`
	P95LatencyMs *float64 `json:"p95_latency_ms"`
}

func FromLatencyPoints(points []eventapp.LatencyPoint) []LatencyPointResponse {
	out := make([]LatencyPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, LatencyPointResponse{Bucket: p.Bucket, AvgLatencyMs: p.AvgLatencyMs, P95LatencyMs: p.P95LatencyMs})
	}
	return out
}

type ErrorPointResponse struct {
	Bucket       string  `json:"bucket"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
}

func FromErrorPoints(points []eventapp.ErrorPoint) []ErrorPointResponse {
	out := make([]ErrorPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, ErrorPointResponse{Bucket: p.Bucket, RequestCount: p.RequestCount, ErrorCount: p.ErrorCount, ErrorRate: p.ErrorRate})
	}
	return out
}

type ProviderStatResponse struct {
	Provider     string  `json:"provider"`
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      *string `json:"cost_usd"`
}

func FromProviderStats(stats []eventdomain.ProviderStat) []ProviderStatResponse {
	out := make([]ProviderStatResponse, 0, len(stats))
	for _, s := range stats {
		out = append(out, ProviderStatResponse{
			Provider: s.Provider, RequestCount: s.RequestCount,
			InputTokens: s.InputTokens, OutputTokens: s.OutputTokens, CostUSD: decimalPtrString(s.CostUSD),
		})
	}
	return out
}

type ModelStatResponse struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      *string `json:"cost_usd"`
}

func FromModelStats(stats []eventdomain.ModelStat) []ModelStatResponse {
	out := make([]ModelStatResponse, 0, len(stats))
	for _, s := range stats {
		out = append(out, ModelStatResponse{
			Provider: s.Provider, Model: s.Model, RequestCount: s.RequestCount,
			InputTokens: s.InputTokens, OutputTokens: s.OutputTokens, CostUSD: decimalPtrString(s.CostUSD),
		})
	}
	return out
}
