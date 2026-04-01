-- name: CreateEnvironment :one
INSERT INTO environments (name, project_id)
VALUES ($1, $2)
RETURNING id, name, project_id, created_at, updated_at;

-- name: GetEnvironmentByID :one
SELECT id, name, project_id, created_at, updated_at
FROM environments WHERE id = $1;

-- name: ListEnvironmentsByProject :many
SELECT id, name, project_id, created_at, updated_at
FROM environments WHERE project_id = $1
ORDER BY created_at ASC;

-- name: UpdateEnvironment :one
UPDATE environments
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, name, project_id, created_at, updated_at;

-- name: DeleteEnvironment :exec
DELETE FROM environments WHERE id = $1;
