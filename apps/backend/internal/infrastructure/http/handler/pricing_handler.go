package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	pricingapp "router-lens/internal/application/pricing"
	pricingdomain "router-lens/internal/domain/pricing"
	"router-lens/internal/shared/response"
	"router-lens/internal/shared/validator"
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

type pricingRequest struct {
	Provider      string           `json:"provider" validate:"required,max=100"`
	Model         string           `json:"model" validate:"required,max=200"`
	InputPrice1M  *decimal.Decimal `json:"input_price_per_1m" validate:"required"`
	OutputPrice1M *decimal.Decimal `json:"output_price_per_1m" validate:"required"`
	Currency      string           `json:"currency" validate:"max=8"`
}

// toInput dereferences the price pointers — safe because `validate:"required"`
// rejects an omitted/null price (nil) at the boundary BEFORE this runs. A price
// explicitly set to 0 is allowed (free models); a missing price is a 400. This
// is the distinction Codex (GPT-5.5) flagged: an omitted price must not silently
// become a $0 rule.
func (r pricingRequest) toInput() pricingapp.Input {
	return pricingapp.Input{
		Provider: r.Provider, Model: r.Model,
		Input: *r.InputPrice1M, Output: *r.OutputPrice1M, Currency: r.Currency,
	}
}

type pricingDTO struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	InputPrice1M  string `json:"input_price_per_1m"`
	OutputPrice1M string `json:"output_price_per_1m"`
	Currency      string `json:"currency"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func toPricingDTO(p *pricingdomain.PricingRule) pricingDTO {
	return pricingDTO{
		ID:            p.ID,
		Provider:      p.Provider,
		Model:         p.Model,
		InputPrice1M:  p.InputPricePer1M.String(),
		OutputPrice1M: p.OutputPricePer1M.String(),
		Currency:      p.Currency,
		CreatedAt:     p.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:     p.UpdatedAt.UTC().Format(timeLayout),
	}
}

func (h *PricingHandler) list(c echo.Context) error {
	rules, err := h.svc.List(c.Request().Context())
	if err != nil {
		return err
	}
	dtos := make([]pricingDTO, 0, len(rules))
	for _, p := range rules {
		dtos = append(dtos, toPricingDTO(p))
	}
	return response.Data(c, http.StatusOK, dtos)
}

func (h *PricingHandler) upsert(c echo.Context) error {
	var req pricingRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	p, err := h.svc.Upsert(c.Request().Context(), req.toInput())
	if err != nil {
		return err
	}
	return response.Created(c, toPricingDTO(p))
}

func (h *PricingHandler) update(c echo.Context) error {
	var req pricingRequest
	if err := bindAndValidate(c, h.v, &req); err != nil {
		return err
	}
	if err := h.svc.Update(c.Request().Context(), c.Param("id"), req.toInput()); err != nil {
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
