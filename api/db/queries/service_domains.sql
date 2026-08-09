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

-- name: ListDomainNamesByServiceIDs :many
-- Just the hostnames, for a whole set of services at once — what a deploy puts
-- in the container's Traefik rule, and so what a config snapshot holds. Sorted
-- within each service for the same reason the env vars are: the config is
-- compared as a document, and reordering must not read as a change.
SELECT service_id, domain FROM service_domains
WHERE service_id = ANY(@service_ids::text[])
ORDER BY service_id, domain ASC;

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

-- name: SetDomainPending :exec
-- Puts a domain back to waiting, keeping ready_at as the record that it did work
-- once. Used when a domain that had a certificate stops resolving to this server
-- — the certificate is still on disk and still proves nothing about now.
UPDATE service_domains
SET status = 'pending', last_error = NULL, last_reconciled_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: TouchDomainChecked :exec
-- Records that a domain was looked at and nothing changed, so the panel can say
-- when the answer was last confirmed rather than only when it last moved.
UPDATE service_domains
SET last_reconciled_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: SetDomainError :exec
UPDATE service_domains
SET status = 'error', last_error = $2, last_reconciled_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListDomainsToReconcile :many
-- The domains worth checking, ready ones included.
--
-- Readiness is read off acme.json, which is Traefik's store for the whole
-- server and is never pruned: a hostname that got a certificate once stays
-- spelled out in that file after the domain is removed, after the service is
-- redeployed without it, forever. Grepping it alone therefore answers "did this
-- name ever have a certificate here", not "is this service serving it now" —
-- and a domain added back later was promoted the next minute while no container
-- carried a matching Traefik rule at all.
--
-- The deploy requirement is what closes that: the Host() rule is only written
-- during a deploy, so a certificate can only belong to this domain if a
-- successful deploy happened after the domain row existed. An older one issued
-- the stale entry and does not count.
--
-- Ready domains are here too, which is why this is no longer "unresolved". A
-- certificate proves the name worked once; nothing about it expires when the DNS
-- record is deleted, so a domain that stopped resolving would otherwise keep
-- claiming HTTPS forever. Re-checking is what lets one fall back to pending, and
-- the DNS half of the check costs no SSH, so a pass over settled domains is
-- nearly free.
SELECT d.id, d.domain, d.status, d.service_id,
       s.server_id,
       srv.host, srv.port AS server_port, srv.ssh_user,
       k.private_key
FROM service_domains d
JOIN services s ON s.id = d.service_id
JOIN servers srv ON srv.id = s.server_id
JOIN ssh_keys k ON k.id = srv.ssh_key_id
WHERE d.status IN ('pending', 'error', 'ready')
  AND EXISTS (
    SELECT 1 FROM deployments dep
    WHERE dep.service_id = d.service_id
      AND dep.status = 'success'
      AND dep.created_at > d.created_at
  );
