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
WITH service_deployments AS (
    SELECT d.*,
           COALESCE((
               SELECT phase
               FROM deployment_logs l
               WHERE l.deployment_id = d.id
                 AND l.phase <> 'recovery_pending'
               ORDER BY l."order" DESC
               LIMIT 1
           ), '') AS phase
    FROM deployments d
    WHERE d.service_id = $1
), current_deployment AS (
    SELECT id, status, phase
    FROM service_deployments
    ORDER BY created_at DESC
    LIMIT 1
), latest_success AS (
    SELECT id
    FROM service_deployments
    WHERE status = 'success'
    ORDER BY created_at DESC
    LIMIT 1
)
, marked AS (
SELECT d.id,
       d.status::text AS status,
       d.workspace_id,
       d.service_id,
       d.created_at,
       d.configuration_snapshot,
       d.phase::text AS phase,
       COALESCE((CASE
           WHEN current_deployment.status = 'in_progress' AND current_deployment.phase IN ('active', 'drain')
               THEN d.id = current_deployment.id
           ELSE d.id = latest_success.id
       END), false)::boolean AS is_active,
       COALESCE(
           current_deployment.status = 'in_progress'
           AND current_deployment.phase = 'drain'
           AND d.id = latest_success.id,
           false
       )::boolean AS is_draining
FROM service_deployments d
LEFT JOIN current_deployment ON true
LEFT JOIN latest_success ON true
)
SELECT id, status, workspace_id, service_id, created_at, configuration_snapshot, phase, is_active, is_draining
FROM marked
WHERE id IN (SELECT id FROM marked ORDER BY created_at DESC LIMIT $2)
   OR is_active
   OR is_draining
ORDER BY created_at DESC;

-- name: ListInProgressDeploymentIDs :many
SELECT id
FROM deployments
WHERE status = 'in_progress'
ORDER BY created_at ASC;

-- name: GetLatestDeploymentPhase :one
SELECT COALESCE((
    SELECT phase
    FROM deployment_logs
    WHERE deployment_id = $1
      AND phase <> 'recovery_pending'
    ORDER BY "order" DESC
    LIMIT 1
), '')::text AS phase;

-- name: GetDeploymentConfig :one
SELECT configuration_snapshot
FROM deployments
WHERE id = $1;

-- name: GetLatestSuccessfulDeploymentConfig :one
SELECT id, configuration_snapshot
FROM deployments
WHERE service_id = $1
  AND status = 'success'
ORDER BY created_at DESC
LIMIT 1;
