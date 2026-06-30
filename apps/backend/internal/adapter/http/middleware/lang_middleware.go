package middleware

import (
	"github.com/labstack/echo/v4"

	"router-lens/internal/shared/i18n"
)

// Lang resolves the request language from the Accept-Language header and stores
// it in the Echo context for response localization.
func Lang(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(i18n.ContextKey, i18n.Resolve(c.Request().Header.Get("Accept-Language")))
		return next(c)
	}
}
