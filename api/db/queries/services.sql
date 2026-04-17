-- name: CreateService :one
INSERT INTO services (name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at;

-- name: GetServiceByID :one
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at
FROM services WHERE id = $1;

-- name: ListServicesByWorkspace :many
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at
FROM services WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: ListServicesByEnvironment :many
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at
FROM services WHERE environment_id = $1 ORDER BY created_at DESC;

-- name: ListServicesByProject :many
SELECT id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at
FROM services WHERE project_id = $1 ORDER BY created_at DESC;

-- name: UpdateService :one
UPDATE services
SET name = $2, image = $3, container_name = $4, port = $5, server_id = $6, updated_at = NOW()
WHERE id = $1
RETURNING id, name, image, container_name, port, server_id, workspace_id, kind, project_id, environment_id, created_at, updated_at;

-- name: DeleteService :exec
DELETE FROM services WHERE id = $1;

-- name: GetServiceWithServer :one
SELECT
    s.id, s.name, s.image, s.container_name, s.port,
    s.server_id, s.workspace_id, s.kind, s.project_id, s.environment_id,
    s.created_at, s.updated_at,
    srv.host, srv.port AS server_port, srv.ssh_user,
    srv.proxy_status,
    k.private_key
FROM services s
JOIN servers srv ON srv.id = s.server_id
JOIN ssh_keys k ON k.id = srv.ssh_key_id
WHERE s.id = $1;
