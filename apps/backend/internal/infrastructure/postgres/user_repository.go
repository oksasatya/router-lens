package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/user"
)

type UserRepository struct{ pool *pgxpool.Pool }

func NewUserRepository(pool *pgxpool.Pool) *UserRepository { return &UserRepository{pool: pool} }

var _ user.UserRepository = (*UserRepository)(nil)

// CreateInitialAdmin inserts the admin only when no user exists yet (race-safe).
// An advisory lock serialises concurrent setup attempts under READ COMMITTED.
func (r *UserRepository) CreateInitialAdmin(ctx context.Context, u *user.User) (bool, error) {
	const q = `
		INSERT INTO users (email, password_hash, name)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM users)
		RETURNING id, created_at, updated_at`
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(8723461)); err != nil {
		return false, err
	}
	err = tx.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Name).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
		return false, nil // a user already exists
	}
	if err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.scanOne(ctx, `SELECT id, email, password_hash, name, created_at, updated_at
		FROM users WHERE email = $1`, email)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	return r.scanOne(ctx, `SELECT id, email, password_hash, name, created_at, updated_at
		FROM users WHERE id = $1`, id)
}

func (r *UserRepository) AnyExists(ctx context.Context) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&exists)
	return exists, err
}

func (r *UserRepository) scanOne(ctx context.Context, q string, arg any) (*user.User, error) {
	var u user.User
	err := r.pool.QueryRow(ctx, q, arg).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
