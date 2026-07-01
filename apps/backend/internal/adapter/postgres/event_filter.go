package postgres

import (
	"fmt"
	"strings"

	"router-lens/internal/domain/event"
)

// buildEventWhere assembles a parameterized WHERE clause + ordered args from a
// Filter. Column names are fixed literals; only values are parameterized ($n) —
// no SQL injection surface. Returns ("", nil) when no condition applies. The
// returned args are positional: the caller appends LIMIT as the next $n.
func buildEventWhere(f event.Filter) (string, []any) {
	var conds []string
	var args []any
	add := func(tmpl string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(tmpl, len(args)))
	}
	if f.ProjectID != "" {
		add("project_id = $%d", f.ProjectID)
	}
	if !f.From.IsZero() {
		add("request_started_at >= $%d", f.From)
	}
	if !f.To.IsZero() {
		add("request_started_at <= $%d", f.To)
	}
	if f.Provider != "" {
		add("provider = $%d", f.Provider)
	}
	if f.Model != "" {
		add("model = $%d", f.Model)
	}
	if f.IsError != nil {
		add("is_error = $%d", *f.IsError)
	}
	if f.CursorID != "" && !f.CursorTime.IsZero() {
		// keyset: (request_started_at, id) strictly before the cursor, DESC order.
		args = append(args, f.CursorTime, f.CursorID)
		conds = append(conds, fmt.Sprintf("(request_started_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}
