package pricing

import "context"

// PricingRepository is the port for persisting and querying pricing rules.
type PricingRepository interface {
	List(ctx context.Context) ([]*PricingRule, error)
	FindByID(ctx context.Context, id string) (*PricingRule, error)
	// FindByProviderModel resolves a rule by its unique (provider, model) pair.
	// Returns ErrNotFound when the pair is unpriced.
	FindByProviderModel(ctx context.Context, provider, model string) (*PricingRule, error)
	// Upsert inserts r, or updates prices + currency on a (provider, model)
	// conflict. Sets ID/CreatedAt/UpdatedAt.
	Upsert(ctx context.Context, r *PricingRule) error
	// Update changes provider/model/prices/currency by id. Returns ErrNotFound
	// when the id is absent, ErrConflict when (provider, model) collides with
	// a different row.
	Update(ctx context.Context, r *PricingRule) error
	Delete(ctx context.Context, id string) error
}
