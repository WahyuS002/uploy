-- +goose Up
CREATE TABLE application_envs (
    id             BIGSERIAL PRIMARY KEY,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    key            TEXT NOT NULL,
    value          TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (application_id, key)
);

CREATE INDEX idx_application_envs_app_id ON application_envs(application_id);

-- +goose Down
DROP TABLE IF EXISTS application_envs;
