-- name: CreateServer :one
INSERT INTO servers (name, host, port, ssh_user, ssh_key_id, workspace_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, host, port, ssh_user, ssh_key_id, workspace_id, proxy_status, proxy_mode, proxy_last_checked_at, proxy_last_error, created_at;

-- name: GetServerByID :one
SELECT id, name, host, port, ssh_user, ssh_key_id, workspace_id, proxy_status, proxy_mode, proxy_last_checked_at, proxy_last_error, created_at
FROM servers WHERE id = $1;

-- name: ListServersByWorkspace :many
SELECT id, name, host, port, ssh_user, ssh_key_id, workspace_id, proxy_status, proxy_mode, proxy_last_checked_at, proxy_last_error, created_at
FROM servers WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: SetServerProxyReady :exec
UPDATE servers
SET proxy_status = $2, proxy_mode = 'managed', proxy_last_checked_at = NOW(), proxy_last_error = NULL
WHERE id = $1;

-- name: SetServerProxyError :exec
UPDATE servers
SET proxy_status = $2, proxy_last_checked_at = NOW(), proxy_last_error = $3
WHERE id = $1;

-- name: GetServerWithKey :one
SELECT s.id, s.name, s.host, s.port, s.ssh_user, s.ssh_key_id, s.workspace_id, s.created_at,
       k.private_key
FROM servers s
JOIN ssh_keys k ON k.id = s.ssh_key_id
WHERE s.id = $1;

-- name: ListTLSPendingServers :many
SELECT s.id, s.host, s.port, s.ssh_user, k.private_key
FROM servers s
JOIN ssh_keys k ON k.id = s.ssh_key_id
WHERE s.proxy_status = 'tls_pending';
