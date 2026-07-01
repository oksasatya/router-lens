// Package auth holds the authentication use cases. It depends only on domain
// ports + the security/errors shared packages (no HTTP, no SQL).
package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"router-lens/internal/domain/user"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
)

const SessionTTL = 7 * 24 * time.Hour

// dummyHash is used to perform constant-time work when email is not found,
// preventing timing-based email enumeration.
var (
	dummyHash     string
	dummyHashOnce sync.Once
)

type Service struct {
	users    user.UserRepository
	sessions user.SessionRepository
}

func NewService(users user.UserRepository, sessions user.SessionRepository) *Service {
	return &Service{users: users, sessions: sessions}
}

func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	exists, err := s.users.AnyExists(ctx)
	return !exists, err
}

// Setup creates the single admin. Returns 403 setup_locked once any user exists.
// AnyExists is checked before hashing to avoid argon2 DoS on locked instances.
func (s *Service) Setup(ctx context.Context, email, password, name string) error {
	exists, err := s.users.AnyExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return apperrors.New(apperrors.KindForbidden, i18n.CodeAuthSetupLocked, "setup already completed")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	created, err := s.users.CreateInitialAdmin(ctx, &user.User{Email: email, PasswordHash: hash, Name: name})
	if err != nil {
		return err
	}
	if !created {
		// Concurrent setup won the race; CreateInitialAdmin already serialised via advisory lock.
		return apperrors.New(apperrors.KindForbidden, i18n.CodeAuthSetupLocked, "setup already completed")
	}
	return nil
}

// Login verifies credentials and creates a session, returning the opaque cookie token.
func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (string, error) {
	dummyHashOnce.Do(func() {
		dummyHash, _ = security.HashPassword("router-lens-dummy")
	})
	u, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, user.ErrNotFound) {
		// Constant-time guard: do equivalent argon2 work so unknown-email vs wrong-password
		// cannot be distinguished by timing.
		_, _ = security.VerifyPassword(password, dummyHash)
		return "", s.invalidCredentials()
	}
	if err != nil {
		return "", err
	}
	ok, err := security.VerifyPassword(password, u.PasswordHash)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", s.invalidCredentials()
	}

	token, err := security.GenerateSessionToken()
	if err != nil {
		return "", err
	}
	sess := &user.Session{
		UserID:    u.ID,
		TokenHash: security.HashToken(token),
		ExpiresAt: time.Now().Add(SessionTTL),
		UserAgent: userAgent,
		IP:        ip,
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) Logout(ctx context.Context, tokenHash string) error {
	return s.sessions.DeleteByTokenHash(ctx, tokenHash)
}

// UpdateProfile changes the admin's display name.
func (s *Service) UpdateProfile(ctx context.Context, userID, name string) (*user.User, error) {
	if err := s.users.UpdateName(ctx, userID, name); err != nil {
		return nil, err
	}
	return s.users.FindByID(ctx, userID)
}

// ChangePassword verifies currentPassword, rotates the password hash, and revokes every
// other active session — the session identified by currentSessionTokenHash (the one
// making this request) is left alone. This closes the "leaked session cookie" case: the
// moment the admin changes their password, any other session stops working.
func (s *Service) ChangePassword(ctx context.Context, userID, currentSessionTokenHash, currentPassword, newPassword string) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	ok, err := security.VerifyPassword(currentPassword, u.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.New(apperrors.KindValidation, i18n.CodeAuthInvalidCurrentPassword, "current password is incorrect")
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}
	return s.sessions.DeleteByUserIDExceptTokenHash(ctx, userID, currentSessionTokenHash)
}

func (s *Service) invalidCredentials() error {
	return apperrors.New(apperrors.KindUnauthorized, i18n.CodeAuthInvalidCredentials, "invalid email or password")
}
