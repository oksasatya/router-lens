package app

import "testing"

func TestParseConfig(t *testing.T) {
	t.Run("defaults applied and required present", func(t *testing.T) {
		env := map[string]string{
			"DATABASE_URL":   "postgres://x",
			"SESSION_SECRET": "secret",
		}
		cfg, err := parse(func(k string) string { return env[k] })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AppPort != "8080" || cfg.AppEnv != "development" {
			t.Fatalf("bad defaults: %+v", cfg)
		}
		if cfg.MaxBackdateDays != 7 || cfg.RetentionDays != 0 || cfg.CookieCrossSite {
			t.Fatalf("bad numeric/bool defaults: %+v", cfg)
		}
	})

	t.Run("missing required fails", func(t *testing.T) {
		if _, err := parse(func(string) string { return "" }); err == nil {
			t.Fatal("expected error for missing DATABASE_URL/SESSION_SECRET")
		}
	})

	t.Run("overrides parsed", func(t *testing.T) {
		env := map[string]string{
			"DATABASE_URL": "u", "SESSION_SECRET": "s",
			"APP_ENV": "production", "COOKIE_CROSS_SITE": "true", "MAX_BACKDATE_DAYS": "30",
		}
		cfg, _ := parse(func(k string) string { return env[k] })
		if !cfg.IsProduction() || !cfg.CookieCrossSite || cfg.MaxBackdateDays != 30 {
			t.Fatalf("overrides not applied: %+v", cfg)
		}
	})
}
