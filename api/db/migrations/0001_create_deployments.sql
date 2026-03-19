-- +goose Up
CREATE TABLE deployments (
    id TEXT PRIMARY KEY,
    status TEXT
);

CREATE TABLE deployment_logs (
    id BIGSERIAL PRIMARY KEY,
    deployment_id TEXT NOT NULL REFERENCES deployments(id),
    output TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_deployment_logs_deployment_id_created_at
    ON deployment_logs (deployment_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_deployment_logs_deployment_id_created_at;
DROP TABLE IF EXISTS deployment_logs;
DROP TABLE IF EXISTS deployments;
