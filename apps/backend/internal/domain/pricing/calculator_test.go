package pricing

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculateCost(t *testing.T) {
	t.Run("priced computes tokens*price/1e6", func(t *testing.T) {
		rule := &Rule{
			InputPricePer1M:  decimal.RequireFromString("3.00"),
			OutputPricePer1M: decimal.RequireFromString("15.00"),
		}
		c := CalculateCost(TokenUsage{InputTokens: 12000, OutputTokens: 1800}, rule)
		if c == nil {
			t.Fatal("expected a cost")
		}
		// 12000*3/1e6 = 0.036 ; 1800*15/1e6 = 0.027 ; total = 0.063
		if !c.USD.Equal(decimal.RequireFromString("0.063")) {
			t.Fatalf("cost: got %s want 0.063", c.USD.String())
		}
		if !c.InputPrice1M.Equal(rule.InputPricePer1M) {
			t.Fatalf("snapshot not captured: %s", c.InputPrice1M)
		}
	})

	t.Run("nil rule => unpriced", func(t *testing.T) {
		if c := CalculateCost(TokenUsage{InputTokens: 100, OutputTokens: 100}, nil); c != nil {
			t.Fatalf("expected nil cost for unpriced, got %+v", c)
		}
	})

	t.Run("zero tokens => zero cost", func(t *testing.T) {
		rule := &Rule{InputPricePer1M: decimal.NewFromInt(3), OutputPricePer1M: decimal.NewFromInt(15)}
		c := CalculateCost(TokenUsage{}, rule)
		if c == nil || !c.USD.Equal(decimal.Zero) {
			t.Fatalf("expected zero cost, got %+v", c)
		}
	})
}
