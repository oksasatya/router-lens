package project

import (
	"context"
	"errors"
	"testing"

	projectdomain "router-lens/internal/domain/project"
	apperrors "router-lens/internal/shared/errors"
)

type fakeRepo struct {
	createErr error
	findErr   error
	updateErr error
	deleteErr error
	got       *projectdomain.Project
}

func (f *fakeRepo) Create(_ context.Context, p *projectdomain.Project) error {
	f.got = p
	if f.createErr != nil {
		return f.createErr
	}
	p.ID = "p1"
	return nil
}
func (f *fakeRepo) List(context.Context, int, int) ([]*projectdomain.Project, error) {
	return []*projectdomain.Project{{ID: "p1"}}, nil
}
func (f *fakeRepo) Count(context.Context) (int, error) { return 1, nil }
func (f *fakeRepo) FindByID(context.Context, string) (*projectdomain.Project, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return &projectdomain.Project{ID: "p1", Name: "x"}, nil
}
func (f *fakeRepo) Update(_ context.Context, p *projectdomain.Project) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.got = p
	return nil
}
func (f *fakeRepo) Delete(context.Context, string) error { return f.deleteErr }

func TestCreate(t *testing.T) {
	t.Run("derives slug and stamps owner", func(t *testing.T) {
		f := &fakeRepo{}
		s := NewService(f)
		p, err := s.Create(context.Background(), "owner1", "My App", "desc")
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if p.Slug != "my-app" || f.got.OwnerUserID != "owner1" {
			t.Fatalf("slug=%q owner=%q", p.Slug, f.got.OwnerUserID)
		}
	})
	t.Run("maps slug collision to conflict AppError", func(t *testing.T) {
		f := &fakeRepo{createErr: projectdomain.ErrSlugTaken}
		_, err := NewService(f).Create(context.Background(), "o", "n", "")
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindConflict {
			t.Fatalf("want conflict AppError, got %v", err)
		}
	})
}

func TestGetNotFound(t *testing.T) {
	f := &fakeRepo{findErr: projectdomain.ErrNotFound}
	_, err := NewService(f).Get(context.Background(), "missing")
	ae, ok := apperrors.As(err)
	if !ok || ae.Kind != apperrors.KindNotFound {
		t.Fatalf("want not_found AppError, got %v", err)
	}
}

func TestDeleteUnknownErrorPropagates(t *testing.T) {
	sentinel := errors.New("db down")
	f := &fakeRepo{deleteErr: sentinel}
	err := NewService(f).Delete(context.Background(), "p1")
	if !errors.Is(err, sentinel) {
		t.Fatalf("want raw error propagated, got %v", err)
	}
}
