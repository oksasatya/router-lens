package event

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	eventdomain "router-lens/internal/domain/event"
	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
)

type fakeEventRepo struct {
	inserted bool
	got      *eventdomain.Event
}

func (f *fakeEventRepo) Insert(_ context.Context, e *eventdomain.Event) (bool, error) {
	f.got = e
	return f.inserted, nil
}
func (f *fakeEventRepo) List(context.Context, eventdomain.Filter) ([]*eventdomain.Event, error) {
	return nil, nil
}
func (f *fakeEventRepo) FindByID(context.Context, string) (*eventdomain.Event, error) {
	return nil, nil
}
func (f *fakeEventRepo) Export(context.Context, eventdomain.Filter, func(*eventdomain.Event) error) error {
	return nil
}

type fakePricingRepo struct{ rule *pricingdomain.PricingRule }

func (f *fakePricingRepo) FindByProviderModel(context.Context, string, string) (*pricingdomain.PricingRule, error) {
	if f.rule == nil {
		return nil, pricingdomain.ErrNotFound
	}
	return f.rule, nil
}

func validInput() eventdomain.IngestInput {
	return eventdomain.IngestInput{
		Provider: "anthropic", Model: "claude", InputTokens: 1_000_000, OutputTokens: 1_000_000,
		RequestStartedAt: time.Now().Add(-time.Minute),
	}
}

func TestIngest(t *testing.T) {
	t.Run("prices a known model and stores the snapshot", func(t *testing.T) {
		fr := &fakeEventRepo{inserted: true}
		pr := &fakePricingRepo{rule: &pricingdomain.PricingRule{
			InputPricePer1M: decimal.NewFromInt(3), OutputPricePer1M: decimal.NewFromInt(15),
		}}
		res, err := NewIngestService(fr, pr, 7*24*time.Hour).Ingest(context.Background(), "p1", validInput())
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Deduplicated {
			t.Fatal("new event should not be deduplicated")
		}
		if fr.got.CostUSD == nil || !fr.got.CostUSD.Equal(decimal.NewFromInt(18)) {
			t.Fatalf("cost = %v, want 18", fr.got.CostUSD)
		}
		if fr.got.InputPrice1M == nil || fr.got.ProjectID != "p1" {
			t.Fatal("snapshot/project not stored")
		}
	})
	t.Run("leaves cost NULL for an unpriced model", func(t *testing.T) {
		fr := &fakeEventRepo{inserted: true}
		_, err := NewIngestService(fr, &fakePricingRepo{rule: nil}, 7*24*time.Hour).
			Ingest(context.Background(), "p1", validInput())
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if fr.got.CostUSD != nil || fr.got.InputPrice1M != nil {
			t.Fatal("unpriced event must have NULL cost + snapshot")
		}
	})
	t.Run("duplicate -> deduplicated true", func(t *testing.T) {
		fr := &fakeEventRepo{inserted: false}
		res, _ := NewIngestService(fr, &fakePricingRepo{}, 7*24*time.Hour).
			Ingest(context.Background(), "p1", validInput())
		if !res.Deduplicated {
			t.Fatal("want deduplicated true")
		}
	})
	t.Run("maps validation error to KindValidation AppError", func(t *testing.T) {
		in := validInput()
		in.InputTokens = -1
		_, err := NewIngestService(&fakeEventRepo{}, &fakePricingRepo{}, 7*24*time.Hour).
			Ingest(context.Background(), "p1", in)
		ae, ok := apperrors.As(err)
		if !ok || ae.Kind != apperrors.KindValidation {
			t.Fatalf("want validation AppError, got %v", err)
		}
	})
}
