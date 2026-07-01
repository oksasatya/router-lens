package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"router-lens/internal/adapter/http/dto"
	mw "router-lens/internal/adapter/http/middleware"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
	eventapp "router-lens/internal/usecase/event"
)

type IngestHandler struct {
	svc *eventapp.IngestService
	v   *validator.Validator
}

func NewIngestHandler(svc *eventapp.IngestService, v *validator.Validator) *IngestHandler {
	return &IngestHandler{svc: svc, v: v}
}

// Register mounts POST /events behind the API-key middleware ONLY.
func (h *IngestHandler) Register(api *echo.Group, apiKey echo.MiddlewareFunc) {
	api.POST("/events", h.create, apiKey)
}

func (h *IngestHandler) create(c echo.Context) error {
	var req dto.EventIngestRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	res, err := h.svc.Ingest(c.Request().Context(), mw.CurrentProjectID(c), req.ToIngestInput())
	if err != nil {
		return err
	}
	body := map[string]any{"deduplicated": res.Deduplicated}
	if !res.Deduplicated {
		body["id"] = res.ID
	}
	return response.Data(c, http.StatusAccepted, body)
}
