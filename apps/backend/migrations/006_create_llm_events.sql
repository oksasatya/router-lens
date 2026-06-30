-- +goose Up
CREATE TABLE llm_events (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    event_id            text,
    provider            text NOT NULL,
    model               text NOT NULL,
    route_source        text,
    agent               text,
    input_tokens        bigint NOT NULL,
    output_tokens       bigint NOT NULL,
    cost_usd            numeric,
    input_price_1m      numeric,
    output_price_1m     numeric,
    latency_ms          integer,
    status_code         integer,
    is_error            boolean NOT NULL,
    error_message       text,
    request_started_at  timestamptz NOT NULL,
    request_finished_at timestamptz,
    received_at         timestamptz NOT NULL DEFAULT now(),
    metadata            jsonb,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_llm_events_idempotency
    ON llm_events (project_id, event_id) WHERE event_id IS NOT NULL;
CREATE INDEX idx_llm_events_project_time
    ON llm_events (project_id, request_started_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS llm_events;
