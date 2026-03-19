-- name: CreateMembership :one
INSERT INTO workspace_memberships (workspace_id, user_id, role) VALUES ($1, $2, $3)
RETURNING id, workspace_id, user_id, role, created_at;

-- name: GetMembership :one
SELECT id, workspace_id, user_id, role, created_at
FROM workspace_memberships WHERE workspace_id = $1 AND user_id = $2;

-- name: GetUserFirstWorkspace :one
SELECT
    w.id AS w_id, w.name AS w_name, w.owner_user_id AS w_owner_user_id,
    w.created_at AS w_created_at, w.updated_at AS w_updated_at,
    wm.id AS wm_id, wm.workspace_id AS wm_workspace_id, wm.user_id AS wm_user_id,
    wm.role AS wm_role, wm.created_at AS wm_created_at
FROM workspace_memberships wm
JOIN workspaces w ON w.id = wm.workspace_id
WHERE wm.user_id = $1
ORDER BY wm.created_at ASC
LIMIT 1;
