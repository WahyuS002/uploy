-- name: InsertDeploymentLog :one
INSERT INTO deployment_logs (deployment_id, "order", output, type, phase)
VALUES (@deployment_id, (SELECT COALESCE(MAX("order"), 0) + 1 FROM deployment_logs WHERE deployment_id = @deployment_id), @output, @type, @phase)
RETURNING id, "order", created_at;

-- name: GetLogsAfter :many
SELECT id, "order", created_at, output, type, phase
FROM deployment_logs
WHERE deployment_id = $1 AND "order" > $2
ORDER BY "order" ASC;
