-- name: CreateApplication :one
INSERT INTO applications (name, image, container_name, port, server_id, workspace_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, image, container_name, port, server_id, workspace_id, created_at, updated_at;

-- name: GetApplicationByID :one
SELECT id, name, image, container_name, port, server_id, workspace_id, created_at, updated_at
FROM applications WHERE id = $1;

-- name: ListApplicationsByWorkspace :many
SELECT id, name, image, container_name, port, server_id, workspace_id, created_at, updated_at
FROM applications WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: UpdateApplication :one
UPDATE applications
SET name = $2, image = $3, container_name = $4, port = $5, server_id = $6, updated_at = NOW()
WHERE id = $1
RETURNING id, name, image, container_name, port, server_id, workspace_id, created_at, updated_at;

-- name: DeleteApplication :exec
DELETE FROM applications WHERE id = $1;

-- name: GetApplicationWithServer :one
SELECT
    a.id, a.name, a.image, a.container_name, a.port,
    a.server_id, a.workspace_id, a.created_at, a.updated_at,
    s.host, s.port AS server_port, s.ssh_user,
    s.proxy_status,
    k.private_key
FROM applications a
JOIN servers s ON s.id = a.server_id
JOIN ssh_keys k ON k.id = s.ssh_key_id
WHERE a.id = $1;
