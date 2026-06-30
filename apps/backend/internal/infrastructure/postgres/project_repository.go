package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/project"
)

type ProjectRepository struct{ pool *pgxpool.Pool }

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

var _ project.ProjectRepository = (*ProjectRepository)(nil)

const projectColumns = `id, owner_user_id, name, slug, COALESCE(description, ''), created_at, updated_at`

func (r *ProjectRepository) Create(ctx context.Context, p *project.Project) error {
	const q = `INSERT INTO projects (owner_user_id, name, slug, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, q, p.OwnerUserID, p.Name, p.Slug, p.Description).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if isUniqueViolation(err) {
		return project.ErrSlugTaken
	}
	return err
}

func (r *ProjectRepository) List(ctx context.Context, limit, offset int) ([]*project.Project, error) {
	q := `SELECT ` + projectColumns + ` FROM projects ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*project.Project, 0, limit)
	for rows.Next() {
		var p project.Project
		if err := rows.Scan(&p.ID, &p.OwnerUserID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *ProjectRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects`).Scan(&n)
	return n, err
}

func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*project.Project, error) {
	q := `SELECT ` + projectColumns + ` FROM projects WHERE id = $1`
	var p project.Project
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&p.ID, &p.OwnerUserID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, project.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) Update(ctx context.Context, p *project.Project) error {
	const q = `UPDATE projects SET name = $2, description = $3, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	err := r.pool.QueryRow(ctx, q, p.ID, p.Name, p.Description).Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return project.ErrNotFound
	}
	return err
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return project.ErrNotFound
	}
	return nil
}
