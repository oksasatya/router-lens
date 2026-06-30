package project

import "context"

// ProjectRepository is the port for persisting and querying Project aggregates.
type ProjectRepository interface {
	// Create inserts p, setting ID/CreatedAt/UpdatedAt. Returns ErrSlugTaken on
	// a (owner_user_id, slug) unique violation.
	Create(ctx context.Context, p *Project) error
	List(ctx context.Context, limit, offset int) ([]*Project, error)
	Count(ctx context.Context) (int, error)
	FindByID(ctx context.Context, id string) (*Project, error)
	// Update changes name + description (slug is immutable). Returns ErrNotFound.
	Update(ctx context.Context, p *Project) error
	// Delete removes the row. Returns ErrNotFound when no row matched.
	Delete(ctx context.Context, id string) error
}
