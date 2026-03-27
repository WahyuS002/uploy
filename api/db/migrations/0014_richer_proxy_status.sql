-- +goose Up

-- Replace simple boolean with richer status model
ALTER TABLE servers ADD COLUMN proxy_status TEXT NOT NULL DEFAULT 'not_configured';
ALTER TABLE servers ADD COLUMN proxy_last_checked_at TIMESTAMPTZ;
ALTER TABLE servers ADD COLUMN proxy_last_error TEXT;
-- none = no proxy managed, managed = Uploy manages Traefik on this server
ALTER TABLE servers ADD COLUMN proxy_mode TEXT NOT NULL DEFAULT 'none';

-- Migrate existing data
UPDATE servers SET proxy_status = 'ready', proxy_mode = 'managed' WHERE proxy_installed = TRUE;

ALTER TABLE servers DROP COLUMN proxy_installed;

-- +goose Down
ALTER TABLE servers ADD COLUMN proxy_installed BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE servers SET proxy_installed = TRUE WHERE proxy_status = 'ready';
ALTER TABLE servers DROP COLUMN proxy_status;
ALTER TABLE servers DROP COLUMN proxy_last_checked_at;
ALTER TABLE servers DROP COLUMN proxy_last_error;
ALTER TABLE servers DROP COLUMN proxy_mode;
