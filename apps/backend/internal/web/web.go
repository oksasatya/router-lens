// Package web serves the embedded frontend. Until Plan 07 embeds the real
// build, it returns a placeholder so the route exists.
package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func SPAHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "RouterLens — frontend not yet embedded")
	}
}
