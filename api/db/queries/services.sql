-- has_deployed is derived, not stored: true once at least one deployment of the
-- service has succeeded. It is what tells the two undeployed states apart —
-- false means the service has never landed on a server (the review dialog says
-- "will be added"), true means it is live ("will be updated").
-- idx_deployments_service_id keeps the lookup cheap.
--
-- Whether a service is *pending* is deliberately not here. It used to be: a
-- subquery comparing services.updated_at against the last successful deploy.
-- That asked "was this row edited since", which is a different question from
-- "does the server still match", and the two came apart in both directions —
-- env vars and domains live in their own tables and never moved updated_at, so
-- edits went unmarked, while an edit made and undone left the row marked
-- forever. Pending is now a comparison between the service's current config and
-- the config its last successful deployment actually shipped, which is the only
-- form of the question that can also say what differs. See db/service_config.go.

-- name: CreateService :one
INSERT INTO services (name, image, container_name, container_port, host_port, server_id, workspace_id, kind, project_id, environment_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, name, image, container_name, container_port, host_port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    FALSE::boolean AS has_deployed;

-- name: GetServiceByID :one
SELECT id, name, image, container_name, container_port, host_port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success'))::boolean AS has_deployed
FROM services WHERE services.id = $1;

-- name: ListServicesByWorkspace :many
SELECT id, name, image, container_name, container_port, host_port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success'))::boolean AS has_deployed
FROM services WHERE services.workspace_id = $1 ORDER BY created_at DESC;

-- name: ListServicesByEnvironment :many
SELECT id, name, image, container_name, container_port, host_port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success'))::boolean AS has_deployed
FROM services WHERE services.environment_id = $1 ORDER BY created_at DESC;

-- name: ListServicesByProject :many
SELECT id, name, image, container_name, container_port, host_port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success'))::boolean AS has_deployed
FROM services WHERE services.project_id = $1 ORDER BY created_at DESC;

-- name: UpdateService :one
UPDATE services
SET name = $2, image = $3, container_name = $4, container_port = $5, host_port = $6, server_id = $7, updated_at = NOW()
WHERE services.id = $1
RETURNING id, name, image, container_name, container_port, host_port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at,
    (EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = services.id AND d.status = 'success'))::boolean AS has_deployed;

-- name: DeleteService :exec
DELETE FROM services WHERE id = $1;

-- name: GetServiceWithServer :one
SELECT
    s.id, s.name, s.image, s.container_name, s.container_port, s.host_port,
    s.server_id, s.workspace_id, s.kind, s.project_id, s.environment_id,
    s.created_at, s.updated_at,
    (EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = s.id AND d.status = 'success'))::boolean AS has_deployed,
    srv.host, srv.port AS server_port, srv.ssh_user,
    srv.proxy_status,
    k.private_key
FROM services s
JOIN servers srv ON srv.id = s.server_id
JOIN ssh_keys k ON k.id = srv.ssh_key_id
WHERE s.id = $1;
