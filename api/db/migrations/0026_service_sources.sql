-- +goose Up

CREATE TABLE service_sources (
    service_id   TEXT PRIMARY KEY REFERENCES services(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL DEFAULT 'github' CHECK (provider = 'github'),
    owner        TEXT NOT NULL,
    repo         TEXT NOT NULL,
    branch       TEXT NOT NULL,
    root_dir     TEXT,
    detected     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down

DROP TABLE IF EXISTS service_sources;
