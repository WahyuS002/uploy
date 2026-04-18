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

-- name: ProjectNameExistsInWorkspace :one
SELECT EXISTS (
    SELECT 1 FROM projects
    WHERE workspace_id = $1 AND name = $2
) AS exists;

-- name: LockWorkspaceProjectNames :exec
-- Acquires a transaction-scoped Postgres advisory lock keyed on the workspace
-- so concurrent project-create transactions in the same workspace serialize
-- their name-allocation step. The lock is released automatically at COMMIT or
-- ROLLBACK. Safe to call in any transaction that inserts into `projects`.
SELECT pg_advisory_xact_lock(hashtextextended('project_name:' || sqlc.arg(workspace_id)::text, 0));

-- name: UpdateProject :one
UPDATE projects
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, name, workspace_id, created_at, updated_at;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;
