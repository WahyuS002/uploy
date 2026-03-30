-- +goose Up
ALTER TABLE deployment_logs ADD COLUMN phase TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE deployment_logs DROP COLUMN phase;
