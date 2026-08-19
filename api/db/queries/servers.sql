-- name: CreateServer :one
INSERT INTO servers (name, host, port, ssh_user, ssh_key_id, workspace_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, host, port, ssh_user, ssh_key_id, workspace_id, proxy_status, proxy_last_reconciled_at, proxy_last_error, created_at,
          monitoring_enabled, monitoring_port, monitoring_retention_days, monitoring_fqdn, monitoring_control_token, monitoring_reader_token,
          monitoring_status, monitoring_last_reconciled_at, monitoring_last_error, monitoring_cleanup_at;

-- name: GetServerByID :one
SELECT id, name, host, port, ssh_user, ssh_key_id, workspace_id, proxy_status, proxy_last_reconciled_at, proxy_last_error, created_at,
       monitoring_enabled, monitoring_port, monitoring_retention_days, monitoring_fqdn, monitoring_control_token, monitoring_reader_token,
       monitoring_status, monitoring_last_reconciled_at, monitoring_last_error, monitoring_cleanup_at
FROM servers WHERE id = $1;

-- name: ListServersByWorkspace :many
SELECT id, name, host, port, ssh_user, ssh_key_id, workspace_id, proxy_status, proxy_last_reconciled_at, proxy_last_error, created_at,
       monitoring_enabled, monitoring_port, monitoring_retention_days, monitoring_fqdn, monitoring_control_token, monitoring_reader_token,
       monitoring_status, monitoring_last_reconciled_at, monitoring_last_error, monitoring_cleanup_at
FROM servers WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: SetServerProxyReady :exec
UPDATE servers
SET proxy_status = $2, proxy_last_reconciled_at = NOW(), proxy_last_error = NULL
WHERE id = $1;

-- name: SetServerProxyError :exec
UPDATE servers
SET proxy_status = $2, proxy_last_reconciled_at = NOW(), proxy_last_error = $3
WHERE id = $1;

-- name: GetServerWithKey :one
SELECT s.id, s.name, s.host, s.port, s.ssh_user, s.ssh_key_id, s.workspace_id, s.created_at,
       s.monitoring_enabled, s.monitoring_port, s.monitoring_retention_days, s.monitoring_fqdn, s.monitoring_control_token, s.monitoring_reader_token,
       s.monitoring_status, s.monitoring_last_reconciled_at, s.monitoring_last_error, s.monitoring_cleanup_at, k.private_key
FROM servers s
JOIN ssh_keys k ON k.id = s.ssh_key_id
WHERE s.id = $1;

-- name: SetServerMonitoring :exec
UPDATE servers
SET monitoring_enabled = sqlc.arg(monitoring_enabled)::boolean,
    monitoring_port = sqlc.arg(monitoring_port)::integer,
    monitoring_retention_days = sqlc.arg(monitoring_retention_days)::integer,
    monitoring_fqdn = NULLIF(sqlc.arg(monitoring_fqdn)::text, ''),
    monitoring_control_token = sqlc.arg(monitoring_control_token)::text,
    monitoring_reader_token = sqlc.arg(monitoring_reader_token)::text,
    monitoring_status = sqlc.arg(monitoring_status)::text,
    monitoring_last_reconciled_at = NOW(),
    monitoring_last_error = NULLIF(sqlc.arg(monitoring_last_error)::text, ''),
    monitoring_cleanup_at = NULLIF(sqlc.arg(monitoring_cleanup_at)::timestamptz, 'epoch'::timestamptz)
WHERE id = sqlc.arg(id)::text;

-- name: ListMonitoringCleanupDue :many
SELECT s.id, s.name, s.host, s.port, s.ssh_user, s.ssh_key_id, s.workspace_id, s.created_at,
       s.monitoring_enabled, s.monitoring_port, s.monitoring_retention_days, s.monitoring_fqdn, s.monitoring_control_token, s.monitoring_reader_token,
       s.monitoring_status, s.monitoring_last_reconciled_at, s.monitoring_last_error, s.monitoring_cleanup_at, k.private_key
FROM servers s
JOIN ssh_keys k ON k.id = s.ssh_key_id
WHERE s.monitoring_enabled = FALSE
  AND s.monitoring_cleanup_at IS NOT NULL
  AND s.monitoring_cleanup_at <= NOW();

-- name: ClearServerMonitoringData :exec
UPDATE servers
SET monitoring_control_token = NULL,
    monitoring_reader_token = NULL,
    monitoring_cleanup_at = NULL,
    monitoring_last_error = NULL
WHERE id = $1;

-- name: ServerFQDNInUse :one
SELECT EXISTS (
    SELECT 1
    FROM service_domains d
    WHERE lower(d.domain) = lower(sqlc.arg(fqdn)::text)
    UNION ALL
    SELECT 1
    FROM servers s
    WHERE s.id <> sqlc.arg(id)::text AND lower(s.monitoring_fqdn) = lower(sqlc.arg(fqdn)::text)
);
