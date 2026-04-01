-- name: CreateServiceDomain :one
INSERT INTO service_domains (service_id, domain, is_primary)
VALUES ($1, $2, $3)
RETURNING id, service_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at;

-- name: GetServiceDomainByID :one
SELECT id, service_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at
FROM service_domains WHERE id = $1;

-- name: ListDomainsByService :many
SELECT id, service_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at
FROM service_domains WHERE service_id = $1
ORDER BY is_primary DESC, created_at ASC;

-- name: ClearPrimaryByService :exec
UPDATE service_domains
SET is_primary = FALSE, updated_at = NOW()
WHERE service_id = $1 AND is_primary = TRUE;

-- name: UpdateServiceDomainPrimary :one
UPDATE service_domains
SET is_primary = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, service_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at;

-- name: DeleteServiceDomain :exec
DELETE FROM service_domains WHERE id = $1;

-- name: SetDomainReady :exec
UPDATE service_domains
SET status = 'ready', ready_at = NOW(), last_error = NULL, last_reconciled_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: SetDomainError :exec
UPDATE service_domains
SET status = 'error', last_error = $2, last_reconciled_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListUnresolvedDomains :many
SELECT d.id, d.domain, d.service_id,
       s.server_id,
       srv.host, srv.port AS server_port, srv.ssh_user,
       k.private_key
FROM service_domains d
JOIN services s ON s.id = d.service_id
JOIN servers srv ON srv.id = s.server_id
JOIN ssh_keys k ON k.id = srv.ssh_key_id
WHERE d.status IN ('pending', 'error');
