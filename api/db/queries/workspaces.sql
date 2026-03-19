-- name: CreateWorkspace :one
INSERT INTO workspaces (name, owner_user_id) VALUES ($1, $2)
RETURNING id, name, owner_user_id, created_at, updated_at;

-- name: GetWorkspace :one
SELECT id, name, owner_user_id, created_at, updated_at
FROM workspaces WHERE id = $1;
