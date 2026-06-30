// Package http builds the Echo application.
package http

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"

	"router-lens/internal/app"
	"router-lens/internal/infrastructure/http/middleware"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/web"
)

const ingestionBodyLimit = "64KB"

// NewServer constructs the Echo instance with shared middleware, the
// /api/v1 group, and the SPA fallback for all other paths.
func NewServer(cfg app.Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Debug = !cfg.IsProduction()
	e.HTTPErrorHandler = middleware.ErrorHandler

	e.Use(emw.Recover())
	e.Use(emw.Logger())
	e.Use(emw.RequestID())
	e.Use(emw.BodyLimit(ingestionBodyLimit))
	e.Use(middleware.Lang)

	api := e.Group("/api/v1")
	RegisterHealth(api, func() bool { return true })

	// SPA fallback: serve frontend for non-API paths; return 404 for unknown /api/* paths.
	e.GET("/*", func(c echo.Context) error {
		if strings.HasPrefix(c.Request().URL.Path, "/api/") {
			return apperrors.New(apperrors.KindNotFound, "not_found", "route not found")
		}
		return web.SPAHandler()(c)
	})
	return e
}

// RegisterHealth mounts liveness and readiness probes.
func RegisterHealth(g *echo.Group, ready func() bool) {
	g.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	g.GET("/readyz", func(c echo.Context) error {
		if !ready() {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
}
