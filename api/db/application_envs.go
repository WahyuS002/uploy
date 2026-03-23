package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
)

type ApplicationEnv struct {
	ID            int64     `json:"id"`
	ApplicationID string    `json:"application_id"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func envFromGen(e sqlcgen.ApplicationEnv) ApplicationEnv {
	return ApplicationEnv{
		ID:            e.ID,
		ApplicationID: e.ApplicationID,
		Key:           e.Key,
		Value:         e.Value,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

func UpsertApplicationEnv(ctx context.Context, applicationID, key, value string) (ApplicationEnv, error) {
	e, err := Queries.UpsertApplicationEnv(ctx, sqlcgen.UpsertApplicationEnvParams{
		ApplicationID: applicationID,
		Key:           key,
		Value:         value,
	})
	if err != nil {
		return ApplicationEnv{}, err
	}
	return envFromGen(e), nil
}

func ListApplicationEnvs(ctx context.Context, applicationID string) ([]ApplicationEnv, error) {
	rows, err := Queries.ListApplicationEnvs(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	envs := make([]ApplicationEnv, len(rows))
	for i, r := range rows {
		envs[i] = envFromGen(r)
	}
	return envs, nil
}

func DeleteApplicationEnv(ctx context.Context, applicationID, key string) error {
	return Queries.DeleteApplicationEnv(ctx, sqlcgen.DeleteApplicationEnvParams{
		ApplicationID: applicationID,
		Key:           key,
	})
}

// EnvPair — untuk deploy job, cuma butuh key=value
type EnvPair struct {
	Key   string
	Value string
}

func GetApplicationEnvPairs(ctx context.Context, applicationID string) ([]EnvPair, error) {
	rows, err := Queries.GetApplicationEnvsByAppID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	pairs := make([]EnvPair, len(rows))
	for i, r := range rows {
		pairs[i] = EnvPair{Key: r.Key, Value: r.Value}
	}
	return pairs, nil
}
