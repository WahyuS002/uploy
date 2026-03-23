-- +goose Up
ALTER TABLE applications ADD CONSTRAINT uq_applications_container_server UNIQUE (container_name, server_id);

-- +goose Down
ALTER TABLE applications DROP CONSTRAINT IF EXISTS uq_applications_container_server;
