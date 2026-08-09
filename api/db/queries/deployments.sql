-- name: CreateDeployment :one
-- The snapshot is written here, at creation, from the same config object the
-- job is about to render into a `docker run` — not read back from the database
-- when the deploy finishes. A deploy takes tens of seconds, and an edit landing
-- during one would otherwise be recorded as deployed without ever reaching the
-- server, which is exactly the lie this column exists to prevent.
INSERT INTO deployments (status, workspace_id, service_id, configuration_snapshot)
VALUES ('in_progress', $1, $2, $3)
RETURNING id, status, workspace_id, service_id, created_at;

-- name: ListLatestSuccessfulConfigs :many
-- The config each service is actually running: its most recent successful
-- deployment's snapshot. One row per service, so a canvas full of them costs one
-- query rather than one each.
--
-- Failed deployments are excluded because they changed nothing on the server,
-- and null snapshots because they predate the column and describe nothing.
SELECT DISTINCT ON (service_id) service_id, configuration_snapshot
FROM deployments
WHERE service_id = ANY(@service_ids::text[])
  AND status = 'success'
  AND configuration_snapshot IS NOT NULL
ORDER BY service_id, created_at DESC;

-- name: SetDeploymentStatus :exec
UPDATE deployments SET status = $1 WHERE id = $2;

-- name: GetDeployment :one
SELECT id, status, workspace_id, service_id, created_at
FROM deployments WHERE id = $1;

-- name: ListDeploymentsByService :many
SELECT id, status, workspace_id, service_id, created_at
FROM deployments
WHERE service_id = $1
ORDER BY created_at DESC
LIMIT $2;
