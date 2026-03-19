package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type OAuthIdentity struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	ProviderEmail  string    `json:"provider_email"`
	CreatedAt      time.Time `json:"created_at"`
}

func CreateOAuthIdentityTx(ctx context.Context, tx pgx.Tx, userID, provider, providerUserID, providerEmail string) (OAuthIdentity, error) {
	var oi OAuthIdentity
	err := tx.QueryRow(ctx,
		`INSERT INTO oauth_identities (user_id, provider, provider_user_id, provider_email)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, provider, provider_user_id, provider_email, created_at`,
		userID, provider, providerUserID, providerEmail,
	).Scan(&oi.ID, &oi.UserID, &oi.Provider, &oi.ProviderUserID, &oi.ProviderEmail, &oi.CreatedAt)
	return oi, err
}

func GetOAuthIdentity(ctx context.Context, provider, providerUserID string) (OAuthIdentity, error) {
	var oi OAuthIdentity
	err := Pool.QueryRow(ctx,
		`SELECT id, user_id, provider, provider_user_id, provider_email, created_at
		 FROM oauth_identities WHERE provider = $1 AND provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&oi.ID, &oi.UserID, &oi.Provider, &oi.ProviderUserID, &oi.ProviderEmail, &oi.CreatedAt)
	return oi, err
}

func GetOAuthIdentitiesByUser(ctx context.Context, userID string) ([]OAuthIdentity, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id, user_id, provider, provider_user_id, provider_email, created_at
		 FROM oauth_identities WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []OAuthIdentity
	for rows.Next() {
		var oi OAuthIdentity
		if err := rows.Scan(&oi.ID, &oi.UserID, &oi.Provider, &oi.ProviderUserID, &oi.ProviderEmail, &oi.CreatedAt); err != nil {
			return nil, err
		}
		identities = append(identities, oi)
	}
	return identities, rows.Err()
}
