-- name: CreateProject :one
INSERT INTO projects (name, workspace_id)
VALUES ($1, $2)
RETURNING id, name, workspace_id, created_at, updated_at;

-- name: GetProjectByID :one
SELECT id, name, workspace_id, created_at, updated_at
FROM projects WHERE id = $1;

-- name: ListProjectsByWorkspace :many
SELECT id, name, workspace_id, created_at, updated_at
FROM projects WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, name, workspace_id, created_at, updated_at;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;
