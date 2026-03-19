-- +goose Up
-- Requires PostgreSQL 13+ for built-in gen_random_uuid()
ALTER TABLE deployments ALTER COLUMN id SET DEFAULT 'dep-' || gen_random_uuid()::text;
ALTER TABLE users ALTER COLUMN id SET DEFAULT 'usr-' || gen_random_uuid()::text;
ALTER TABLE workspaces ALTER COLUMN id SET DEFAULT 'ws-' || gen_random_uuid()::text;
ALTER TABLE workspace_memberships ALTER COLUMN id SET DEFAULT 'wm-' || gen_random_uuid()::text;
ALTER TABLE oauth_identities ALTER COLUMN id SET DEFAULT 'oi-' || gen_random_uuid()::text;

-- +goose Down
ALTER TABLE deployments ALTER COLUMN id DROP DEFAULT;
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE workspaces ALTER COLUMN id DROP DEFAULT;
ALTER TABLE workspace_memberships ALTER COLUMN id DROP DEFAULT;
ALTER TABLE oauth_identities ALTER COLUMN id DROP DEFAULT;
