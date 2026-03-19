-- name: CreateOAuthIdentity :one
INSERT INTO oauth_identities (user_id, provider, provider_user_id, provider_email)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, provider, provider_user_id, provider_email, created_at;

-- name: GetOAuthIdentity :one
SELECT id, user_id, provider, provider_user_id, provider_email, created_at
FROM oauth_identities WHERE provider = $1 AND provider_user_id = $2;

-- name: GetOAuthIdentitiesByUser :many
SELECT id, user_id, provider, provider_user_id, provider_email, created_at
FROM oauth_identities WHERE user_id = $1;
