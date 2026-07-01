package openrouter

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestTransform(t *testing.T) {
	t.Run("splits provider/model and converts price to per-1M", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "openai/gpt-4o", Pricing: openRouterPricing{Prompt: "0.0000025", Completion: "0.00001"}},
		}
		got := transform(raw)
		if len(got) != 1 {
			t.Fatalf("want 1 suggestion, got %d", len(got))
		}
		want := decimal.NewFromFloat(2.5)
		if got[0].Provider != "openai" || got[0].Model != "gpt-4o" {
			t.Fatalf("provider/model split wrong: %+v", got[0])
		}
		if !got[0].InputPricePer1M.Equal(want) {
			t.Fatalf("input price = %s, want %s", got[0].InputPricePer1M, want)
		}
	})

	t.Run("skips entries with unknown (-1) pricing", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "some/model", Pricing: openRouterPricing{Prompt: "-1", Completion: "-1"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for unknown pricing, got %d", len(got))
		}
	})

	t.Run("skips aliases and malformed ids", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "~openai/gpt-4o-alias", Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
			{ID: "no-slash-here", Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
			{ID: "too/many/slashes", Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for alias/malformed ids, got %d", len(got))
		}
	})

	t.Run("skips entries with non-numeric pricing", func(t *testing.T) {
		raw := []openRouterModel{
			{ID: "openai/gpt-4o", Pricing: openRouterPricing{Prompt: "not-a-number", Completion: "0.00001"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for unparseable pricing, got %d", len(got))
		}
	})

	t.Run("skips oversized provider/model strings", func(t *testing.T) {
		longModel := ""
		for range 250 {
			longModel += "x"
		}
		raw := []openRouterModel{
			{ID: "openai/" + longModel, Pricing: openRouterPricing{Prompt: "0.000001", Completion: "0.000002"}},
		}
		if got := transform(raw); len(got) != 0 {
			t.Fatalf("want 0 suggestions for oversized model string, got %d", len(got))
		}
	})
}
