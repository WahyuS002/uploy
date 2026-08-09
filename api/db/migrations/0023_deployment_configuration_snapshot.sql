-- +goose Up

-- What this deployment actually put on the server, recorded at the moment the
-- deployment was created and never touched again.
--
-- Until now nothing remembered it. A deployment was assembled from the database,
-- rendered into a `docker run`, and forgotten — so "is the running container
-- still what this service says it is" had to be answered by comparing
-- services.updated_at against a deployment timestamp. That answers "was anything
-- edited", not "is anything different", and the two come apart constantly:
-- editing a domain and editing it back leaves the service permanently marked,
-- while editing an env var was never marked at all.
--
-- With the config itself on the row, the question becomes a comparison between
-- two configs, which is also the only thing that can say *what* differs.
--
-- Nullable, and rows written before this migration stay null. A null snapshot
-- has nothing to compare against, so those services read as pending until their
-- next deploy writes one — which is the honest answer rather than a guess in
-- either direction.
--
-- TEXT rather than JSONB because the value is encrypted before it gets here. A
-- config carries environment variable values, and those are encrypted at rest in
-- service_env_vars — storing them again in the clear on this row would quietly
-- undo that, and a deployment history is exactly the kind of table nobody thinks
-- of as holding secrets. Nothing queries inside a snapshot, so the structure is
-- worth nothing to the database and everything to the reader.
ALTER TABLE deployments ADD COLUMN configuration_snapshot TEXT;

-- +goose Down
ALTER TABLE deployments DROP COLUMN IF EXISTS configuration_snapshot;
