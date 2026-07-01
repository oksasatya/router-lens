// Package event holds the event use cases: ingest (write path, this file) and
// list/get/export (read path, Task 3). Depends only on domain ports + the cost
// calculator + shared errors/i18n (no HTTP, no SQL).
package event

import (
	"context"
	"errors"
	"time"

	eventdomain "router-lens/internal/domain/event"
	pricingdomain "router-lens/internal/domain/pricing"
	apperrors "router-lens/internal/shared/errors"
	"router-lens/internal/shared/i18n"
)

// pricingLookup is the slice of the pricing port the ingest path needs.
type pricingLookup interface {
	FindByProviderModel(ctx context.Context, provider, model string) (*pricingdomain.PricingRule, error)
}

// IngestResult reports whether the event was newly stored or a duplicate.
type IngestResult struct {
	ID           string
	Deduplicated bool
}

type IngestService struct {
	events      eventdomain.EventRepository
	pricing     pricingLookup
	maxBackdate time.Duration
}

func NewIngestService(events eventdomain.EventRepository, pricing pricingLookup, maxBackdate time.Duration) *IngestService {
	return &IngestService{events: events, pricing: pricing, maxBackdate: maxBackdate}
}

// Ingest validates, prices, and idempotently stores an event for projectID.
func (s *IngestService) Ingest(ctx context.Context, projectID string, in eventdomain.IngestInput) (IngestResult, error) {
	now := time.Now().UTC()
	if err := eventdomain.Validate(in, now, s.maxBackdate); err != nil {
		return IngestResult{}, mapValidation(err)
	}
	e := buildEvent(projectID, in, now)
	if err := s.applyPricing(ctx, e); err != nil {
		return IngestResult{}, err
	}
	inserted, err := s.events.Insert(ctx, e)
	if err != nil {
		return IngestResult{}, err
	}
	return IngestResult{ID: e.ID, Deduplicated: !inserted}, nil
}

// applyPricing looks up the rule and snapshots cost; unpriced leaves NULLs.
func (s *IngestService) applyPricing(ctx context.Context, e *eventdomain.Event) error {
	rule, err := s.pricing.FindByProviderModel(ctx, e.Provider, e.Model)
	if errors.Is(err, pricingdomain.ErrNotFound) {
		return nil // unpriced: cost + snapshot stay nil
	}
	if err != nil {
		return err
	}
	r := rule.Rule()
	cost := pricingdomain.CalculateCost(
		pricingdomain.TokenUsage{InputTokens: e.InputTokens, OutputTokens: e.OutputTokens},
		&r,
	)
	if cost != nil {
		e.CostUSD = &cost.USD
		e.InputPrice1M = &cost.InputPrice1M
		e.OutputPrice1M = &cost.OutputPrice1M
	}
	return nil
}

func buildEvent(projectID string, in eventdomain.IngestInput, now time.Time) *eventdomain.Event {
	return &eventdomain.Event{
		ProjectID:         projectID,
		EventID:           in.EventID,
		Provider:          in.Provider,
		Model:             in.Model,
		RouteSource:       in.RouteSource,
		Agent:             in.Agent,
		InputTokens:       in.InputTokens,
		OutputTokens:      in.OutputTokens,
		LatencyMs:         in.LatencyMs,
		StatusCode:        in.StatusCode,
		IsError:           eventdomain.DeriveIsError(in.StatusCode, in.ErrorMessage),
		ErrorMessage:      in.ErrorMessage,
		RequestStartedAt:  in.RequestStartedAt,
		RequestFinishedAt: in.RequestFinishedAt,
		ReceivedAt:        now,
		Metadata:          in.Metadata,
	}
}

// mapValidation converts a domain validationError (carrying an i18n code via a
// Code() method) to a localized KindValidation AppError; any other error
// becomes a generic validation AppError.
func mapValidation(err error) error {
	var ve interface{ Code() string }
	if errors.As(err, &ve) {
		return apperrors.New(apperrors.KindValidation, ve.Code(), err.Error())
	}
	return apperrors.New(apperrors.KindValidation, i18n.CodeValidation, err.Error())
}
