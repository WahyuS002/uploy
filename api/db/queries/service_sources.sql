-- name: CreateServiceSource :one
INSERT INTO service_sources (service_id, provider, owner, repo, branch, root_dir, detected)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING service_id, provider, owner, repo, branch, root_dir, detected, created_at, updated_at;

-- name: GetServiceSourceByServiceID :one
SELECT service_id, provider, owner, repo, branch, root_dir, detected, created_at, updated_at
FROM service_sources
WHERE service_id = $1;

-- name: ListServiceSourcesByServiceIDs :many
SELECT service_id, provider, owner, repo, branch, root_dir, detected, created_at, updated_at
FROM service_sources
WHERE service_id = ANY(@service_ids::text[])
ORDER BY service_id;
