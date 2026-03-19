-- name: CreateUser :one
INSERT INTO users (email, password_hash) VALUES ($1, $2)
RETURNING id, email, password_hash, platform_role, status, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, platform_role, status, created_at, updated_at
FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, platform_role, status, created_at, updated_at
FROM users WHERE id = $1;
