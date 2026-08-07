-- +goose Up

-- host_port changes meaning here. It used to be read as "NULL means publish on
-- the same number as port"; from now on NULL means the service is not published
-- to the host at all — reachable by other services on the uploy network, and by
-- nothing outside the machine. That is the default a database should have.
--
-- Every existing row was written under the old reading, so leaving them NULL
-- would silently unpublish services that are running and reachable right now.
-- Backfill them to what they actually resolve to today, so the change of
-- meaning applies only to services created from here on.
UPDATE services SET host_port = port WHERE host_port IS NULL;

-- +goose Down

-- The old reading treats "host_port = port" and NULL identically, so collapsing
-- them back loses nothing.
UPDATE services SET host_port = NULL WHERE host_port = port;
