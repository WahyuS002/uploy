-- has_pending_changes is derived, not stored: a service is pending when no
-- *successful* deployment has landed at or after its last row change. That
-- covers both cases the canvas cares about — never deployed at all, and edited
-- since the last deploy. Derived rather than denormalised onto services so it
-- cannot drift out of sync with the deployments table, and expressed here
-- rather than in the client so the API and the UI can never disagree on what
-- "pending" means. idx_deployments_service_id keeps the lookup cheap.
--
-- Known gap: editing only env vars or domains does not bump services.updated_at,
-- so those edits do not mark the service pending.

-- name: CreateService :one
INSERT INTO services (name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    -- A service created one statement ago has no deployments, so it is always pending.
    TRUE::boolean AS has_pending_changes;

-- name: GetServiceByID :one
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (NOT EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success' AND d.created_at >= services.updated_at))::boolean AS has_pending_changes
FROM services WHERE services.id = $1;

-- name: ListServicesByWorkspace :many
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (NOT EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success' AND d.created_at >= services.updated_at))::boolean AS has_pending_changes
FROM services WHERE services.workspace_id = $1 ORDER BY created_at DESC;

-- name: ListServicesByEnvironment :many
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (NOT EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success' AND d.created_at >= services.updated_at))::boolean AS has_pending_changes
FROM services WHERE services.environment_id = $1 ORDER BY created_at DESC;

-- name: ListServicesByProject :many
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (NOT EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success' AND d.created_at >= services.updated_at))::boolean AS has_pending_changes
FROM services WHERE services.project_id = $1 ORDER BY created_at DESC;

-- name: UpdateService :one
UPDATE services
SET name = $2, image = $3, container_name = $4, port = $5, server_id = $6, updated_at = NOW()
WHERE services.id = $1
RETURNING id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    -- updated_at was just set to NOW(), so nothing can have deployed after it.
    TRUE::boolean AS has_pending_changes;

-- name: DeleteService :exec
DELETE FROM services WHERE id = $1;

-- name: GetServiceWithServer :one
SELECT
    s.id, s.name, s.image, s.container_name, s.port,
    s.server_id, s.workspace_id, s.kind, s.project_id, s.environment_id,
    s.created_at, s.updated_at,
    (NOT EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = s.id AND d.status = 'success' AND d.created_at >= s.updated_at))::boolean AS has_pending_changes,
    srv.host, srv.port AS server_port, srv.ssh_user,
    srv.proxy_status,
    k.private_key
FROM services s
JOIN servers srv ON srv.id = s.server_id
JOIN ssh_keys k ON k.id = srv.ssh_key_id
WHERE s.id = $1;
