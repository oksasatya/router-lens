package app

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	envProduction = "production"
	defaultPort   = "8080"
)

type Config struct {
	AppEnv          string
	AppPort         string
	DatabaseURL     string
	SessionSecret   string
	CookieCrossSite bool
	MaxBackdateDays int
	RetentionDays   int
}

func (c Config) IsProduction() bool { return c.AppEnv == envProduction }

// Load reads configuration from the environment, loading a local .env first if present.
func Load() (Config, error) {
	_ = godotenv.Load() // best-effort in dev; ignored in prod where vars are set directly
	return parse(os.Getenv)
}

func parse(get func(string) string) (Config, error) {
	cfg := Config{
		AppEnv:          orDefault(get("APP_ENV"), "development"),
		AppPort:         orDefault(get("APP_PORT"), defaultPort),
		DatabaseURL:     get("DATABASE_URL"),
		SessionSecret:   get("SESSION_SECRET"),
		CookieCrossSite: get("COOKIE_CROSS_SITE") == "true",
		MaxBackdateDays: atoiOr(get("MAX_BACKDATE_DAYS"), 7),
		RetentionDays:   atoiOr(get("RETENTION_DAYS"), 0),
	}
	if cfg.DatabaseURL == "" || cfg.SessionSecret == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL and SESSION_SECRET are required")
	}
	return cfg, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func atoiOr(v string, def int) int {
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}
