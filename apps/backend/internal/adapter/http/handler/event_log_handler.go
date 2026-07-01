package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	"router-lens/internal/adapter/http/dto"
	eventdomain "router-lens/internal/domain/event"
	"router-lens/internal/shared/csv"
	"router-lens/internal/shared/datetime"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/pagination"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
	eventapp "router-lens/internal/usecase/event"
)

type EventLogHandler struct {
	query *eventapp.QueryService
	v     *validator.Validator
}

func NewEventLogHandler(query *eventapp.QueryService, v *validator.Validator) *EventLogHandler {
	return &EventLogHandler{query: query, v: v}
}

// Register mounts the session-authenticated read routes.
func (h *EventLogHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/events", h.list, session)
	api.GET("/events/export.csv", h.exportCSV, session)
	api.GET("/events/:id", h.get, session)
}

func (h *EventLogHandler) list(c echo.Context) error {
	f, err := h.parseFilter(c)
	if err != nil {
		return err
	}
	pageCursor, err := pagination.DecodeCursor(c.QueryParam("cursor"))
	if err != nil {
		return apperrors.New(apperrors.KindValidation, i18n.CodeValidation, "invalid cursor")
	}
	f.CursorTime, f.CursorID = pageCursor.Time, pageCursor.ID
	events, hasMore, err := h.query.List(c.Request().Context(), f)
	if err != nil {
		return err
	}
	dtos := make([]dto.EventResponse, 0, len(events))
	for _, e := range events {
		dtos = append(dtos, dto.FromEvent(e))
	}
	cursor := ""
	if hasMore && len(events) > 0 {
		last := events[len(events)-1]
		cursor = pagination.EncodeCursor(pagination.Cursor{Time: last.RequestStartedAt, ID: last.ID})
	}
	return response.Data(c, http.StatusOK, map[string]any{"items": dtos, "next_cursor": cursor})
}

func (h *EventLogHandler) get(c echo.Context) error {
	e, err := h.query.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromEvent(e))
}

// parseFilter reads query params into a domain Filter: optional project/provider/
// model/is_error and a bounded date range (default 24h, max 90d). The keyset
// cursor is deliberately NOT handled here — only list() applies it; exportCSV
// always ignores pagination and must never touch/validate the cursor param.
func (h *EventLogHandler) parseFilter(c echo.Context) (eventdomain.Filter, error) {
	rng, err := datetime.ParseRange(c.QueryParam("from"), c.QueryParam("to"), c.QueryParam("preset"), time.Now().UTC())
	if err != nil {
		return eventdomain.Filter{}, err
	}
	off := pagination.ParseOffset("1", c.QueryParam("limit")) // reuse the [1,100] limit clamp only
	return eventdomain.Filter{
		ProjectID: c.QueryParam("project_id"),
		From:      rng.From,
		To:        rng.To,
		Provider:  c.QueryParam("provider"),
		Model:     c.QueryParam("model"),
		IsError:   parseBoolPtr(c.QueryParam("is_error")),
		Limit:     off.Limit,
	}, nil
}

func parseBoolPtr(s string) *bool {
	switch s {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	default:
		return nil
	}
}

// csvHeader is the export column order (kept once — S1192).
var csvHeader = []string{
	"id", "project_id", "provider", "model", "route_source", "agent",
	"input_tokens", "output_tokens", "cost_usd", "latency_ms", "status_code",
	"is_error", "error_message", "request_started_at",
}

// eventToCSVRow maps an event to a CSV record in csvHeader order. Unpriced cost
// renders as an empty cell (never "0" — preserves the unpriced signal).
func eventToCSVRow(e *eventdomain.Event) []string {
	return []string{
		e.ID, e.ProjectID, e.Provider, e.Model, e.RouteSource, e.Agent,
		strconv.FormatInt(e.InputTokens, 10), strconv.FormatInt(e.OutputTokens, 10),
		decimalCell(e.CostUSD), intCell(e.LatencyMs), intCell(e.StatusCode),
		strconv.FormatBool(e.IsError), e.ErrorMessage,
		e.RequestStartedAt.UTC().Format(time.RFC3339),
	}
}

func decimalCell(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

func intCell(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}

// exportCSV streams a range-bounded, injection-safe CSV. The HTTP status + CSV
// header row are written LAZILY on the first row (or for zero rows after the
// stream opens cleanly) so a query-open error returns a proper error envelope
// instead of a partial 200.
func (h *EventLogHandler) exportCSV(c echo.Context) error {
	f, err := h.parseFilter(c)
	if err != nil {
		return err
	}
	w := csv.NewWriter(c.Response())
	headerWritten := false
	writeHeader := func() error {
		c.Response().Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="events.csv"`)
		c.Response().WriteHeader(http.StatusOK)
		headerWritten = true
		return w.Write(csvHeader)
	}
	streamErr := h.query.Export(c.Request().Context(), f, func(e *eventdomain.Event) error {
		if !headerWritten {
			if err := writeHeader(); err != nil {
				return err
			}
		}
		return w.Write(eventToCSVRow(e))
	})
	if streamErr != nil && !headerWritten {
		return streamErr // query failed before any output -> proper error envelope
	}
	if streamErr != nil {
		return streamErr // partial stream after headers sent (rare); Echo logs it
	}
	if !headerWritten { // zero rows -> still return a valid header-only CSV
		if err := writeHeader(); err != nil {
			return err
		}
	}
	return w.Flush()
}
