// Package pricing holds the Pricing CRUD use cases. Depends on the pricing
// domain port + shared/errors (no HTTP, no SQL).
package pricing

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"

	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

const defaultCurrency = "USD"

// Input is the validated command for creating/updating a rule (a params object,
// keeping service methods under S107).
type Input struct {
	Provider string
	Model    string
	Input    decimal.Decimal
	Output   decimal.Decimal
	Currency string
}

// PriceSuggestion is a third-party reference price for a (provider, model) pair,
// offered while filling in a Pricing Rule form. It is never persisted — see
// CONTEXT.md's "Pricing Suggestion" glossary note. Not a domain type: this is a
// UI convenience backed by an external integration, not a domain rule.
type PriceSuggestion struct {
	Provider         string
	Model            string
	InputPricePer1M  decimal.Decimal
	OutputPricePer1M decimal.Decimal
}

// SuggestionSource is the application-level port to a third-party price
// reference (OpenRouter). It lives here, not in internal/domain/pricing.
type SuggestionSource interface {
	List(ctx context.Context) ([]PriceSuggestion, error)
}

// ErrSuggestionsDisabled is returned by a SuggestionSource when
// PRICING_SUGGESTIONS_ENABLED=false. The HTTP layer maps this to 404.
var ErrSuggestionsDisabled = errors.New("pricing: suggestions disabled")

type Service struct {
	repo   pricingdomain.PricingRepository
	source SuggestionSource
}

func NewService(repo pricingdomain.PricingRepository, source SuggestionSource) *Service {
	return &Service{repo: repo, source: source}
}

func (s *Service) List(ctx context.Context) ([]*pricingdomain.PricingRule, error) {
	return s.repo.List(ctx)
}

// ListSuggestions returns third-party reference prices, or ErrSuggestionsDisabled
// if the feature is turned off by config (source is a nil-safe check away from
// a panic if a caller ever constructs a Service without one — production wiring
// always provides one via Fx).
func (s *Service) ListSuggestions(ctx context.Context) ([]PriceSuggestion, error) {
	if s.source == nil {
		return nil, ErrSuggestionsDisabled
	}
	return s.source.List(ctx)
}

func (s *Service) Upsert(ctx context.Context, in Input) (*pricingdomain.PricingRule, error) {
	r, err := s.build("", in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) Update(ctx context.Context, id string, in Input) error {
	r, err := s.build(id, in)
	if err != nil {
		return err
	}
	if err := s.repo.Update(ctx, r); err != nil {
		return s.mapErr(err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return s.mapErr(err)
	}
	return nil
}

// build validates the input and assembles a PricingRule. Negative prices are a
// validation error; an empty currency defaults to USD.
func (s *Service) build(id string, in Input) (*pricingdomain.PricingRule, error) {
	if in.Input.IsNegative() || in.Output.IsNegative() {
		return nil, apperrors.New(apperrors.KindValidation, i18n.CodePricingInvalidPrice, "price must not be negative")
	}
	currency := in.Currency
	if currency == "" {
		currency = defaultCurrency
	}
	return &pricingdomain.PricingRule{
		ID:               id,
		Provider:         in.Provider,
		Model:            in.Model,
		InputPricePer1M:  in.Input,
		OutputPricePer1M: in.Output,
		Currency:         currency,
	}, nil
}

func (s *Service) mapErr(err error) error {
	switch {
	case errors.Is(err, pricingdomain.ErrNotFound):
		return apperrors.New(apperrors.KindNotFound, i18n.CodePricingNotFound, "pricing rule not found")
	case errors.Is(err, pricingdomain.ErrConflict):
		return apperrors.New(apperrors.KindConflict, i18n.CodePricingDuplicate, "a pricing rule for this provider/model already exists")
	default:
		return err
	}
}
