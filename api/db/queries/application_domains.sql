-- name: CreateApplicationDomain :one
INSERT INTO application_domains (application_id, domain, is_primary)
VALUES ($1, $2, $3)
RETURNING id, application_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at;

-- name: GetApplicationDomainByID :one
SELECT id, application_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at
FROM application_domains WHERE id = $1;

-- name: ListDomainsByApplication :many
SELECT id, application_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at
FROM application_domains WHERE application_id = $1
ORDER BY is_primary DESC, created_at ASC;

-- name: ClearPrimaryByApplication :exec
UPDATE application_domains
SET is_primary = FALSE, updated_at = NOW()
WHERE application_id = $1 AND is_primary = TRUE;

-- name: UpdateApplicationDomainPrimary :one
UPDATE application_domains
SET is_primary = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, application_id, domain, is_primary, status, last_error, last_reconciled_at, ready_at, created_at, updated_at;

-- name: DeleteApplicationDomain :exec
DELETE FROM application_domains WHERE id = $1;

-- name: SetDomainReady :exec
UPDATE application_domains
SET status = 'ready', ready_at = NOW(), last_error = NULL, last_reconciled_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: SetDomainError :exec
UPDATE application_domains
SET status = 'error', last_error = $2, last_reconciled_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListUnresolvedDomains :many
SELECT d.id, d.domain, d.application_id,
       a.server_id,
       s.host, s.port AS server_port, s.ssh_user,
       k.private_key
FROM application_domains d
JOIN applications a ON a.id = d.application_id
JOIN servers s ON s.id = a.server_id
JOIN ssh_keys k ON k.id = s.ssh_key_id
WHERE d.status IN ('pending', 'error');
