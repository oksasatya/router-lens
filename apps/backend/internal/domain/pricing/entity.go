package pricing

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// ErrNotFound is returned when a pricing_rules row is absent.
var ErrNotFound = errors.New("pricing: not found")

// ErrConflict is returned when an update would collide with another row's
// (provider, model) unique pair.
var ErrConflict = errors.New("pricing: provider/model already exists")

// PricingRule is the full persisted rule. Its prices feed the cost calculator
// via Rule(); the calculator's value object stays the minimal Rule type.
type PricingRule struct {
	ID               string
	Provider         string
	Model            string
	InputPricePer1M  decimal.Decimal
	OutputPricePer1M decimal.Decimal
	Currency         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Rule returns the value object the cost calculator consumes.
func (p *PricingRule) Rule() Rule {
	return Rule{InputPricePer1M: p.InputPricePer1M, OutputPricePer1M: p.OutputPricePer1M}
}
