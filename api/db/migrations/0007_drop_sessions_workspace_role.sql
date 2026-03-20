-- +goose Up
ALTER TABLE sessions DROP COLUMN workspace_role;

-- +goose Down
ALTER TABLE sessions ADD COLUMN workspace_role TEXT;
UPDATE sessions s SET workspace_role = wm.role
FROM workspace_memberships wm
WHERE wm.workspace_id = s.workspace_id AND wm.user_id = s.user_id;
DELETE FROM sessions WHERE workspace_role IS NULL;
ALTER TABLE sessions ALTER COLUMN workspace_role SET NOT NULL;
