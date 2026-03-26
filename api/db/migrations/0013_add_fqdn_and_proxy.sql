-- +goose Up

-- Application domain routing
ALTER TABLE applications ADD COLUMN fqdn TEXT;

-- Prevent two apps from claiming the same domain
CREATE UNIQUE INDEX idx_applications_fqdn ON applications (fqdn) WHERE fqdn IS NOT NULL;

-- Cache hint apakah bootstrap proxy pernah sukses di server
ALTER TABLE servers ADD COLUMN proxy_installed BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
DROP INDEX IF EXISTS idx_applications_fqdn;
ALTER TABLE applications DROP COLUMN IF EXISTS fqdn;
ALTER TABLE servers DROP COLUMN IF EXISTS proxy_installed;
