package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"router-lens/internal/domain/pricing"
)

// PricingRepository implements pricing.PricingRepository using pgx.
type PricingRepository struct{ pool *pgxpool.Pool }

// NewPricingRepository returns a new PricingRepository.
func NewPricingRepository(pool *pgxpool.Pool) *PricingRepository {
	return &PricingRepository{pool: pool}
}

var _ pricing.PricingRepository = (*PricingRepository)(nil)

const pricingColumns = `id, provider, model, input_price_per_1m, output_price_per_1m, currency, created_at, updated_at`

func scanRule(row pgx.Row, r *pricing.PricingRule) error {
	return row.Scan(&r.ID, &r.Provider, &r.Model, &r.InputPricePer1M, &r.OutputPricePer1M, &r.Currency, &r.CreatedAt, &r.UpdatedAt)
}

func (r *PricingRepository) List(ctx context.Context) ([]*pricing.PricingRule, error) {
	q := `SELECT ` + pricingColumns + ` FROM pricing_rules ORDER BY provider, model`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*pricing.PricingRule, 0)
	for rows.Next() {
		var rule pricing.PricingRule
		if err := scanRule(rows, &rule); err != nil {
			return nil, err
		}
		out = append(out, &rule)
	}
	return out, rows.Err()
}

func (r *PricingRepository) FindByID(ctx context.Context, id string) (*pricing.PricingRule, error) {
	q := `SELECT ` + pricingColumns + ` FROM pricing_rules WHERE id = $1`
	var rule pricing.PricingRule
	err := scanRule(r.pool.QueryRow(ctx, q, id), &rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pricing.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *PricingRepository) FindByProviderModel(ctx context.Context, provider, model string) (*pricing.PricingRule, error) {
	q := `SELECT ` + pricingColumns + ` FROM pricing_rules WHERE provider = $1 AND model = $2`
	var rule pricing.PricingRule
	err := scanRule(r.pool.QueryRow(ctx, q, provider, model), &rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pricing.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *PricingRepository) Upsert(ctx context.Context, rule *pricing.PricingRule) error {
	const q = `INSERT INTO pricing_rules (provider, model, input_price_per_1m, output_price_per_1m, currency)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider, model) DO UPDATE SET
			input_price_per_1m = EXCLUDED.input_price_per_1m,
			output_price_per_1m = EXCLUDED.output_price_per_1m,
			currency = EXCLUDED.currency,
			updated_at = now()
		RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q, rule.Provider, rule.Model, rule.InputPricePer1M, rule.OutputPricePer1M, rule.Currency).
		Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *PricingRepository) Update(ctx context.Context, rule *pricing.PricingRule) error {
	const q = `UPDATE pricing_rules SET provider = $2, model = $3, input_price_per_1m = $4,
			output_price_per_1m = $5, currency = $6, updated_at = now()
		WHERE id = $1
		RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, q, rule.ID, rule.Provider, rule.Model, rule.InputPricePer1M, rule.OutputPricePer1M, rule.Currency).
		Scan(&rule.CreatedAt, &rule.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return pricing.ErrNotFound
	}
	if isUniqueViolation(err) {
		return pricing.ErrConflict
	}
	return err
}

func (r *PricingRepository) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM pricing_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pricing.ErrNotFound
	}
	return nil
}
