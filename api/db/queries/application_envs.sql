-- name: UpsertApplicationEnv :one
INSERT INTO application_envs (application_id, key, value)
VALUES ($1, $2, $3)
ON CONFLICT (application_id, key)
DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
RETURNING *;

-- name: ListApplicationEnvs :many
SELECT * FROM application_envs
WHERE application_id = $1
ORDER BY key ASC;

-- name: DeleteApplicationEnv :exec
DELETE FROM application_envs
WHERE application_id = $1 AND key = $2;

-- name: GetApplicationEnvsByAppID :many
SELECT key, value FROM application_envs
WHERE application_id = $1
ORDER BY key ASC;
