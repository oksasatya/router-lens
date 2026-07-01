package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/event"
)

type EventRepository struct{ pool *pgxpool.Pool }

func NewEventRepository(pool *pgxpool.Pool) *EventRepository { return &EventRepository{pool: pool} }

var _ event.EventRepository = (*EventRepository)(nil)

// eventColumns is the full select/return column list (kept once — S1192).
// event_id is COALESCEd to "" because Event.EventID is a plain (non-pointer)
// string and a NULL event_id (empty at insert, see nullEventID) would
// otherwise fail the scan.
const eventColumns = `id, project_id, COALESCE(event_id, '') AS event_id, provider, model, route_source, agent,
	input_tokens, output_tokens, cost_usd, input_price_1m, output_price_1m,
	latency_ms, status_code, is_error, error_message,
	request_started_at, request_finished_at, received_at, metadata, created_at`

// nullEventID maps "" to a NULL event_id so the partial idempotency index does
// not treat blank keys as collidable.
func nullEventID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// metadataArg passes JSON to the jsonb column as TEXT. pgx encodes a Go string
// into jsonb correctly; a raw []byte would be sent as bytea (wrong type). NULL
// when empty.
func metadataArg(m []byte) any {
	if len(m) == 0 {
		return nil
	}
	return string(m)
}

func (r *EventRepository) Insert(ctx context.Context, e *event.Event) (bool, error) {
	const q = `INSERT INTO llm_events (
			project_id, event_id, provider, model, route_source, agent,
			input_tokens, output_tokens, cost_usd, input_price_1m, output_price_1m,
			latency_ms, status_code, is_error, error_message,
			request_started_at, request_finished_at, received_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		ON CONFLICT (project_id, event_id) WHERE event_id IS NOT NULL DO NOTHING
		RETURNING id`
	err := r.pool.QueryRow(ctx, q,
		e.ProjectID, nullEventID(e.EventID), e.Provider, e.Model, e.RouteSource, e.Agent,
		e.InputTokens, e.OutputTokens, e.CostUSD, e.InputPrice1M, e.OutputPrice1M,
		e.LatencyMs, e.StatusCode, e.IsError, e.ErrorMessage,
		e.RequestStartedAt, e.RequestFinishedAt, e.ReceivedAt, metadataArg(e.Metadata),
	).Scan(&e.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // duplicate: ON CONFLICT DO NOTHING returned no row
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// scanEvent reads one row in eventColumns order. Shared by List/FindByID/Export.
func scanEvent(row pgx.Row, e *event.Event) error {
	return row.Scan(
		&e.ID, &e.ProjectID, &e.EventID, &e.Provider, &e.Model, &e.RouteSource, &e.Agent,
		&e.InputTokens, &e.OutputTokens, &e.CostUSD, &e.InputPrice1M, &e.OutputPrice1M,
		&e.LatencyMs, &e.StatusCode, &e.IsError, &e.ErrorMessage,
		&e.RequestStartedAt, &e.RequestFinishedAt, &e.ReceivedAt, &e.Metadata, &e.CreatedAt,
	)
}

func (r *EventRepository) List(ctx context.Context, f event.Filter) ([]*event.Event, error) {
	where, args := buildEventWhere(f)
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit)
	q := `SELECT ` + eventColumns + ` FROM llm_events ` + where +
		` ORDER BY request_started_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*event.Event, 0, limit)
	for rows.Next() {
		var e event.Event
		if err := scanEvent(rows, &e); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *EventRepository) FindByID(ctx context.Context, id string) (*event.Event, error) {
	q := `SELECT ` + eventColumns + ` FROM llm_events WHERE id = $1`
	var e event.Event
	err := scanEvent(r.pool.QueryRow(ctx, q, id), &e)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, event.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepository) Export(ctx context.Context, f event.Filter, fn func(*event.Event) error) error {
	where, args := buildEventWhere(f)
	q := `SELECT ` + eventColumns + ` FROM llm_events ` + where +
		` ORDER BY request_started_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e event.Event
		if err := scanEvent(rows, &e); err != nil {
			return err
		}
		if err := fn(&e); err != nil {
			return err
		}
	}
	return rows.Err()
}
