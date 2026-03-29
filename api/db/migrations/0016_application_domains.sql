-- +goose Up

-- 1. Create application_domains table
CREATE TABLE application_domains (
    id TEXT PRIMARY KEY DEFAULT 'dom-' || gen_random_uuid(),
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    domain TEXT NOT NULL UNIQUE,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'pending',
    last_error TEXT,
    last_reconciled_at TIMESTAMPTZ,
    ready_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_application_domains_application_id ON application_domains(application_id);
CREATE UNIQUE INDEX idx_application_domains_one_primary ON application_domains(application_id) WHERE is_primary = TRUE;

-- 2. Backfill from applications.fqdn
INSERT INTO application_domains (application_id, domain, is_primary, status)
SELECT id, fqdn, TRUE, 'pending'
FROM applications
WHERE fqdn IS NOT NULL;

-- 3. Rename proxy_last_checked_at → proxy_last_reconciled_at
ALTER TABLE servers RENAME COLUMN proxy_last_checked_at TO proxy_last_reconciled_at;

-- 4. Drop proxy_mode from servers
ALTER TABLE servers DROP COLUMN proxy_mode;

-- 5. Drop fqdn from applications
ALTER TABLE applications DROP COLUMN fqdn;

-- 6. tls_pending is no longer a valid server-level status
UPDATE servers SET proxy_status = 'ready' WHERE proxy_status = 'tls_pending';

-- +goose Down

-- Reverse step 6: no-op (we can't know which were originally tls_pending)

-- Reverse step 5: re-add fqdn
ALTER TABLE applications ADD COLUMN fqdn TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_applications_fqdn ON applications(fqdn) WHERE fqdn IS NOT NULL;

-- Backfill fqdn from application_domains (primary domains only)
UPDATE applications a
SET fqdn = d.domain
FROM application_domains d
WHERE d.application_id = a.id AND d.is_primary = TRUE;

-- Reverse step 4: re-add proxy_mode
ALTER TABLE servers ADD COLUMN proxy_mode TEXT NOT NULL DEFAULT 'none';
UPDATE servers SET proxy_mode = 'managed' WHERE proxy_status IN ('ready', 'degraded', 'port_conflict');

-- Reverse step 3: rename back
ALTER TABLE servers RENAME COLUMN proxy_last_reconciled_at TO proxy_last_checked_at;

-- Reverse step 1: drop table (includes index)
DROP TABLE IF EXISTS application_domains;
