-- +goose Up

-- One `port` column was doing two different jobs: the port the image listens on
-- inside the container, and the port the service is reachable on from outside.
-- They only coincide by luck — redis is 6379 on both, but nginx listens on 80
-- and cannot be published there because Traefik owns 80/443. Publishing
-- port:port made every image that listens on 80 unreachable.
--
-- port keeps its original meaning (what the image listens on). host_port is the
-- outside address, and NULL means "same as port", which is what every existing
-- row was already doing.
ALTER TABLE services ADD COLUMN host_port INTEGER;

-- +goose Down
ALTER TABLE services DROP COLUMN host_port;
