package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"router-lens/internal/adapter/http/dto"
	eventdomain "router-lens/internal/domain/event"
	"router-lens/internal/shared/datetime"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
	"router-lens/internal/shared/response"
	eventapp "router-lens/internal/usecase/event"
)

type AnalyticsHandler struct{ svc *eventapp.AnalyticsService }

func NewAnalyticsHandler(svc *eventapp.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// Register mounts the seven session-authenticated analytics routes.
func (h *AnalyticsHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/analytics/overview", h.overview, session)
	api.GET("/analytics/tokens", h.tokens, session)
	api.GET("/analytics/cost", h.cost, session)
	api.GET("/analytics/latency", h.latency, session)
	api.GET("/analytics/errors", h.errorsSeries, session)
	api.GET("/analytics/providers", h.providers, session)
	api.GET("/analytics/models", h.models, session)
}

// parseFilter reads the mandatory bounded date range + optional project scope
// shared by every endpoint. requireInterval is true for the four series
// endpoints, which additionally validate `interval` against the allow-list
// (defense-in-depth: the Postgres adapter validates it again independently).
func (h *AnalyticsHandler) parseFilter(c echo.Context, requireInterval bool) (eventdomain.AnalyticsFilter, error) {
	rng, err := datetime.ParseRange(c.QueryParam("from"), c.QueryParam("to"), c.QueryParam("preset"), time.Now().UTC())
	if err != nil {
		return eventdomain.AnalyticsFilter{}, err
	}
	f := eventdomain.AnalyticsFilter{ProjectID: c.QueryParam("project_id"), From: rng.From, To: rng.To}
	if !requireInterval {
		return f, nil
	}
	interval := c.QueryParam("interval")
	if interval == "" {
		interval = "day"
	}
	switch interval {
	case "hour", "day", "week":
		f.Interval = interval
		return f, nil
	default:
		return eventdomain.AnalyticsFilter{}, apperrors.New(apperrors.KindValidation, i18n.CodeAnalyticsInvalidInterval, "invalid interval")
	}
}

func (h *AnalyticsHandler) overview(c echo.Context) error {
	f, err := h.parseFilter(c, false)
	if err != nil {
		return err
	}
	result, err := h.svc.Overview(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromOverviewResult(result))
}

func (h *AnalyticsHandler) tokens(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.TokensSeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromTokenPoints(points))
}

func (h *AnalyticsHandler) cost(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.CostSeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromCostPoints(points))
}

func (h *AnalyticsHandler) latency(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.LatencySeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromLatencyPoints(points))
}

func (h *AnalyticsHandler) errorsSeries(c echo.Context) error {
	f, err := h.parseFilter(c, true)
	if err != nil {
		return err
	}
	points, err := h.svc.ErrorSeries(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromErrorPoints(points))
}

func (h *AnalyticsHandler) providers(c echo.Context) error {
	f, err := h.parseFilter(c, false)
	if err != nil {
		return err
	}
	stats, err := h.svc.Providers(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromProviderStats(stats))
}

func (h *AnalyticsHandler) models(c echo.Context) error {
	f, err := h.parseFilter(c, false)
	if err != nil {
		return err
	}
	stats, err := h.svc.Models(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return response.Data(c, http.StatusOK, dto.FromModelStats(stats))
}
