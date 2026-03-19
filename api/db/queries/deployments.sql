-- name: CreateDeployment :one
INSERT INTO deployments (status, workspace_id) VALUES ('in_progress', $1)
RETURNING id, status, workspace_id;

-- name: SetDeploymentStatus :exec
UPDATE deployments SET status = $1 WHERE id = $2;

-- name: GetDeployment :one
SELECT id, status, workspace_id FROM deployments WHERE id = $1;
