// Package event is the LLM-event bounded context: the immutable observed call,
// its ingest input + boundary validation, the query filter, and the repository
// port. Imports only stdlib + shopspring/decimal (domain purity).
package event

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// ErrNotFound is returned when an llm_events row is absent.
var ErrNotFound = errors.New("event: not found")

// Event is an immutable observed LLM call. Nil *decimal pointers mean unpriced;
// nil *int / *time mean the client omitted that optional field.
type Event struct {
	ID                string
	ProjectID         string
	EventID           string // optional client idempotency key; "" = none
	Provider          string
	Model             string
	RouteSource       string
	Agent             string
	InputTokens       int64
	OutputTokens      int64
	CostUSD           *decimal.Decimal
	InputPrice1M      *decimal.Decimal
	OutputPrice1M     *decimal.Decimal
	LatencyMs         *int
	StatusCode        *int
	IsError           bool
	ErrorMessage      string
	RequestStartedAt  time.Time
	RequestFinishedAt *time.Time
	ReceivedAt        time.Time
	Metadata          json.RawMessage // nil = none
	CreatedAt         time.Time
}

// DeriveIsError computes the stored is_error flag once at ingest: an HTTP status
// >= 400, or any non-empty error message, marks the call failed.
func DeriveIsError(statusCode *int, errorMessage string) bool {
	if errorMessage != "" {
		return true
	}
	return statusCode != nil && *statusCode >= 400
}
