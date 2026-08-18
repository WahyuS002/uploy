package db

import (
	"context"
	"fmt"
	"github.com/WahyuS002/uploy/telemetry"
	"time"

	"github.com/WahyuS002/uploy/crypto"
	"github.com/WahyuS002/uploy/db/sqlcgen"
)

// decryptEnvValue reads a stored env var value.
//
// ponytail: rows written before env vars were encrypted are still plaintext and
// fail to decrypt, so they are returned as-is. That makes an undecryptable value
// indistinguishable from a legacy one. Drop the fallback once a backfill has
// re-encrypted every existing row.
func decryptEnvValue(serviceID, key, stored string) string {
	plaintext, err := crypto.Decrypt(stored)
	if err != nil {
		telemetry.Printf("env var service=%s key=%s: reading as legacy plaintext: %v", serviceID, key, err)
		return stored
	}
	return plaintext
}

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
	encrypted, err := crypto.Encrypt(value)
	if err != nil {
		return ServiceEnvVar{}, fmt.Errorf("encrypt env var %s: %w", key, err)
	}
	e, err := Queries.UpsertServiceEnvVar(ctx, sqlcgen.UpsertServiceEnvVarParams{
		ServiceID: serviceID,
		Key:       key,
		Value:     encrypted,
	})
	if err != nil {
		return ServiceEnvVar{}, err
	}
	// Hand back the plaintext, not the ciphertext the database echoed.
	result := envFromGen(e)
	result.Value = value
	return result, nil
}

func ListServiceEnvVars(ctx context.Context, serviceID string) ([]ServiceEnvVar, error) {
	rows, err := Queries.ListServiceEnvVars(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	envs := make([]ServiceEnvVar, len(rows))
	for i, r := range rows {
		envs[i] = envFromGen(r)
		envs[i].Value = decryptEnvValue(serviceID, r.Key, r.Value)
	}
	return envs, nil
}

func DeleteServiceEnvVar(ctx context.Context, serviceID, key string) error {
	return Queries.DeleteServiceEnvVar(ctx, sqlcgen.DeleteServiceEnvVarParams{
		ServiceID: serviceID,
		Key:       key,
	})
}

// EnvPair is a variable as a deploy sees it: a name and a plaintext value, with
// none of the row's bookkeeping. It is also what a config snapshot stores, which
// is why it carries explicit JSON field names in db/service_config.go — those
// names are a storage format, not a detail of this struct.
type EnvPair struct {
	Key   string
	Value string
}
