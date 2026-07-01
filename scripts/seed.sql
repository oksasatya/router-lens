-- Optional demo data for local exploration. Run AFTER first-run setup has
-- created at least one user (this script reads the first user as owner).
-- Usage: make seed  (or: psql "$DATABASE_URL" -f scripts/seed.sql)

INSERT INTO projects (owner_user_id, name, slug, description)
SELECT id, 'Demo Project', 'demo-project', 'Seeded via scripts/seed.sql'
FROM users ORDER BY created_at LIMIT 1
ON CONFLICT (owner_user_id, slug) DO NOTHING;

INSERT INTO pricing_rules (provider, model, input_price_per_1m, output_price_per_1m, currency)
VALUES
	('anthropic', 'claude-sonnet-4-5', 3.00, 15.00, 'USD'),
	('openai', 'gpt-4o', 2.50, 10.00, 'USD')
ON CONFLICT (provider, model) DO NOTHING;
