// Package project holds the Project CRUD use cases. Depends only on the domain
// port + shared/errors (no HTTP, no SQL).
package project

import (
	"context"
	"errors"

	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

type Service struct {
	repo projectdomain.ProjectRepository
}

func NewService(repo projectdomain.ProjectRepository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, ownerUserID, name, description string) (*projectdomain.Project, error) {
	p := &projectdomain.Project{
		OwnerUserID: ownerUserID,
		Name:        name,
		Slug:        projectdomain.Slugify(name),
		Description: description,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		if errors.Is(err, projectdomain.ErrSlugTaken) {
			return nil, apperrors.New(apperrors.KindConflict, i18n.CodeProjectSlugTaken, "a project with this name already exists")
		}
		return nil, err
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*projectdomain.Project, int, error) {
	items, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) Get(ctx context.Context, id string) (*projectdomain.Project, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapNotFound(err)
	}
	return p, nil
}

func (s *Service) Update(ctx context.Context, id, name, description string) (*projectdomain.Project, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapNotFound(err)
	}
	p.Name = name
	p.Description = description
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, s.mapNotFound(err)
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return s.mapNotFound(err)
	}
	return nil
}

// mapNotFound translates the domain ErrNotFound sentinel to a 404 AppError and
// passes any other error through unchanged.
func (s *Service) mapNotFound(err error) error {
	if errors.Is(err, projectdomain.ErrNotFound) {
		return apperrors.New(apperrors.KindNotFound, i18n.CodeProjectNotFound, "project not found")
	}
	return err
}
