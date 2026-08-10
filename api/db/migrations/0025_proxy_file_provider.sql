-- +goose Up

-- Existing "ready" proxies predate the watched file provider and cannot safely
-- perform atomic rolling cutovers. Require the explicit one-time upgrade.
UPDATE servers
SET proxy_status = 'degraded',
    proxy_last_error = 'proxy upgrade required for rolling deployments'
WHERE proxy_status = 'ready';

-- +goose Down
UPDATE servers
SET proxy_status = 'ready',
    proxy_last_error = NULL
WHERE proxy_status = 'degraded'
  AND proxy_last_error = 'proxy upgrade required for rolling deployments';
