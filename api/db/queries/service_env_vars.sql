-- name: UpsertServiceEnvVar :one
INSERT INTO service_env_vars (service_id, key, value)
VALUES ($1, $2, $3)
ON CONFLICT (service_id, key)
DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
RETURNING *;

-- name: ListServiceEnvVars :many
SELECT * FROM service_env_vars
WHERE service_id = $1
ORDER BY key ASC;

-- name: DeleteServiceEnvVar :exec
DELETE FROM service_env_vars
WHERE service_id = $1 AND key = $2;

-- name: GetServiceEnvVarsByServiceID :many
SELECT key, value FROM service_env_vars
WHERE service_id = $1
ORDER BY key ASC;
