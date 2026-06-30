package apikey

import (
	"context"
	"strings"
	"testing"

	apikeydomain "router-lens/internal/domain/apikey"
	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/security"
)

type fakeKeyRepo struct{ created *apikeydomain.APIKey }

func (f *fakeKeyRepo) Create(_ context.Context, k *apikeydomain.APIKey) error {
	k.ID = "k1"
	f.created = k
	return nil
}
func (f *fakeKeyRepo) ListByProject(context.Context, string) ([]*apikeydomain.APIKey, error) {
	return []*apikeydomain.APIKey{{ID: "k1", KeyPrefix: "rl_live_ab"}}, nil
}
func (f *fakeKeyRepo) Revoke(context.Context, string) error { return nil }

type fakeProjRepo struct{ exists bool }

func (f *fakeProjRepo) Create(context.Context, *projectdomain.Project) error { return nil }
func (f *fakeProjRepo) List(context.Context, int, int) ([]*projectdomain.Project, error) {
	return nil, nil
}
func (f *fakeProjRepo) Count(context.Context) (int, error) { return 0, nil }
func (f *fakeProjRepo) FindByID(context.Context, string) (*projectdomain.Project, error) {
	if f.exists {
		return &projectdomain.Project{ID: "p1"}, nil
	}
	return nil, projectdomain.ErrNotFound
}
func (f *fakeProjRepo) Update(context.Context, *projectdomain.Project) error { return nil }
func (f *fakeProjRepo) Delete(context.Context, string) error                 { return nil }

func TestCreate(t *testing.T) {
	t.Run("returns plaintext once and stores hash", func(t *testing.T) {
		kr := &fakeKeyRepo{}
		s := NewService(kr, &fakeProjRepo{exists: true})
		plaintext, k, err := s.Create(context.Background(), "p1", "ci")
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if !strings.HasPrefix(plaintext, security.APIKeyPrefix) {
			t.Fatalf("plaintext missing prefix: %q", plaintext)
		}
		if k.KeyHash == "" || k.KeyHash == plaintext {
			t.Fatal("hash must be set and differ from plaintext")
		}
		if security.HashAPIKey(plaintext) != kr.created.KeyHash {
			t.Fatal("stored hash must equal HashAPIKey(plaintext)")
		}
	})
	t.Run("unknown project -> not_found AppError", func(t *testing.T) {
		s := NewService(&fakeKeyRepo{}, &fakeProjRepo{exists: false})
		_, _, err := s.Create(context.Background(), "missing", "ci")
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindNotFound {
			t.Fatalf("want not_found AppError, got %v", err)
		}
	})
}

func TestListUnknownProject(t *testing.T) {
	s := NewService(&fakeKeyRepo{}, &fakeProjRepo{exists: false})
	_, err := s.List(context.Background(), "missing")
	ae, ok := apperrors.As(err)
	if !ok || ae.Kind != apperrors.KindNotFound {
		t.Fatalf("want not_found AppError, got %v", err)
	}
}
