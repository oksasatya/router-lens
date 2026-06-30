// Package apikey holds the API Key use cases. Depends on the apikey + project
// domain ports + shared security/errors (no HTTP, no SQL).
package apikey

import (
	"context"
	"errors"

	apikeydomain "router-lens/internal/domain/apikey"
	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
)

type Service struct {
	repo     apikeydomain.APIKeyRepository
	projects projectdomain.ProjectRepository
}

func NewService(repo apikeydomain.APIKeyRepository, projects projectdomain.ProjectRepository) *Service {
	return &Service{repo: repo, projects: projects}
}

// Create generates a key, returns the plaintext exactly once (never persisted),
// and stores only the hash. Verifies the project exists first.
func (s *Service) Create(ctx context.Context, projectID, name string) (plaintext string, key *apikeydomain.APIKey, err error) {
	if _, err = s.projects.FindByID(ctx, projectID); err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return "", nil, apperrors.New(apperrors.KindNotFound, i18n.CodeProjectNotFound, "project not found")
		}
		return "", nil, err
	}
	plaintext, prefix, hash, err := security.GenerateAPIKey()
	if err != nil {
		return "", nil, err
	}
	k := &apikeydomain.APIKey{ProjectID: projectID, Name: name, KeyHash: hash, KeyPrefix: prefix}
	if err = s.repo.Create(ctx, k); err != nil {
		return "", nil, err
	}
	return plaintext, k, nil
}

func (s *Service) List(ctx context.Context, projectID string) ([]*apikeydomain.APIKey, error) {
	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return nil, apperrors.New(apperrors.KindNotFound, i18n.CodeProjectNotFound, "project not found")
		}
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	if err := s.repo.Revoke(ctx, id); err != nil {
		if errors.Is(err, apikeydomain.ErrNotFound) {
			return apperrors.New(apperrors.KindNotFound, i18n.CodeAPIKeyNotFound, "api key not found")
		}
		return err
	}
	return nil
}
