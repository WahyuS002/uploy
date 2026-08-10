-- +goose Up

-- Keep the newest interrupted row if an older build ever allowed concurrent
-- deploys. The API reconciler will resolve that row against Docker on startup.
WITH ranked AS (
    SELECT id,
           row_number() OVER (PARTITION BY service_id ORDER BY created_at DESC, id DESC) AS position
    FROM deployments
    WHERE status = 'in_progress'
)
UPDATE deployments
SET status = 'failed'
WHERE id IN (SELECT id FROM ranked WHERE position > 1);

CREATE UNIQUE INDEX uq_deployments_in_progress_service
ON deployments (service_id)
WHERE status = 'in_progress';

-- +goose Down
DROP INDEX IF EXISTS uq_deployments_in_progress_service;
