package event

import (
	"encoding/json"
	"time"
)

// Sane upper bounds for a single observed call.
const (
	maxTokens        = 100_000_000 // 100M tokens per call is already absurd
	minStatusCode    = 100
	maxStatusCode    = 599
	maxMetadataBytes = 8 * 1024 // ~8 KB JSONB cap
	maxStringLen     = 256      // provider/model/agent/route_source
)

// Validation error codes. These string literals MUST match the i18n catalog
// keys added in Task 2 — the domain stays i18n-free (no import), so the strings
// live here and are mirrored, by value, in shared/i18n. This is the one accepted
// literal duplication (domain purity vs S1192).
const (
	codeInvalidTokens    = "event.invalid_tokens"
	codeInvalidLatency   = "event.invalid_latency"
	codeInvalidStatus    = "event.invalid_status"
	codeFutureTimestamp  = "event.future_timestamp"
	codeBackdateExceeded = "event.backdate_exceeded"
	codeStringTooLong    = "event.string_too_long"
	codeMetadataTooLarge = "event.metadata_too_large"
)

// IngestInput is the validated ingest command (a params object — keeps the use
// case under S107 and is the unit the validator operates on). It carries no
// project_id: that comes from the authenticated API key (decision 5).
type IngestInput struct {
	EventID           string
	Provider          string
	Model             string
	RouteSource       string
	Agent             string
	InputTokens       int64
	OutputTokens      int64
	LatencyMs         *int
	StatusCode        *int
	ErrorMessage      string
	RequestStartedAt  time.Time
	RequestFinishedAt *time.Time
	Metadata          json.RawMessage
}

// validationError is a typed boundary error carrying an i18n code. The
// application layer maps it to a KindValidation AppError (via errors.As on the
// Code() method), so the domain owns the rules without importing shared/errors.
// The field is unexported (`code`) to avoid a field/method name collision with
// the Code() accessor.
type validationError struct {
	code string
	msg  string
}

func (e validationError) Error() string { return e.msg }

// Code exposes the i18n code for the application-layer mapper.
func (e validationError) Code() string { return e.code }

// Validate enforces the ingest boundary rules. now + maxBackdate are injected so
// callers/tests control the reference point. Returns a validationError (use
// errors.As against `interface{ Code() string }`) on the first violated rule.
func Validate(in IngestInput, now time.Time, maxBackdate time.Duration) error {
	if err := validateTokens(in); err != nil {
		return err
	}
	if err := validateLatencyStatus(in); err != nil {
		return err
	}
	if err := validateTimestamps(in, now, maxBackdate); err != nil {
		return err
	}
	return validateSizes(in)
}

func validateTokens(in IngestInput) error {
	if in.InputTokens < 0 || in.OutputTokens < 0 || in.InputTokens > maxTokens || in.OutputTokens > maxTokens {
		return validationError{codeInvalidTokens, "token counts must be between 0 and the maximum"}
	}
	return nil
}

func validateLatencyStatus(in IngestInput) error {
	if in.LatencyMs != nil && *in.LatencyMs < 0 {
		return validationError{codeInvalidLatency, "latency_ms must not be negative"}
	}
	if in.StatusCode != nil && (*in.StatusCode < minStatusCode || *in.StatusCode > maxStatusCode) {
		return validationError{codeInvalidStatus, "status_code must be between 100 and 599"}
	}
	return nil
}

func validateTimestamps(in IngestInput, now time.Time, maxBackdate time.Duration) error {
	if in.RequestStartedAt.After(now) {
		return validationError{codeFutureTimestamp, "request_started_at must not be in the future"}
	}
	if now.Sub(in.RequestStartedAt) > maxBackdate {
		return validationError{codeBackdateExceeded, "request_started_at is older than the allowed backdate window"}
	}
	return nil
}

func validateSizes(in IngestInput) error {
	for _, s := range []string{in.Provider, in.Model, in.Agent, in.RouteSource} {
		if len(s) > maxStringLen {
			return validationError{codeStringTooLong, "a provider/model/agent/route_source field is too long"}
		}
	}
	if len(in.Metadata) > maxMetadataBytes {
		return validationError{codeMetadataTooLarge, "metadata exceeds the 8KB limit"}
	}
	return nil
}
