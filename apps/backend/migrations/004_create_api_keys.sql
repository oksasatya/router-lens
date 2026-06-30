-- +goose Up
CREATE TABLE api_keys (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         text NOT NULL,
    key_hash     text NOT NULL UNIQUE,
    key_prefix   text NOT NULL,
    last_used_at timestamptz,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_api_keys_project_id ON api_keys (project_id);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
