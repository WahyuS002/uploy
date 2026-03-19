-- +goose Up
CREATE TABLE oauth_identities (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    provider TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    provider_email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_oauth_provider_user ON oauth_identities (provider, provider_user_id);
CREATE INDEX idx_oauth_user_id ON oauth_identities (user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_oauth_user_id;
DROP INDEX IF EXISTS idx_oauth_provider_user;
DROP TABLE IF EXISTS oauth_identities;
