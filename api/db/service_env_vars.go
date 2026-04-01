package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
)

type ServiceEnvVar struct {
	ID        int64     `json:"id"`
	ServiceID string    `json:"service_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func envFromGen(e sqlcgen.ServiceEnvVar) ServiceEnvVar {
	return ServiceEnvVar{
		ID:        e.ID,
		ServiceID: e.ServiceID,
		Key:       e.Key,
		Value:     e.Value,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func UpsertServiceEnvVar(ctx context.Context, serviceID, key, value string) (ServiceEnvVar, error) {
	e, err := Queries.UpsertServiceEnvVar(ctx, sqlcgen.UpsertServiceEnvVarParams{
		ServiceID: serviceID,
		Key:       key,
		Value:     value,
	})
	if err != nil {
		return ServiceEnvVar{}, err
	}
	return envFromGen(e), nil
}

func ListServiceEnvVars(ctx context.Context, serviceID string) ([]ServiceEnvVar, error) {
	rows, err := Queries.ListServiceEnvVars(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	envs := make([]ServiceEnvVar, len(rows))
	for i, r := range rows {
		envs[i] = envFromGen(r)
	}
	return envs, nil
}

func DeleteServiceEnvVar(ctx context.Context, serviceID, key string) error {
	return Queries.DeleteServiceEnvVar(ctx, sqlcgen.DeleteServiceEnvVarParams{
		ServiceID: serviceID,
		Key:       key,
	})
}

// EnvPair — for deploy job, only needs key=value
type EnvPair struct {
	Key   string
	Value string
}

func GetServiceEnvPairs(ctx context.Context, serviceID string) ([]EnvPair, error) {
	rows, err := Queries.GetServiceEnvVarsByServiceID(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	pairs := make([]EnvPair, len(rows))
	for i, r := range rows {
		pairs[i] = EnvPair{Key: r.Key, Value: r.Value}
	}
	return pairs, nil
}
