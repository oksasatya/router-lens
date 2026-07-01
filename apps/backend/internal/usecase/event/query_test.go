package event

import (
	"context"
	"testing"

	eventdomain "router-lens/internal/domain/event"
	apperrors "router-lens/internal/shared/errors"
)

type notFoundRepo struct{ fakeEventRepo }

func (*notFoundRepo) FindByID(context.Context, string) (*eventdomain.Event, error) {
	return nil, eventdomain.ErrNotFound
}

func TestGetNotFound(t *testing.T) {
	_, err := NewQueryService(&notFoundRepo{}).Get(context.Background(), "x")
	ae, ok := apperrors.As(err)
	if !ok || ae.Kind != apperrors.KindNotFound {
		t.Fatalf("want not_found AppError, got %v", err)
	}
}

func TestListProbesOneExtraForHasMore(t *testing.T) {
	// repo returns 3 rows for a requested limit of 2 -> hasMore true, 2 returned.
	repo := &limitProbeRepo{rows: 3}
	events, hasMore, err := NewQueryService(repo).List(context.Background(), eventdomain.Filter{Limit: 2})
	if err != nil || !hasMore || len(events) != 2 {
		t.Fatalf("hasMore=%v len=%d err=%v", hasMore, len(events), err)
	}
	if repo.askedLimit != 3 {
		t.Fatalf("expected probe limit 3, got %d", repo.askedLimit)
	}
}

type limitProbeRepo struct {
	fakeEventRepo
	rows       int
	askedLimit int
}

func (r *limitProbeRepo) List(_ context.Context, f eventdomain.Filter) ([]*eventdomain.Event, error) {
	r.askedLimit = f.Limit
	out := make([]*eventdomain.Event, 0, r.rows)
	for i := 0; i < r.rows; i++ {
		out = append(out, &eventdomain.Event{ID: "e"})
	}
	return out, nil
}
