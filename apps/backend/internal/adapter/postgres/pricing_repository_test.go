package postgres

import (
	"testing"

	"github.com/shopspring/decimal"

	"router-lens/internal/domain/pricing"
)

func TestPricingRepository(t *testing.T) {
	ctx, pool := setupTestDB(t)
	repo := NewPricingRepository(pool)

	t.Run("upsert inserts then updates same pair", func(t *testing.T) {
		rule := &pricing.PricingRule{
			Provider: "anthropic", Model: "claude-test",
			InputPricePer1M: decimal.RequireFromString("3.00"), OutputPricePer1M: decimal.RequireFromString("15.00"),
			Currency: "USD",
		}
		if err := repo.Upsert(ctx, rule); err != nil {
			t.Fatalf("insert: %v", err)
		}
		firstID := rule.ID
		rule.InputPricePer1M = decimal.RequireFromString("4.50")
		if err := repo.Upsert(ctx, rule); err != nil {
			t.Fatalf("update upsert: %v", err)
		}
		got, err := repo.FindByID(ctx, firstID)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		if !got.InputPricePer1M.Equal(decimal.RequireFromString("4.50")) {
			t.Fatalf("price not updated: %s", got.InputPricePer1M)
		}
	})

	t.Run("delete missing -> ErrNotFound", func(t *testing.T) {
		if err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000000"); err != pricing.ErrNotFound {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
