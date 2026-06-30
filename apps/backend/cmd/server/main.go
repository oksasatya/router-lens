package main

import (
	"context"
	"errors"
	"flag"
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

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations then exit")
	flag.Parse()

	if *migrateOnly {
		if err := migrateAndExit(); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		return
	}

	fx.New(
		fx.Provide(
			app.Load,            // () -> (app.Config, error)
			providePool,         // (fx.Lifecycle, app.Config) -> (*pgxpool.Pool, error)
			infrahttp.NewServer, // (app.Config) -> *echo.Echo
			validator.New,       // () -> (*validator.Validator, error)
			fx.Annotate(postgres.NewUserRepository, fx.As(new(user.UserRepository))),
			fx.Annotate(postgres.NewSessionRepository, fx.As(new(user.SessionRepository))),
			auth.NewService,          // (user.UserRepository, user.SessionRepository) -> *auth.Service
			handler.NewAuthHandler,   // (*auth.Service, *validator.Validator, app.Config) -> *AuthHandler
			provideSessionMiddleware, // (user.SessionRepository, user.UserRepository) -> echo.MiddlewareFunc
			fx.Annotate(postgres.NewProjectRepository, fx.As(new(project.ProjectRepository))),
			projectapp.NewService,     // (project.ProjectRepository) -> *projectapp.Service
			handler.NewProjectHandler, // (*projectapp.Service, *validator.Validator) -> *ProjectHandler
			fx.Annotate(postgres.NewAPIKeyRepository, fx.As(new(apikey.APIKeyRepository))),
			apikeyapp.NewService,     // (apikey.APIKeyRepository, project.ProjectRepository) -> *apikeyapp.Service
			handler.NewAPIKeyHandler, // (*apikeyapp.Service, *validator.Validator) -> *APIKeyHandler
			fx.Annotate(postgres.NewPricingRepository, fx.As(new(pricing.PricingRepository))),
			pricingapp.NewService,     // (pricing.PricingRepository) -> *pricingapp.Service
			handler.NewPricingHandler, // (*pricingapp.Service, *validator.Validator) -> *PricingHandler
		),
		fx.Invoke(runMigrations),
		fx.Invoke(registerAuthRoutes),
		fx.Invoke(registerProjectRoutes),
		fx.Invoke(registerAPIKeyRoutes),
		fx.Invoke(registerPricingRoutes),
		fx.Invoke(startServer),
	).Run()
}

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
	h.Register(e.Group("/api/v1"), session)
}

// registerProjectRoutes mounts the project routes behind the session middleware.
func registerProjectRoutes(e *echo.Echo, h *handler.ProjectHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}

// registerAPIKeyRoutes mounts the api-key routes behind the session middleware.
func registerAPIKeyRoutes(e *echo.Echo, h *handler.APIKeyHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}

// registerPricingRoutes mounts the pricing routes behind the session middleware.
func registerPricingRoutes(e *echo.Echo, h *handler.PricingHandler, session echo.MiddlewareFunc) {
	h.Register(e.Group("/api/v1"), session)
}

// migrateAndExit is the non-Fx path for `-migrate-only`.
func migrateAndExit() error {
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
