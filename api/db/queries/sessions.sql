-- name: CreateSession :exec
INSERT INTO sessions (token, user_id, workspace_id, workspace_role, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetSession :one
SELECT token, user_id, workspace_id, workspace_role, created_at, expires_at
FROM sessions WHERE token = $1 AND expires_at > NOW();

-- name: ExtendSession :one
UPDATE sessions
SET expires_at = LEAST(NOW() + @idle_timeout::interval, created_at + @absolute_lifetime::interval)
WHERE token = @token AND expires_at > NOW()
RETURNING expires_at;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;
