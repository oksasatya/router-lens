package dto

import (
	"github.com/shopspring/decimal"

	pricingdomain "router-lens/internal/domain/pricing"
)

// PricingRequest is the upsert/update payload for a pricing rule. The prices are
// pointers so `validate:"required"` rejects an omitted/null price at the
// boundary (a missing price is a 400) while still allowing an explicit 0 (free
// models) — an omitted price must never silently become a $0 rule.
type PricingRequest struct {
	Provider      string           `json:"provider" validate:"required,max=100"`
	Model         string           `json:"model" validate:"required,max=200"`
	InputPrice1M  *decimal.Decimal `json:"input_price_per_1m" validate:"required"`
	OutputPrice1M *decimal.Decimal `json:"output_price_per_1m" validate:"required"`
	Currency      string           `json:"currency" validate:"max=8"`
}

// PricingResponse is the wire shape of a pricing rule (prices as strings to
// preserve NUMERIC precision over JSON).
type PricingResponse struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	InputPrice1M  string `json:"input_price_per_1m"`
	OutputPrice1M string `json:"output_price_per_1m"`
	Currency      string `json:"currency"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// FromPricingRule maps a domain pricing rule to its response shape.
func FromPricingRule(p *pricingdomain.PricingRule) PricingResponse {
	return PricingResponse{
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
