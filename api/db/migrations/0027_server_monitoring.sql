-- +goose Up

ALTER TABLE servers
    ADD COLUMN monitoring_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN monitoring_port INTEGER NOT NULL DEFAULT 9184,
    ADD COLUMN monitoring_retention_days INTEGER NOT NULL DEFAULT 7 CHECK (monitoring_retention_days BETWEEN 1 AND 30),
    ADD COLUMN monitoring_private_address TEXT NOT NULL DEFAULT '',
    ADD COLUMN monitoring_fqdn TEXT,
    ADD COLUMN monitoring_control_token TEXT,
    ADD COLUMN monitoring_reader_token TEXT,
    ADD COLUMN monitoring_status TEXT NOT NULL DEFAULT 'disabled',
    ADD COLUMN monitoring_last_reconciled_at TIMESTAMPTZ,
    ADD COLUMN monitoring_last_error TEXT,
    ADD COLUMN monitoring_cleanup_at TIMESTAMPTZ;

CREATE UNIQUE INDEX idx_servers_monitoring_fqdn
    ON servers (lower(monitoring_fqdn))
    WHERE monitoring_fqdn IS NOT NULL;

-- +goose Down

ALTER TABLE servers
    DROP COLUMN IF EXISTS monitoring_enabled,
    DROP COLUMN IF EXISTS monitoring_port,
    DROP COLUMN IF EXISTS monitoring_retention_days,
    DROP COLUMN IF EXISTS monitoring_private_address,
    DROP COLUMN IF EXISTS monitoring_fqdn,
    DROP COLUMN IF EXISTS monitoring_control_token,
    DROP COLUMN IF EXISTS monitoring_reader_token,
    DROP COLUMN IF EXISTS monitoring_status,
    DROP COLUMN IF EXISTS monitoring_last_reconciled_at,
    DROP COLUMN IF EXISTS monitoring_last_error,
    DROP COLUMN IF EXISTS monitoring_cleanup_at;
