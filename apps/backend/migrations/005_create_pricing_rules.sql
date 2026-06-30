-- +goose Up
CREATE TABLE pricing_rules (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            text NOT NULL,
    model               text NOT NULL,
    input_price_per_1m  numeric NOT NULL,
    output_price_per_1m numeric NOT NULL,
    currency            text NOT NULL DEFAULT 'USD',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, model)
);

-- +goose Down
DROP TABLE IF EXISTS pricing_rules;
