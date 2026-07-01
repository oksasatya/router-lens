package dto

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
)

// EventIngestRequest is the wire payload for POST /events. It deliberately has
// NO project_id field — the project comes solely from the authenticated API
// key, via the request context, never the request body.
type EventIngestRequest struct {
	EventID           string          `json:"event_id" validate:"max=200"`
	Provider          string          `json:"provider" validate:"required,max=256"`
	Model             string          `json:"model" validate:"required,max=256"`
	RouteSource       string          `json:"route_source" validate:"max=256"`
	Agent             string          `json:"agent" validate:"max=256"`
	InputTokens       int64           `json:"input_tokens" validate:"gte=0"`
	OutputTokens      int64           `json:"output_tokens" validate:"gte=0"`
	LatencyMs         *int            `json:"latency_ms" validate:"omitempty,gte=0"`
	StatusCode        *int            `json:"status_code" validate:"omitempty,gte=100,lte=599"`
	ErrorMessage      string          `json:"error_message"`
	RequestStartedAt  time.Time       `json:"request_started_at" validate:"required"`
	RequestFinishedAt *time.Time      `json:"request_finished_at"`
	Metadata          json.RawMessage `json:"metadata"`
}

// ToIngestInput maps the wire payload to the domain ingest command.
func (r EventIngestRequest) ToIngestInput() eventdomain.IngestInput {
	return eventdomain.IngestInput{
		EventID: r.EventID, Provider: r.Provider, Model: r.Model,
		RouteSource: r.RouteSource, Agent: r.Agent,
		InputTokens: r.InputTokens, OutputTokens: r.OutputTokens,
		LatencyMs: r.LatencyMs, StatusCode: r.StatusCode, ErrorMessage: r.ErrorMessage,
		RequestStartedAt: r.RequestStartedAt.UTC(), RequestFinishedAt: r.RequestFinishedAt,
		Metadata: r.Metadata,
	}
}

// EventResponse is the wire shape of an observed event. Nil price/latency/
// status fields mean the client omitted them (or, for cost, the model is
// unpriced — never rendered as "0", which would be a silent lie).
type EventResponse struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	Provider          string          `json:"provider"`
	Model             string          `json:"model"`
	RouteSource       string          `json:"route_source"`
	Agent             string          `json:"agent"`
	InputTokens       int64           `json:"input_tokens"`
	OutputTokens      int64           `json:"output_tokens"`
	CostUSD           *string         `json:"cost_usd"`
	InputPrice1M      *string         `json:"input_price_1m"`
	OutputPrice1M     *string         `json:"output_price_1m"`
	LatencyMs         *int            `json:"latency_ms"`
	StatusCode        *int            `json:"status_code"`
	IsError           bool            `json:"is_error"`
	ErrorMessage      string          `json:"error_message"`
	RequestStartedAt  string          `json:"request_started_at"`
	RequestFinishedAt *string         `json:"request_finished_at"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
}

// FromEvent maps a domain event to its response shape.
func FromEvent(e *eventdomain.Event) EventResponse {
	return EventResponse{
		ID: e.ID, ProjectID: e.ProjectID, Provider: e.Provider, Model: e.Model,
		RouteSource: e.RouteSource, Agent: e.Agent,
		InputTokens: e.InputTokens, OutputTokens: e.OutputTokens,
		CostUSD: decimalPtrString(e.CostUSD), InputPrice1M: decimalPtrString(e.InputPrice1M),
		OutputPrice1M: decimalPtrString(e.OutputPrice1M),
		LatencyMs:     e.LatencyMs, StatusCode: e.StatusCode, IsError: e.IsError, ErrorMessage: e.ErrorMessage,
		RequestStartedAt:  e.RequestStartedAt.UTC().Format(timeLayout),
		RequestFinishedAt: formatNullableTime(e.RequestFinishedAt),
		Metadata:          e.Metadata,
	}
}

// decimalPtrString renders a nullable money value as a *string (nil = unpriced),
// preserving NUMERIC precision over JSON.
func decimalPtrString(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	return new(d.String())
}
