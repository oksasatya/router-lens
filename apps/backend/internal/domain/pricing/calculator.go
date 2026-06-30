// Package pricing holds pricing rules and the pure cost calculator. It depends
// on nothing outside stdlib + shopspring/decimal (domain purity).
// ponytail: O(1), two multiplications + one division — no loop needed.
package pricing

import "github.com/shopspring/decimal"

var oneMillion = decimal.NewFromInt(1_000_000)

// TokenUsage carries input and output token counts for a single LLM call.
type TokenUsage struct {
	InputTokens  int64
	OutputTokens int64
}

// Rule holds the per-1M-token prices for a model.
type Rule struct {
	InputPricePer1M  decimal.Decimal
	OutputPricePer1M decimal.Decimal
}

// Cost is the calculated dollar cost for a single LLM call.
// InputPrice1M and OutputPrice1M snapshot the rule at calculation time.
type Cost struct {
	USD           decimal.Decimal
	InputPrice1M  decimal.Decimal
	OutputPrice1M decimal.Decimal
}

// CalculateCost returns nil when rule is nil (the model is unpriced).
// Multiplies before dividing to avoid precision loss: tokens*price/1_000_000.
func CalculateCost(usage TokenUsage, rule *Rule) *Cost {
	if rule == nil {
		return nil
	}
	in := decimal.NewFromInt(usage.InputTokens).Mul(rule.InputPricePer1M).Div(oneMillion)
	out := decimal.NewFromInt(usage.OutputTokens).Mul(rule.OutputPricePer1M).Div(oneMillion)
	return &Cost{
		USD:           in.Add(out),
		InputPrice1M:  rule.InputPricePer1M,
		OutputPrice1M: rule.OutputPricePer1M,
	}
}
