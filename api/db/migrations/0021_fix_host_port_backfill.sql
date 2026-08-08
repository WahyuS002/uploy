-- +goose Up

-- 0020 read every NULL host_port as "published on port:port" and backfilled it.
-- That reading only holds for a service with no domain. A service with a domain
-- takes the Traefik branch of buildDockerRunCmd and publishes nothing on the
-- host, so the backfill claimed a host port it never held — and for a pre-0019
-- nginx on port 80 it also made the row permanently unsaveable, because the API
-- rejects 80 and 443 as the proxy's own.
--
-- Hand those rows back to NULL, which is what they actually resolved to.
UPDATE services SET host_port = NULL
WHERE host_port = port
  AND (
    port IN (80, 443)
    OR EXISTS (SELECT 1 FROM service_domains d WHERE d.service_id = services.id)
  );

-- +goose Down

-- Undoing this would republish services that were never on the host, and would
-- re-brick the port-80 rows. Nothing worth restoring.
SELECT 1;
