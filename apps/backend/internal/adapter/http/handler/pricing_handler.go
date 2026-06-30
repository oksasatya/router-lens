package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"router-lens/internal/adapter/http/dto"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
	pricingapp "router-lens/internal/usecase/pricing"
)

// PricingHandler handles pricing rule CRUD endpoints.
type PricingHandler struct {
	svc *pricingapp.Service
	v   *validator.Validator
}

// NewPricingHandler returns a new PricingHandler.
func NewPricingHandler(svc *pricingapp.Service, v *validator.Validator) *PricingHandler {
	return &PricingHandler{svc: svc, v: v}
}

// Register mounts the pricing routes behind the session middleware.
func (h *PricingHandler) Register(api *echo.Group, session echo.MiddlewareFunc) {
	api.GET("/pricing", h.list, session)
	api.POST("/pricing", h.upsert, session)
	api.PUT("/pricing/:id", h.update, session)
	api.DELETE("/pricing/:id", h.delete, session)
}

// toPricingInput dereferences the price pointers — safe because the request's
// `validate:"required"` rejects an omitted/null price (nil) at the boundary
// BEFORE this runs. A price explicitly set to 0 is allowed (free models); a
// missing price is a 400. An omitted price must not silently become a $0 rule.
func toPricingInput(r dto.PricingRequest) pricingapp.Input {
	return pricingapp.Input{
		Provider: r.Provider, Model: r.Model,
		Input: *r.InputPrice1M, Output: *r.OutputPrice1M, Currency: r.Currency,
	}
}

func (h *PricingHandler) list(c echo.Context) error {
	rules, err := h.svc.List(c.Request().Context())
	if err != nil {
		return err
	}
	dtos := make([]dto.PricingResponse, 0, len(rules))
	for _, p := range rules {
		dtos = append(dtos, dto.FromPricingRule(p))
	}
	return response.Data(c, http.StatusOK, dtos)
}

func (h *PricingHandler) upsert(c echo.Context) error {
	var req dto.PricingRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	p, err := h.svc.Upsert(c.Request().Context(), toPricingInput(req))
	if err != nil {
		return err
	}
	return response.Created(c, dto.FromPricingRule(p))
}

func (h *PricingHandler) update(c echo.Context) error {
	var req dto.PricingRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	if err := h.svc.Update(c.Request().Context(), c.Param("id"), toPricingInput(req)); err != nil {
		return err
	}
	return response.NoContent(c)
}

func (h *PricingHandler) delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return response.NoContent(c)
}
