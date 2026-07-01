package middleware

import (
	"errors"
	"strings"

	"github.com/labstack/echo/v4"

	"router-lens/internal/domain/apikey"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/security"
)

// ContextProjectIDKey holds the project id resolved from the API key.
const ContextProjectIDKey = "ingest_project_id"

const bearerPrefix = "Bearer "

// APIKey authenticates an ingestion request from a Bearer API key: resolve by
// hash, reject if missing or revoked, best-effort touch last_used_at, inject the
// project id. SECOND auth boundary — never combined with the session middleware.
func APIKey(keys apikey.APIKeyRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			k, err := resolveKey(c, keys)
			if err != nil {
				return err
			}
			_ = keys.TouchLastUsed(c.Request().Context(), k.ID) // best-effort; never blocks ingestion
			c.Set(ContextProjectIDKey, k.ProjectID)
			return next(c)
		}
	}
}

// resolveKey extracts the Bearer key, hashes it, looks it up, and enforces the
// revoked rule. Extracted to keep the outer closure under S3776.
func resolveKey(c echo.Context, keys apikey.APIKeyRepository) (*apikey.APIKey, error) {
	header := c.Request().Header.Get(echo.HeaderAuthorization)
	if !strings.HasPrefix(header, bearerPrefix) {
		return nil, unauthorizedKey()
	}
	plaintext := strings.TrimPrefix(header, bearerPrefix)
	if plaintext == "" {
		return nil, unauthorizedKey()
	}
	k, err := keys.FindByHash(c.Request().Context(), security.HashAPIKey(plaintext))
	if errors.Is(err, apikey.ErrNotFound) {
		return nil, unauthorizedKey()
	}
	if err != nil {
		return nil, err
	}
	if k.IsRevoked() {
		return nil, unauthorizedKey()
	}
	return k, nil
}

func unauthorizedKey() error {
	return apperrors.New(apperrors.KindUnauthorized, i18n.CodeUnauthorized, "invalid or revoked API key")
}

// CurrentProjectID returns the project id injected by the APIKey middleware.
func CurrentProjectID(c echo.Context) string {
	if id, ok := c.Get(ContextProjectIDKey).(string); ok {
		return id
	}
	return ""
}
