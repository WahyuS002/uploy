-- +goose Up
CREATE TABLE ssh_keys (
    id TEXT PRIMARY KEY DEFAULT 'sk-' || gen_random_uuid()::text,
    name TEXT NOT NULL,
    private_key TEXT NOT NULL,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ssh_keys_workspace_id ON ssh_keys(workspace_id);

-- +goose Down
DROP TABLE ssh_keys;
