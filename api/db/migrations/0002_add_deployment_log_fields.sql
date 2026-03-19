-- +goose Up
ALTER TABLE deployment_logs ADD COLUMN "order" INT;
ALTER TABLE deployment_logs ADD COLUMN type TEXT NOT NULL DEFAULT 'stdout';

CREATE INDEX idx_deployment_logs_deployment_id_order
    ON deployment_logs (deployment_id, "order");

-- +goose Down
DROP INDEX IF EXISTS idx_deployment_logs_deployment_id_order;
ALTER TABLE deployment_logs DROP COLUMN IF EXISTS type;
ALTER TABLE deployment_logs DROP COLUMN IF EXISTS "order";
