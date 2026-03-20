-- +goose Up

-- Enable composite FK: servers.(ssh_key_id, workspace_id) → ssh_keys.(id, workspace_id)
-- This guarantees at the DB level that a server's SSH key belongs to the same workspace.
ALTER TABLE ssh_keys ADD CONSTRAINT uq_ssh_keys_id_workspace UNIQUE (id, workspace_id);

CREATE TABLE servers (
    id TEXT PRIMARY KEY DEFAULT 'srv-' || gen_random_uuid()::text,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 22,
    ssh_user TEXT NOT NULL,
    ssh_key_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (ssh_key_id, workspace_id) REFERENCES ssh_keys(id, workspace_id)
);

CREATE INDEX idx_servers_workspace_id ON servers(workspace_id);

-- +goose Down
DROP TABLE IF EXISTS servers;
ALTER TABLE ssh_keys DROP CONSTRAINT IF EXISTS uq_ssh_keys_id_workspace;
