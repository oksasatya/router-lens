package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"router-lens/internal/platform/config"
)

// discardLogger is a no-op logger for tests (no output noise).
func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestHealthz(t *testing.T) {
	e := NewServer(config.Config{AppEnv: "development", AppPort: "8080"}, discardLogger())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: got %d want 200", rec.Code)
	}
}

func TestSPAFallback_APIPathReturns404(t *testing.T) {
	e := NewServer(config.Config{AppEnv: "development", AppPort: "8080"}, discardLogger())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown api path: got %d want 404", rec.Code)
	}
}
