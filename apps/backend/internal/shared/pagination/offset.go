// Package pagination provides offset pagination for CRUD lists and keyset
// (cursor) pagination for the high-volume events list.
package pagination

import "strconv"

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Offset holds the parsed page and limit for SQL offset pagination.
type Offset struct {
	Page  int
	Limit int
}

// ParseOffset parses page and limit from query-string strings.
// Defaults: page=1, limit=20. Limit is clamped to [1, 100].
// Invalid or out-of-range values fall back to defaults.
func ParseOffset(page, limit string) Offset {
	p, err := strconv.Atoi(page)
	if err != nil || p < 1 {
		p = 1
	}
	l, err := strconv.Atoi(limit)
	if err != nil || l < 1 {
		l = defaultLimit
	}
	if l > maxLimit {
		l = maxLimit
	}
	return Offset{Page: p, Limit: l}
}

// SQLOffset returns the OFFSET value for a SQL query: (Page-1)*Limit.
func (o Offset) SQLOffset() int { return (o.Page - 1) * o.Limit }
