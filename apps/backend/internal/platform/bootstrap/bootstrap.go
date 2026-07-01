// Package bootstrap is the application composition root. It wires configuration,
// adapters, and the per-bounded-context modules into a runnable fx application.
// Nothing imports bootstrap except cmd/server, so it may import every other
// package (config, usecase, domain, adapter) without creating an import cycle —
// config.Config is consumed by the adapters, so the wiring cannot live in the
// platform/config package.
package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	httpserver "router-lens/internal/adapter/http"
	"router-lens/internal/adapter/http/dto"
	"router-lens/internal/adapter/http/handler"
	"router-lens/internal/adapter/http/middleware"
	"router-lens/internal/adapter/openrouter"
	"router-lens/internal/adapter/postgres"
	apikey "router-lens/internal/domain/apikey"
	event "router-lens/internal/domain/event"
	pricing "router-lens/internal/domain/pricing"
	project "router-lens/internal/domain/project"
	"router-lens/internal/domain/user"
	"router-lens/internal/platform/config"
	"router-lens/internal/platform/logging"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/validator"
	apikeyapp "router-lens/internal/usecase/apikey"
	"router-lens/internal/usecase/auth"
	eventapp "router-lens/internal/usecase/event"
	pricingapp "router-lens/internal/usecase/pricing"
	projectapp "router-lens/internal/usecase/project"
)

const apiBasePath = "/api/v1"

// New builds the application graph. cmd/server calls New().Run(), which blocks
// until SIGINT/SIGTERM and then runs the lifecycle OnStop hooks in reverse.
func New() *fx.App {
	return fx.New(options())
}

// options is the composition root: core infrastructure + migrations first, then
// one fx.Module per bounded context, then the HTTP server last (so migrations
// have completed and every route is mounted before the listener starts).
func options() fx.Option {
	return fx.Options(
		// Route Fx's own startup logging through slog. UseLogLevel(DEBUG) puts the
		// verbose provide/invoke events at DEBUG, so they're silent at the default
		// INFO level and only appear when LOG_LEVEL=debug. Errors still log at ERROR.
		fx.WithLogger(func(l *slog.Logger) fxevent.Logger {
			fxlog := &fxevent.SlogLogger{Logger: l}
			fxlog.UseLogLevel(slog.LevelDebug)
			return fxlog
		}),
		coreModule,
		authModule,
		projectModule,
		apiKeyModule,
		pricingModule,
		eventModule,
		fx.Invoke(startServer),
	)
}

// coreModule provides config, the logger, the pgx pool, the Echo server, and
// the validator, and runs migrations before anything else starts.
var coreModule = fx.Module("core",
	fx.Provide(
		config.Load,
		provideLogger,
		providePool,
		httpserver.NewServer,
		validator.New,
	),
	fx.Invoke(runMigrations),
)

// provideLogger builds the structured logger and installs it as slog's default.
func provideLogger(cfg config.Config) *slog.Logger {
	return logging.New(cfg.IsProduction(), cfg.LogLevel)
}

var authModule = fx.Module("auth",
	fx.Provide(
		fx.Annotate(postgres.NewUserRepository, fx.As(new(user.UserRepository))),
		fx.Annotate(postgres.NewSessionRepository, fx.As(new(user.SessionRepository))),
		auth.NewService,
		handler.NewAuthHandler,
		provideSessionMiddleware,
	),
	fx.Invoke(registerAuthRoutes),
)

var projectModule = fx.Module("project",
	fx.Provide(
		fx.Annotate(postgres.NewProjectRepository, fx.As(new(project.ProjectRepository))),
		projectapp.NewService,
		handler.NewProjectHandler,
	),
	fx.Invoke(registerProjectRoutes),
)

var apiKeyModule = fx.Module("apikey",
	fx.Provide(
		fx.Annotate(postgres.NewAPIKeyRepository, fx.As(new(apikey.APIKeyRepository))),
		apikeyapp.NewService,
		handler.NewAPIKeyHandler,
	),
	fx.Invoke(registerAPIKeyRoutes),
)

var pricingModule = fx.Module("pricing",
	fx.Provide(
		fx.Annotate(postgres.NewPricingRepository, fx.As(new(pricing.PricingRepository))),
		fx.Annotate(openrouter.NewClient, fx.As(new(pricingapp.SuggestionSource))),
		pricingapp.NewService,
		handler.NewPricingHandler,
	),
	fx.Invoke(registerPricingRoutes),
)

var eventModule = fx.Module("event",
	fx.Provide(
		fx.Annotate(postgres.NewEventRepository, fx.As(new(event.EventRepository))),
		provideIngestService,
		handler.NewIngestHandler,
		eventapp.NewQueryService,
		handler.NewEventLogHandler,
		fx.Annotate(postgres.NewAnalyticsRepository, fx.As(new(event.AnalyticsRepository))),
		eventapp.NewAnalyticsService,
		handler.NewAnalyticsHandler,
	),
	fx.Invoke(registerEventIngestRoutes, registerEventLogRoutes, registerAnalyticsRoutes),
)

