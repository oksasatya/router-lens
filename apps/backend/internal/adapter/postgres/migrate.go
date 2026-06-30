package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"router-lens/migrations"
)

// gooseSlog routes goose's migration output through slog so it shares the app's
// structured-logging format instead of printing plain text to stderr.
type gooseSlog struct{}

func (gooseSlog) Printf(format string, v ...any) {
	slog.Info(strings.TrimRight(fmt.Sprintf(format, v...), "\n"))
}

func (gooseSlog) Fatalf(format string, v ...any) {
	slog.Error(strings.TrimRight(fmt.Sprintf(format, v...), "\n"))
	os.Exit(1)
}

// Migrate applies all up migrations using goose over a database/sql handle
// derived from the pgx pool.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetLogger(gooseSlog{})
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("postgres: set dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("postgres: migrate up: %w", err)
	}
	return nil
}
