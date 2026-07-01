package event

import "context"

// EventRepository persists and queries observed events. Events are immutable —
// no Update/Delete.
type EventRepository interface {
	// Insert stores e idempotently. inserted=false means a row with the same
	// (project_id, event_id) already existed (event_id non-empty).
	Insert(ctx context.Context, e *Event) (inserted bool, err error)
	// List returns events matching f, newest first, using keyset pagination.
	List(ctx context.Context, f Filter) ([]*Event, error)
	FindByID(ctx context.Context, id string) (*Event, error)
	// Export streams events matching f (range-bounded) to fn, newest first.
	Export(ctx context.Context, f Filter, fn func(*Event) error) error
}
