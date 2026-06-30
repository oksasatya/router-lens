package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/apikey"
)

type APIKeyRepository struct{ pool *pgxpool.Pool }

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository { return &APIKeyRepository{pool: pool} }

var _ apikey.APIKeyRepository = (*APIKeyRepository)(nil)

func (r *APIKeyRepository) Create(ctx context.Context, k *apikey.APIKey) error {
	const q = `INSERT INTO api_keys (project_id, name, key_hash, key_prefix)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, q, k.ProjectID, k.Name, k.KeyHash, k.KeyPrefix).
		Scan(&k.ID, &k.CreatedAt)
}

func (r *APIKeyRepository) ListByProject(ctx context.Context, projectID string) ([]*apikey.APIKey, error) {
	const q = `SELECT id, project_id, name, key_prefix, last_used_at, revoked_at, created_at
		FROM api_keys WHERE project_id = $1 ORDER BY created_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*apikey.APIKey, 0)
	for rows.Next() {
		var k apikey.APIKey
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &k)
	}
	return out, rows.Err()
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE api_keys SET revoked_at = now() WHERE id = $1 RETURNING id`
	var got string
	err := r.pool.QueryRow(ctx, q, id).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		return apikey.ErrNotFound
	}
	return err
}
