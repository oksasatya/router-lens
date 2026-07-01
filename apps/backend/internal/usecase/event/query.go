package event

import (
	"context"
	"errors"

	eventdomain "router-lens/internal/domain/event"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

type QueryService struct{ events eventdomain.EventRepository }

func NewQueryService(events eventdomain.EventRepository) *QueryService {
	return &QueryService{events: events}
}

// List fetches one extra row beyond f.Limit to detect whether another page
// exists, then returns exactly f.Limit rows + hasMore. This avoids the
// off-by-one where a final page of exactly Limit rows emits a dead cursor.
func (s *QueryService) List(ctx context.Context, f eventdomain.Filter) (events []*eventdomain.Event, hasMore bool, err error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	probe := f
	probe.Limit = limit + 1
	rows, err := s.events.List(ctx, probe)
	if err != nil {
		return nil, false, err
	}
	if len(rows) > limit {
		return rows[:limit], true, nil
	}
	return rows, false, nil
}

func (s *QueryService) Get(ctx context.Context, id string) (*eventdomain.Event, error) {
	e, err := s.events.FindByID(ctx, id)
	if errors.Is(err, eventdomain.ErrNotFound) {
		return nil, apperrors.New(apperrors.KindNotFound, i18n.CodeEventNotFound, "event not found")
	}
	return e, err
}

func (s *QueryService) Export(ctx context.Context, f eventdomain.Filter, fn func(*eventdomain.Event) error) error {
	return s.events.Export(ctx, f, fn)
}