// provideIngestService derives the backdate window from config, keeping
// eventapp.NewIngestService itself free of a config.Config dependency
// (hexagonal — the use case doesn't know about env vars).
func provideIngestService(events event.EventRepository, prices pricing.PricingRepository, cfg config.Config) *eventapp.IngestService {
	return eventapp.NewIngestService(events, prices, time.Duration(cfg.MaxBackdateDays)*24*time.Hour)
}

// registerEventIngestRoutes mounts the ingest route behind the API-key
// middleware, built inline here since it is the middleware's only consumer —
// Fx cannot provide two distinct echo.MiddlewareFunc values without named
// results, and provideSessionMiddleware already claims that type.
func registerEventIngestRoutes(e *echo.Echo, h *handler.IngestHandler, keys apikey.APIKeyRepository) {
	h.Register(e.Group(apiBasePath), middleware.APIKey(keys))
}

// registerEventLogRoutes mounts the session-authenticated read routes.
func registerEventLogRoutes(e *echo.Echo, h *handler.EventLogHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group(apiBasePath), session)
}

// registerAnalyticsRoutes mounts the analytics routes behind the session middleware.
func registerAnalyticsRoutes(e *echo.Echo, h *handler.AnalyticsHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group(apiBasePath), session)
}

// providePool opens the pool and ties its lifetime to the fx lifecycle.
func providePool(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { pool.Close(); return nil }})
	return pool, nil
}

// runMigrations applies migrations during startup, before the server listens.
// The *slog.Logger dependency forces the logger to be built first.
func runMigrations(logger *slog.Logger, pool *pgxpool.Pool) error {
	logger.Info("applying database migrations")
	return postgres.Migrate(context.Background(), pool)
}

// startServer drives Echo's own server through the fx lifecycle: e.Start (which
// prints the Echo banner and reuses e.Server's hardened timeouts) on start, and
// e.Shutdown for a graceful drain on stop. fx.Run handles SIGINT/SIGTERM; fx is
// the DI + lifecycle coordinator, Echo owns the server.
func startServer(lc fx.Lifecycle, cfg config.Config, e *echo.Echo, logger *slog.Logger) {
	addr := ":" + cfg.AppPort
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logRoutes(e, logger)
			go func() {
				// e.Start blocks until shutdown; ErrServerClosed is the clean stop.
				if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("http server failed", "error", err)
					os.Exit(1)
				}
			}()
			logger.Info("server listening", "addr", addr, "env", cfg.AppEnv)
			return nil
		},
		OnStop: func(ctx context.Context) error { return e.Shutdown(ctx) },
	})
}

// logRoutes prints the registered route table (method + path) at startup, sorted
// by path for stable output. INFO level so it shows without the verbose Fx graph.
func logRoutes(e *echo.Echo, logger *slog.Logger) {
	routes := e.Routes()
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	logger.Info("routes registered", "count", len(routes))
	for _, r := range routes {
		logger.Info("route", "method", r.Method, "path", r.Path)
	}
}

// provideSessionMiddleware builds the one shared session-auth middleware.
func provideSessionMiddleware(sessions user.SessionRepository, users user.UserRepository) echo.MiddlewareFunc {
	return middleware.Session(sessions, users)
}

// registerAuthRoutes mounts the setup/auth routes on the shared Echo, behind
// the session middleware where required.
func registerAuthRoutes(e *echo.Echo, h *handler.AuthHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group(apiBasePath), session)
}

// registerProjectRoutes mounts the project routes behind the session middleware.
func registerProjectRoutes(e *echo.Echo, h *handler.ProjectHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group(apiBasePath), session)
}

// registerAPIKeyRoutes mounts the api-key routes behind the session middleware.
func registerAPIKeyRoutes(e *echo.Echo, h *handler.APIKeyHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group(apiBasePath), session)
}

// registerPricingRoutes mounts the pricing routes behind the session middleware.
func registerPricingRoutes(e *echo.Echo, h *handler.PricingHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group(apiBasePath), session)
}

// MigrateAndExit is the non-Fx path for the `-migrate-only` flag: load config,
// open a pool, apply migrations, and return.
func MigrateAndExit() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logging.New(cfg.IsProduction(), cfg.LogLevel)
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return postgres.Migrate(context.Background(), pool)
}

// CreateAdminAndExit is the non-Fx path for the `-create-admin` flag: load
// config, open a pool, and call the SAME auth.Service.Setup the HTTP setup
// wizard uses — this is a second caller of that use case, not a second
// admin-creation mechanism. Setup itself already enforces "locked after the
// first user" (decision 5), so a second run against an initialized instance
// fails the same way the web wizard would. Validation reuses dto.SetupRequest
// + the shared validator so the CLI enforces the exact same rules (email
// format, password length) as the HTTP wizard — no second, parallel rule set.
func CreateAdminAndExit(email, password, name string) error {
	v, err := validator.New()
	if err != nil {
		return err
	}
	req := dto.SetupRequest{Email: email, Password: password, Name: name}
	if err := v.Struct(req, i18n.EN); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logging.New(cfg.IsProduction(), cfg.LogLevel)
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	users := postgres.NewUserRepository(pool)
	sessions := postgres.NewSessionRepository(pool)
	return auth.NewService(users, sessions).Setup(context.Background(), req.Email, req.Password, req.Name)
}
