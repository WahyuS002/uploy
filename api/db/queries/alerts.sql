-- name: CreateNotificationChannel :one
INSERT INTO notification_channels (workspace_id, name, type, config_ciphertext)
VALUES ($1, $2, $3, $4)
RETURNING id, workspace_id, name, type, config_ciphertext, enabled, created_at, updated_at;

-- name: ListNotificationChannels :many
SELECT id, workspace_id, name, type, config_ciphertext, enabled, created_at, updated_at
FROM notification_channels WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: GetNotificationChannel :one
SELECT id, workspace_id, name, type, config_ciphertext, enabled, created_at, updated_at
FROM notification_channels WHERE id = $1;

-- name: UpdateNotificationChannel :one
UPDATE notification_channels
SET name = $2, type = $3, config_ciphertext = $4, enabled = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, workspace_id, name, type, config_ciphertext, enabled, created_at, updated_at;

-- name: DeleteNotificationChannel :exec
DELETE FROM notification_channels WHERE id = $1;

-- name: CreateAlertRule :one
INSERT INTO alert_rules (workspace_id, name, condition, threshold, duration_seconds, scope_type, server_id, service_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, workspace_id, name, condition, threshold, duration_seconds, scope_type, server_id, service_id, enabled, created_at, updated_at;

-- name: ListAlertRules :many
SELECT id, workspace_id, name, condition, threshold, duration_seconds, scope_type, server_id, service_id, enabled, created_at, updated_at
FROM alert_rules WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: ListEnabledAlertRules :many
SELECT id, workspace_id, name, condition, threshold, duration_seconds, scope_type, server_id, service_id, enabled, created_at, updated_at
FROM alert_rules WHERE enabled = TRUE ORDER BY created_at ASC;

-- name: GetAlertRule :one
SELECT id, workspace_id, name, condition, threshold, duration_seconds, scope_type, server_id, service_id, enabled, created_at, updated_at
FROM alert_rules WHERE id = $1;

-- name: UpdateAlertRule :one
UPDATE alert_rules
SET name = $2, condition = $3, threshold = $4, duration_seconds = $5, scope_type = $6,
    server_id = $7, service_id = $8, enabled = $9, updated_at = NOW()
WHERE id = $1
RETURNING id, workspace_id, name, condition, threshold, duration_seconds, scope_type, server_id, service_id, enabled, created_at, updated_at;

-- name: DeleteAlertRule :exec
DELETE FROM alert_rules WHERE id = $1;

-- name: ListAlertRuleChannels :many
SELECT channel_id FROM alert_rule_channels WHERE rule_id = $1 ORDER BY channel_id;

-- name: AddAlertRuleChannel :exec
INSERT INTO alert_rule_channels (rule_id, channel_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteAlertRuleChannels :exec
DELETE FROM alert_rule_channels WHERE rule_id = $1;

-- name: FindActiveAlertEvent :one
SELECT id, workspace_id, rule_id, target_id, target_name, status, started_at, resolved_at, trigger_value, resolved_value, created_at
FROM alert_events WHERE rule_id = $1 AND target_id = $2 AND status = 'firing'
ORDER BY started_at DESC LIMIT 1;

-- name: CreateAlertEvent :one
INSERT INTO alert_events (workspace_id, rule_id, target_id, target_name, status, started_at, trigger_value)
VALUES ($1, $2, $3, $4, 'firing', $5, $6)
RETURNING id, workspace_id, rule_id, target_id, target_name, status, started_at, resolved_at, trigger_value, resolved_value, created_at;

-- name: ResolveAlertEvent :one
UPDATE alert_events
SET status = 'resolved', resolved_at = $2, resolved_value = $3
WHERE id = $1 AND status = 'firing'
RETURNING id, workspace_id, rule_id, target_id, target_name, status, started_at, resolved_at, trigger_value, resolved_value, created_at;

-- name: ListAlertEvents :many
SELECT id, workspace_id, rule_id, target_id, target_name, status, started_at, resolved_at, trigger_value, resolved_value, created_at
FROM alert_events WHERE workspace_id = $1
ORDER BY started_at DESC LIMIT $2 OFFSET $3;
