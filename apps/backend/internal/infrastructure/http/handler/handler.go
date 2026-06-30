package handler

import "time"

const timeLayout = time.RFC3339

// formatNullableTime renders a *time.Time as a UTC RFC3339 *string, or nil.
func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(timeLayout)
	return &s
}
