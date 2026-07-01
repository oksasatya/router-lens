package pricing

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
)

type fakeRepo struct {
	upsertErr error
	updateErr error
	got       *pricingdomain.PricingRule
}

func (f *fakeRepo) List(context.Context) ([]*pricingdomain.PricingRule, error) { return nil, nil }
func (f *fakeRepo) FindByID(context.Context, string) (*pricingdomain.PricingRule, error) {
	return nil, nil
}
func (f *fakeRepo) Upsert(_ context.Context, r *pricingdomain.PricingRule) error {
	f.got = r
	if f.upsertErr != nil {
		return f.upsertErr
	}
	r.ID = "pr1"
	return nil
}
func (f *fakeRepo) Update(_ context.Context, r *pricingdomain.PricingRule) error { return f.updateErr }
func (f *fakeRepo) Delete(context.Context, string) error                         { return nil }
func (f *fakeRepo) FindByProviderModel(context.Context, string, string) (*pricingdomain.PricingRule, error) {
	return nil, pricingdomain.ErrNotFound
}

func TestUpsert(t *testing.T) {
	t.Run("defaults currency to USD", func(t *testing.T) {
		f := &fakeRepo{}
		_, err := NewService(f, nil).Upsert(context.Background(), Input{
			Provider: "anthropic", Model: "claude", Input: decimal.NewFromInt(3), Output: decimal.NewFromInt(15),
		})
		if err != nil || f.got.Currency != "USD" {
			t.Fatalf("currency=%q err=%v", f.got.Currency, err)
		}
	})
	t.Run("rejects negative price as validation error", func(t *testing.T) {
		_, err := NewService(&fakeRepo{}, nil).Upsert(context.Background(), Input{
			Provider: "p", Model: "m", Input: decimal.NewFromInt(-1), Output: decimal.NewFromInt(1),
		})
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindValidation {
			t.Fatalf("want validation AppError, got %v", err)
		}
	})
	t.Run("maps update conflict to 409", func(t *testing.T) {
		f := &fakeRepo{updateErr: pricingdomain.ErrConflict}
		err := NewService(f, nil).Update(context.Background(), "pr1", Input{
			Provider: "p", Model: "m", Input: decimal.NewFromInt(1), Output: decimal.NewFromInt(1),
		})
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindConflict {
			t.Fatalf("want conflict AppError, got %v", err)
		}
	})
}
