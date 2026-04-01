-- +goose Up

-- 1. Create projects table
CREATE TABLE projects (
    id           TEXT PRIMARY KEY DEFAULT 'proj-' || gen_random_uuid()::text,
    name         TEXT NOT NULL,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_projects_workspace_id ON projects(workspace_id);

-- 2. Create environments table
CREATE TABLE environments (
    id         TEXT PRIMARY KEY DEFAULT 'env-' || gen_random_uuid()::text,
    name       TEXT NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_environments_project_id ON environments(project_id);

-- 3. Rename applications -> services
ALTER TABLE applications RENAME TO services;

-- 4. Add kind, project_id, environment_id columns
ALTER TABLE services ADD COLUMN kind TEXT NOT NULL DEFAULT 'application'
    CHECK (kind = 'application');
ALTER TABLE services ADD COLUMN project_id TEXT NOT NULL REFERENCES projects(id);
ALTER TABLE services ADD COLUMN environment_id TEXT NOT NULL REFERENCES environments(id);

-- 5. Update ID prefix default for new services
ALTER TABLE services ALTER COLUMN id SET DEFAULT 'svc-' || gen_random_uuid()::text;

-- 6. Rename indexes and constraints on services table
ALTER INDEX idx_applications_workspace_id RENAME TO idx_services_workspace_id;
ALTER TABLE services RENAME CONSTRAINT uq_applications_container_server TO uq_services_container_server;

-- 7. Rename application_domains -> service_domains
ALTER TABLE application_domains RENAME TO service_domains;
ALTER TABLE service_domains RENAME COLUMN application_id TO service_id;
ALTER INDEX idx_application_domains_application_id RENAME TO idx_service_domains_service_id;
ALTER INDEX idx_application_domains_one_primary RENAME TO idx_service_domains_one_primary;

-- 8. Rename application_envs -> service_env_vars
ALTER TABLE application_envs RENAME TO service_env_vars;
ALTER TABLE service_env_vars RENAME COLUMN application_id TO service_id;
ALTER INDEX idx_application_envs_app_id RENAME TO idx_service_env_vars_service_id;

-- 9. Rename deployments.application_id -> service_id
ALTER TABLE deployments RENAME COLUMN application_id TO service_id;
ALTER INDEX idx_deployments_application_id RENAME TO idx_deployments_service_id;

-- +goose Down

-- Reverse 9: deployments column rename
ALTER INDEX idx_deployments_service_id RENAME TO idx_deployments_application_id;
ALTER TABLE deployments RENAME COLUMN service_id TO application_id;

-- Reverse 8: service_env_vars -> application_envs
ALTER INDEX idx_service_env_vars_service_id RENAME TO idx_application_envs_app_id;
ALTER TABLE service_env_vars RENAME COLUMN service_id TO application_id;
ALTER TABLE service_env_vars RENAME TO application_envs;

-- Reverse 7: service_domains -> application_domains
ALTER INDEX idx_service_domains_one_primary RENAME TO idx_application_domains_one_primary;
ALTER INDEX idx_service_domains_service_id RENAME TO idx_application_domains_application_id;
ALTER TABLE service_domains RENAME COLUMN service_id TO application_id;
ALTER TABLE service_domains RENAME TO application_domains;

-- Reverse 6: rename indexes/constraints back
ALTER TABLE services RENAME CONSTRAINT uq_services_container_server TO uq_applications_container_server;
ALTER INDEX idx_services_workspace_id RENAME TO idx_applications_workspace_id;

-- Reverse 5: restore ID prefix
ALTER TABLE services ALTER COLUMN id SET DEFAULT 'app-' || gen_random_uuid()::text;

-- Reverse 4: drop new columns
ALTER TABLE services DROP COLUMN environment_id;
ALTER TABLE services DROP COLUMN project_id;
ALTER TABLE services DROP COLUMN kind;

-- Reverse 3: rename services -> applications
ALTER TABLE services RENAME TO applications;

-- Reverse 1-2: drop environments and projects
DROP TABLE IF EXISTS environments;
DROP TABLE IF EXISTS projects;
