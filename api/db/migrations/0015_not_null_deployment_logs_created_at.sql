-- +goose Up
-- Fill any NULL rows (should be none, DEFAULT NOW() covers normal inserts)
UPDATE deployment_logs SET created_at = NOW() WHERE created_at IS NULL;
ALTER TABLE deployment_logs ALTER COLUMN created_at SET NOT NULL;

-- +goose Down
ALTER TABLE deployment_logs ALTER COLUMN created_at DROP NOT NULL;
