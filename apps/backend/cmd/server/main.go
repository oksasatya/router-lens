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
	infrahttp "router-lens/internal/infrastructure/http"
	"router-lens/internal/infrastructure/postgres"
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
		),
		fx.Invoke(runMigrations), // runs during startup, before the server listens
		fx.Invoke(startServer),   // binds the HTTP server to the lifecycle
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
