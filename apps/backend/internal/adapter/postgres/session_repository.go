package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/user"
)

type SessionRepository struct{ pool *pgxpool.Pool }

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

var _ user.SessionRepository = (*SessionRepository)(nil)

func (r *SessionRepository) Create(ctx context.Context, s *user.Session) error {
	const q = `
		INSERT INTO sessions (user_id, token_hash, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::inet)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, q, s.UserID, s.TokenHash, s.ExpiresAt, s.UserAgent, s.IP).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*user.Session, error) {
	var s user.Session
	const q = `SELECT id, user_id, token_hash, expires_at, created_at
		FROM sessions WHERE token_hash = $1`
	err := r.pool.QueryRow(ctx, q, tokenHash).
		Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}
