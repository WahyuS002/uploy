-- name: CreateSSHKey :one
INSERT INTO ssh_keys (name, private_key, workspace_id) VALUES ($1, $2, $3)
RETURNING id, name, private_key, workspace_id, created_at;

-- name: GetSSHKeyByID :one
SELECT id, name, private_key, workspace_id, created_at
FROM ssh_keys WHERE id = $1;

-- name: ListSSHKeysByWorkspace :many
SELECT id, name, workspace_id, created_at
FROM ssh_keys WHERE workspace_id = $1
ORDER BY created_at DESC;
