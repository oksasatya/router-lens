package auth

import (
	"context"
	"testing"

	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"

	"router-lens/internal/domain/user"
)

// fakeUsers / fakeSessions are minimal in-memory repos.
type fakeUsers struct {
	byEmail map[string]*user.User
	created bool
}

func (f *fakeUsers) CreateInitialAdmin(_ context.Context, u *user.User) (bool, error) {
	if f.created {
		return false, nil
	}
	f.created = true
	u.ID = "u1"
	if f.byEmail == nil {
		f.byEmail = map[string]*user.User{}
	}
	f.byEmail[u.Email] = u
	return true, nil
}
func (f *fakeUsers) FindByEmail(_ context.Context, e string) (*user.User, error) {
	if u, ok := f.byEmail[e]; ok {
		return u, nil
	}
	return nil, user.ErrNotFound
}
func (f *fakeUsers) FindByID(context.Context, string) (*user.User, error) {
	return nil, user.ErrNotFound
}
func (f *fakeUsers) AnyExists(context.Context) (bool, error) { return f.created, nil }

type fakeSessions struct{ saved *user.Session }

func (f *fakeSessions) Create(_ context.Context, s *user.Session) error { f.saved = s; return nil }
func (f *fakeSessions) FindByTokenHash(context.Context, string) (*user.Session, error) {
	return nil, user.ErrNotFound
}
func (f *fakeSessions) DeleteByTokenHash(context.Context, string) error { return nil }

const testPassword = "password123"

func TestAuthService(t *testing.T) {
	ctx := context.Background()

	t.Run("setup creates then locks", func(t *testing.T) {
		svc := NewService(&fakeUsers{}, &fakeSessions{})
		if err := svc.Setup(ctx, "a@b.com", testPassword, "Admin"); err != nil {
			t.Fatalf("first setup: %v", err)
		}
		err := svc.Setup(ctx, "c@d.com", testPassword, "Two")
		ae, ok := apperrors.As(err)
		if !ok || ae.Code != i18n.CodeAuthSetupLocked {
			t.Fatalf("second setup should be locked, got %v", err)
		}
	})

	t.Run("login succeeds and stores session", func(t *testing.T) {
		hash, err := security.HashPassword(testPassword)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		fs := &fakeSessions{}
		svc := NewService(fu, fs)
		tok, err := svc.Login(ctx, "a@b.com", testPassword, "agent", "127.0.0.1")
		if err != nil || tok == "" {
			t.Fatalf("login: tok=%q err=%v", tok, err)
		}
		if fs.saved == nil || fs.saved.TokenHash != security.HashToken(tok) {
			t.Fatal("session must be stored with the token hash")
		}
	})

	t.Run("login rejects wrong password and unknown email identically", func(t *testing.T) {
		hash, err := security.HashPassword(testPassword)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		svc := NewService(fu, &fakeSessions{})
		for _, c := range []struct{ email, pw string }{{"a@b.com", "wrong"}, {"nobody@x.com", testPassword}} {
			_, err := svc.Login(ctx, c.email, c.pw, "", "")
			ae, ok := apperrors.As(err)
			if !ok || ae.Code != i18n.CodeAuthInvalidCredentials {
				t.Fatalf("expected invalid_credentials for %v, got %v", c, err)
			}
		}
	})
}
