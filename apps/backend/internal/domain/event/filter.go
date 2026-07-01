package event

import "time"

// Filter selects events for List/Export. All fields are optional except Limit
// (List). A zero ProjectID means "all projects" (cross-project list).
type Filter struct {
	ProjectID  string
	From       time.Time // zero = unbounded
	To         time.Time // zero = unbounded
	Provider   string
	Model      string
	IsError    *bool
	CursorTime time.Time // zero (with CursorID == "") = first page
	CursorID   string
	Limit      int
}
