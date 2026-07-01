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
func (f *fakeUsers) FindByID(_ context.Context, id string) (*user.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}
func (f *fakeUsers) AnyExists(context.Context) (bool, error) { return f.created, nil }
func (f *fakeUsers) UpdateName(ctx context.Context, id, name string) error {
	u, err := f.FindByID(ctx, id)
	if err != nil {
		return err
	}
	u.Name = name
	return nil
}
func (f *fakeUsers) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	u, err := f.FindByID(ctx, id)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

type fakeSessions struct {
	saved         *user.Session
	deletedUserID string
	keptTokenHash string
}

func (f *fakeSessions) Create(_ context.Context, s *user.Session) error { f.saved = s; return nil }
func (f *fakeSessions) FindByTokenHash(context.Context, string) (*user.Session, error) {
	return nil, user.ErrNotFound
}
func (f *fakeSessions) DeleteByTokenHash(context.Context, string) error { return nil }
func (f *fakeSessions) DeleteByUserIDExceptTokenHash(_ context.Context, userID, keepTokenHash string) error {
	f.deletedUserID = userID
	f.keptTokenHash = keepTokenHash
	return nil
}

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

	t.Run("update profile updates the name", func(t *testing.T) {
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", Name: "Old Name"},
		}}
		svc := NewService(fu, &fakeSessions{})
		u, err := svc.UpdateProfile(ctx, "u1", "New Name")
		if err != nil {
			t.Fatalf("update profile: %v", err)
		}
		if u.Name != "New Name" {
			t.Fatalf("expected name updated, got %q", u.Name)
		}
	})

	t.Run("change password succeeds, rotates hash, revokes other sessions", func(t *testing.T) {
		hash, err := security.HashPassword(testPassword)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		fs := &fakeSessions{}
		svc := NewService(fu, fs)
		if err := svc.ChangePassword(ctx, "u1", "keep-hash", testPassword, "newpassword123"); err != nil {
			t.Fatalf("change password: %v", err)
		}
		ok, _ := security.VerifyPassword("newpassword123", fu.byEmail["a@b.com"].PasswordHash)
		if !ok {
			t.Fatal("password hash was not updated")
		}
		if fs.deletedUserID != "u1" || fs.keptTokenHash != "keep-hash" {
			t.Fatalf("expected other sessions revoked for u1 keeping keep-hash, got userID=%q keep=%q",
				fs.deletedUserID, fs.keptTokenHash)
		}
	})

	t.Run("change password rejects wrong current password without mutating state", func(t *testing.T) {
		hash, err := security.HashPassword(testPassword)
		if err != nil {
			t.Fatalf("HashPassword: %v", err)
		}
		fu := &fakeUsers{created: true, byEmail: map[string]*user.User{
			"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: hash},
		}}
		fs := &fakeSessions{}
		svc := NewService(fu, fs)
		err = svc.ChangePassword(ctx, "u1", "keep-hash", "wrong-password", "newpassword123")
		ae, ok := apperrors.As(err)
		if !ok || ae.Code != i18n.CodeAuthInvalidCurrentPassword {
			t.Fatalf("expected invalid_current_password, got %v", err)
		}
		if fu.byEmail["a@b.com"].PasswordHash != hash {
			t.Fatal("password hash must not change on wrong current password")
		}
		if fs.deletedUserID != "" {
			t.Fatal("sessions must not be revoked when the password change fails")
		}
	})
}
