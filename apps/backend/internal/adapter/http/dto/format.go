// Package dto holds the HTTP wire shapes (request/response structs with json +
// validate tags) and the domain→response mappers. It is an adapter-layer
// concern: it imports only the domain (for mapping) and is consumed by the
// http/handler package. Domain and application stay free of transport types.
package dto

import "time"

const timeLayout = time.RFC3339

// formatNullableTime renders a *time.Time as a UTC RFC3339 *string, or nil.
func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return new(t.UTC().Format(timeLayout))
}
