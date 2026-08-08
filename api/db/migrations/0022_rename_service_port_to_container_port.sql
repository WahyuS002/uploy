-- +goose Up

-- `port` never said which side of the boundary it meant. That ambiguity is what
-- produced 0019/0020/0021: three migrations spent working out whether the column
-- held the port inside the container or the port on the host. It also collides
-- in GetServiceWithServer, where `s.port` sits next to `srv.port` (SSH) and only
-- an alias keeps them apart.
--
-- container_port/host_port is the pair the deploy code actually publishes
-- (`-p host_port:container_port`), so the columns now read the same as the flag.
ALTER TABLE services RENAME COLUMN port TO container_port;

-- +goose Down
ALTER TABLE services RENAME COLUMN container_port TO port;
