-- +goose Up
-- Composite unique constraint harus dibuat sebelum CREATE TABLE karena FK mereferensinya
ALTER TABLE servers ADD CONSTRAINT uq_servers_id_workspace UNIQUE (id, workspace_id);

CREATE TABLE applications (
    id             TEXT PRIMARY KEY DEFAULT 'app-' || gen_random_uuid()::text,
    name           TEXT NOT NULL,
    image          TEXT NOT NULL,
    container_name TEXT NOT NULL,
    port           INTEGER NOT NULL DEFAULT 80,
    server_id      TEXT NOT NULL,
    workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (server_id, workspace_id) REFERENCES servers(id, workspace_id)
);

CREATE INDEX idx_applications_workspace_id ON applications(workspace_id);

-- +goose Down
DROP TABLE IF EXISTS applications;
ALTER TABLE servers DROP CONSTRAINT IF EXISTS uq_servers_id_workspace;
