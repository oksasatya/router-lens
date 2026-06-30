// Package bootstrap is the application composition root. It wires configuration,
// infrastructure, and the per-bounded-context modules into a runnable fx
// application. Nothing imports bootstrap except cmd/server, so it may import
// every other package (app, application, domain, infrastructure) without
// creating an import cycle — app.Config is consumed by the adapters, so the
// wiring cannot live in package app.
package bootstrap

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"go.uber.org/fx"

	"router-lens/internal/app"
	apikeyapp "router-lens/internal/application/apikey"
	"router-lens/internal/application/auth"
	pricingapp "router-lens/internal/application/pricing"
	projectapp "router-lens/internal/application/project"
	apikey "router-lens/internal/domain/apikey"
	pricing "router-lens/internal/domain/pricing"
	project "router-lens/internal/domain/project"
	"router-lens/internal/domain/user"
	infrahttp "router-lens/internal/infrastructure/http"
	"router-lens/internal/infrastructure/http/handler"
	"router-lens/internal/infrastructure/http/middleware"
	"router-lens/internal/infrastructure/postgres"
	"router-lens/internal/shared/validator"
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
		coreModule,
		authModule,
		projectModule,
		apiKeyModule,
		pricingModule,
		fx.Invoke(startServer),
	)
}

// coreModule provides config, the pgx pool, the Echo server, and the validator,
// and runs migrations before anything else starts.
var coreModule = fx.Module("core",
	fx.Provide(
		app.Load,
		providePool,
		infrahttp.NewServer,
		validator.New,
	),
	fx.Invoke(runMigrations),
)

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
		pricingapp.NewService,
		handler.NewPricingHandler,
	),
	fx.Invoke(registerPricingRoutes),
)

// providePool opens the pool and ties its lifetime to the fx lifecycle.
func providePool(lc fx.Lifecycle, cfg app.Config) (*pgxpool.Pool, error) {
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { pool.Close(); return nil }})
	return pool, nil
}

// runMigrations applies migrations during startup, before the server listens.
func runMigrations(pool *pgxpool.Pool) error {
	return postgres.Migrate(context.Background(), pool)
}

// startServer binds the HTTP server to the fx lifecycle. fx.Run handles SIGINT/SIGTERM.
func startServer(lc fx.Lifecycle, cfg app.Config, e *echo.Echo) {
	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           e,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatalf("listen: %v", err)
				}
			}()
			log.Printf("RouterLens listening on :%s", cfg.AppPort)
			return nil
		},
		OnStop: func(ctx context.Context) error { return srv.Shutdown(ctx) },
	})
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
	cfg, err := app.Load()
	if err != nil {
		return err
	}
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return postgres.Migrate(context.Background(), pool)
}
