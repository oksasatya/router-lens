// Package logging builds the application's structured slog logger.
package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// New builds the structured logger and installs it as slog's default. Output
// goes to stdout — not stderr — so dev tooling doesn't render normal logs as
// errors. Local development uses tint (colorized, human-readable); production
// uses slog's JSON handler (machine-parseable). Package-level slog calls and
// adapters (e.g. goose) share this default.
func New(production bool, level string) *slog.Logger {
	lvl := parseLevel(level)
	var handler slog.Handler
	if production {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	} else {
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      lvl,
			TimeFormat: time.TimeOnly,
		})
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
