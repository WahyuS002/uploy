-- +goose Up

-- The control plane no longer reaches the agent over the server's own network.
-- It tunnels through the SSH connection it already holds, so the agent binds to
-- loopback and there is no address left for an operator to supply.
ALTER TABLE servers
    DROP COLUMN monitoring_private_address;

-- +goose Down

ALTER TABLE servers
    ADD COLUMN monitoring_private_address TEXT NOT NULL DEFAULT '';
