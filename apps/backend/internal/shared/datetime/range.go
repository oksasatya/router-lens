// Package datetime parses the dashboard's bounded date-range filter.
package datetime

import (
	"time"

	apperrors "router-lens/internal/shared/errors"
)

const (
	defaultWindow = 24 * time.Hour
	maxWindow     = 90 * 24 * time.Hour
	codeBadRange  = "bad_date_range"
)

// Range is a resolved, bounded time window.
type Range struct {
	From time.Time
	To   time.Time
}

// presets maps label -> window duration subtracted from now.
var presets = map[string]time.Duration{
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}

// ParseRange resolves a bounded range.
// Precedence: explicit from/to > preset > default 24h.
// Presets: "24h","7d","30d". Enforces From<To and (To-From) <= 90d.
// now is injected so callers and tests control the reference point.
func ParseRange(from, to, preset string, now time.Time) (Range, error) {
	if from != "" || to != "" {
		return parseExplicit(from, to, now)
	}
	window := defaultWindow
	if d, ok := presets[preset]; ok {
		window = d
	}
	return Range{From: now.Add(-window), To: now}, nil
}

// parseExplicit handles the case where at least one explicit bound is provided.
func parseExplicit(from, to string, now time.Time) (Range, error) {
	f, err := parseOrFallback(from, now.Add(-defaultWindow))
	if err != nil {
		return Range{}, err
	}
	t, err := parseOrFallback(to, now)
	if err != nil {
		return Range{}, err
	}
	if !f.Before(t) {
		return Range{}, apperrors.New(apperrors.KindValidation, codeBadRange, "from must be before to")
	}
	if t.Sub(f) > maxWindow {
		return Range{}, apperrors.New(apperrors.KindValidation, codeBadRange, "date range exceeds the 90-day maximum")
	}
	return Range{From: f, To: t}, nil
}

// parseOrFallback parses an RFC3339 timestamp or returns the fallback on empty input.
func parseOrFallback(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, apperrors.New(apperrors.KindValidation, codeBadRange, "invalid timestamp; use RFC3339")
	}
	return ts.UTC(), nil
}
