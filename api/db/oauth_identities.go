package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
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

func oauthIdentityFromGen(oi sqlcgen.OauthIdentity) OAuthIdentity {
	return OAuthIdentity{
		ID:             oi.ID,
		UserID:         oi.UserID,
		Provider:       oi.Provider,
		ProviderUserID: oi.ProviderUserID,
		ProviderEmail:  oi.ProviderEmail,
		CreatedAt:      oi.CreatedAt,
	}
}

func CreateOAuthIdentityTx(ctx context.Context, tx pgx.Tx, userID, provider, providerUserID, providerEmail string) (OAuthIdentity, error) {
	oi, err := sqlcgen.New(tx).CreateOAuthIdentity(ctx, sqlcgen.CreateOAuthIdentityParams{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		ProviderEmail:  providerEmail,
	})
	if err != nil {
		return OAuthIdentity{}, err
	}
	return oauthIdentityFromGen(oi), nil
}

func GetOAuthIdentity(ctx context.Context, provider, providerUserID string) (OAuthIdentity, error) {
	oi, err := Queries.GetOAuthIdentity(ctx, sqlcgen.GetOAuthIdentityParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		return OAuthIdentity{}, err
	}
	return oauthIdentityFromGen(oi), nil
}

func GetOAuthIdentitiesByUser(ctx context.Context, userID string) ([]OAuthIdentity, error) {
	rows, err := Queries.GetOAuthIdentitiesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	identities := make([]OAuthIdentity, len(rows))
	for i, r := range rows {
		identities[i] = oauthIdentityFromGen(r)
	}
	return identities, nil
}
