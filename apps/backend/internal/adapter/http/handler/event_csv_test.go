package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
)

func TestEventToCSVRow(t *testing.T) {
	cost := decimal.RequireFromString("0.054")
	status := 200
	e := &eventdomain.Event{
		ID: "e1", ProjectID: "p1", Provider: "anthropic", Model: "claude",
		InputTokens: 12000, OutputTokens: 1800, CostUSD: &cost, StatusCode: &status,
		IsError: false, RequestStartedAt: time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
	}
	row := eventToCSVRow(e)
	if len(row) != len(csvHeader) {
		t.Fatalf("row width %d != header width %d", len(row), len(csvHeader))
	}

	unpriced := &eventdomain.Event{ID: "e2", Provider: "x", Model: "y", RequestStartedAt: time.Now().UTC()}
	urow := eventToCSVRow(unpriced)
	costIdx := indexOf(csvHeader, "cost_usd")
	if urow[costIdx] != "" {
		t.Fatalf("unpriced cost cell must be empty, got %q", urow[costIdx])
	}
}

// TestParseFilter_IgnoresCursor proves the export-cursor bug is fixed: parseFilter
// (the only path exportCSV goes through) must never decode/validate the "cursor"
// query param, so export.csv?cursor=<garbage> is not rejected. list() is the sole
// caller responsible for cursor validation, applied after parseFilter succeeds.
func TestParseFilter_IgnoresCursor(t *testing.T) {
	h := &EventLogHandler{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/events/export.csv?cursor=not-valid-base64!!", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	f, err := h.parseFilter(c)
	if err != nil {
		t.Fatalf("parseFilter must ignore the cursor param, got error: %v", err)
	}
	if !f.CursorTime.IsZero() || f.CursorID != "" {
		t.Fatalf("parseFilter must not set cursor fields, got CursorTime=%v CursorID=%q", f.CursorTime, f.CursorID)
	}
}

func indexOf(ss []string, target string) int {
	for i, s := range ss {
		if s == target {
			return i
		}
	}
	return -1
}
