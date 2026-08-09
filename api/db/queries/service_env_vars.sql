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

-- name: GetServiceEnvVarsByServiceIDs :many
-- The same rows for a whole set of services at once, for building configs for a
-- list without a query per service. Ordered by key within each service because
-- the config compares as a document: a different order is a different snapshot,
-- and would show up as a change nobody made.
SELECT service_id, key, value FROM service_env_vars
WHERE service_id = ANY(@service_ids::text[])
ORDER BY service_id, key ASC;
