// Package http builds the Echo application.
package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"

	"router-lens/internal/adapter/http/middleware"
	"router-lens/internal/platform/config"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/web"
)

const ingestionBodyLimit = "64KB"

// NewServer constructs the Echo instance with shared middleware, the
// /api/v1 group, and the SPA fallback for all other paths. The server is
// started via e.Start (Echo-native, prints the banner); platform/bootstrap
// drives its lifecycle (start/graceful-shutdown) through fx.
func NewServer(cfg config.Config, logger *slog.Logger) *echo.Echo {
	e := echo.New()
	e.Debug = !cfg.IsProduction()
	e.HTTPErrorHandler = middleware.ErrorHandler

	// Harden the underlying server (Echo's default leaves these unset → gosec
	// G112 Slowloris). e.Start reuses e.Server, so these timeouts apply.
	e.Server.ReadTimeout = 15 * time.Second
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.WriteTimeout = 30 * time.Second
	e.Server.IdleTimeout = 60 * time.Second

	e.Use(emw.Recover())
	e.Use(emw.RequestID())
	e.Use(requestLogger(logger))
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

// requestLogger emits one structured slog line per request (5xx/errors at
// ERROR level, the rest at INFO), carrying method, URI, status, latency, and
// the request ID. It replaces Echo's text logger so all output is structured.
func requestLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return emw.RequestLoggerWithConfig(emw.RequestLoggerConfig{
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogError:     true,
		LogRequestID: true,
		LogValuesFunc: func(c echo.Context, v emw.RequestLoggerValues) error {
			level := slog.LevelInfo
			if v.Status >= http.StatusInternalServerError || v.Error != nil {
				level = slog.LevelError
			}
			attrs := []slog.Attr{
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
				slog.String("request_id", v.RequestID),
			}
			if v.Error != nil {
				attrs = append(attrs, slog.String("error", v.Error.Error()))
			}
			logger.LogAttrs(c.Request().Context(), level, "request", attrs...)
			return nil
		},
	})
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
