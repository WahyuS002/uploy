-- name: CreateDeployment :one
INSERT INTO deployments (status, workspace_id, service_id)
VALUES ('in_progress', $1, $2)
RETURNING id, status, workspace_id, service_id, created_at;

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
