package middleware

import (
	"errors"
	"time"

	"github.com/labstack/echo/v4"

	"router-lens/internal/domain/user"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
)

const (
	ContextUserKey    = "auth_user"
	ContextSessionKey = "auth_session"
)

// Session authenticates a request from the session cookie, loading the user and
// session into the Echo context, or returning 401.
func Session(sessions user.SessionRepository, users user.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := resolveSession(c, sessions)
			if err != nil {
				return err
			}
			u, err := users.FindByID(c.Request().Context(), sess.UserID)
			if errors.Is(err, user.ErrNotFound) {
				return unauthorized()
			}
			if err != nil {
				return err
			}
			c.Set(ContextUserKey, u)
			c.Set(ContextSessionKey, sess)
			return next(c)
		}
	}
}

// resolveSession extracts and validates the session from the cookie.
// Extracted to keep the outer closure under cog-complexity 15 (S3776).
func resolveSession(c echo.Context, sessions user.SessionRepository) (*user.Session, error) {
	cookie, err := c.Cookie(security.SessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, unauthorized()
	}
	sess, err := sessions.FindByTokenHash(c.Request().Context(), security.HashToken(cookie.Value))
	if errors.Is(err, user.ErrNotFound) {
		return nil, unauthorized()
	}
	if err != nil {
		return nil, err
	}
	if sess.IsExpired(time.Now()) {
		return nil, unauthorized()
	}
	return sess, nil
}

func unauthorized() error {
	return apperrors.New(apperrors.KindUnauthorized, i18n.CodeUnauthorized, "authentication required")
}

// CurrentUser returns the authenticated user stored in the Echo context, or nil.
func CurrentUser(c echo.Context) *user.User {
	if u, ok := c.Get(ContextUserKey).(*user.User); ok {
		return u
	}
	return nil
}

// CurrentSession returns the authenticated session stored in the Echo context, or nil.
func CurrentSession(c echo.Context) *user.Session {
	if s, ok := c.Get(ContextSessionKey).(*user.Session); ok {
		return s
	}
	return nil
}
