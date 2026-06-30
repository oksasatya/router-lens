-- +goose Up
CREATE TABLE projects (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id uuid NOT NULL REFERENCES users(id),
    name          text NOT NULL,
    slug          text NOT NULL,
    description   text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_user_id, slug)
);

-- +goose Down
DROP TABLE IF EXISTS projects;
