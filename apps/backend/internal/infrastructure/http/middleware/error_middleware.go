// Package middleware holds Echo middleware shared across routes.
package middleware

import (
	"github.com/labstack/echo/v4"

	"router-lens/internal/shared/response"
)

// ErrorHandler is Echo's central HTTPErrorHandler: it routes every handler
// error through the canonical response envelope.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	_ = response.Error(c, err)
}
