-- +goose Up
ALTER TABLE deployments ADD COLUMN application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE;
ALTER TABLE deployments ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX idx_deployments_application_id ON deployments(application_id);

-- +goose Down
DROP INDEX IF EXISTS idx_deployments_application_id;
ALTER TABLE deployments DROP COLUMN IF EXISTS application_id;
ALTER TABLE deployments DROP COLUMN IF EXISTS created_at;
