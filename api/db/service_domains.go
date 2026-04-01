package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type ServiceDomain struct {
	ID               string     `json:"id"`
	ServiceID        string     `json:"service_id"`
	Domain           string     `json:"domain"`
	IsPrimary        bool       `json:"is_primary"`
	Status           string     `json:"status"`
	LastError        *string    `json:"last_error"`
	LastReconciledAt *time.Time `json:"last_reconciled_at"`
	ReadyAt          *time.Time `json:"ready_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type PendingDomain struct {
	ID           string
	Domain       string
	ServiceID    string
	ServerID     string
	Host         string
	ServerPort   int32
	SSHUser      string
	EncryptedKey string
}

func domainFromRow(r sqlcgen.ServiceDomain) ServiceDomain {
	return ServiceDomain{
		ID:               r.ID,
		ServiceID:        r.ServiceID,
		Domain:           r.Domain,
		IsPrimary:        r.IsPrimary,
		Status:           r.Status,
		LastError:        stringPtrFromPgText(r.LastError),
		LastReconciledAt: timePtrFromPgTimestamptz(r.LastReconciledAt),
		ReadyAt:          timePtrFromPgTimestamptz(r.ReadyAt),
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func CreateServiceDomain(ctx context.Context, serviceID, domain string, isPrimary bool) (ServiceDomain, error) {
	r, err := Queries.CreateServiceDomain(ctx, sqlcgen.CreateServiceDomainParams{
		ServiceID: serviceID,
		Domain:    domain,
		IsPrimary: isPrimary,
	})
	if err != nil {
		return ServiceDomain{}, err
	}
	return domainFromRow(r), nil
}

func GetServiceDomainByID(ctx context.Context, id string) (ServiceDomain, error) {
	r, err := Queries.GetServiceDomainByID(ctx, id)
	if err != nil {
		return ServiceDomain{}, err
	}
	return domainFromRow(r), nil
}

func ListDomainsByService(ctx context.Context, serviceID string) ([]ServiceDomain, error) {
	rows, err := Queries.ListDomainsByService(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	domains := make([]ServiceDomain, len(rows))
	for i, r := range rows {
		domains[i] = domainFromRow(r)
	}
	return domains, nil
}

// UpdateServiceDomainPrimary sets is_primary on a domain. When promoting to
// primary, it first demotes any existing primary for the same service inside
// a transaction so the partial unique index is never violated.
func UpdateServiceDomainPrimary(ctx context.Context, serviceID, id string, isPrimary bool) (ServiceDomain, error) {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return ServiceDomain{}, err
	}
	defer tx.Rollback(ctx)

	qtx := Queries.WithTx(tx)

	if isPrimary {
		if err := qtx.ClearPrimaryByService(ctx, serviceID); err != nil {
			return ServiceDomain{}, err
		}
	}

	r, err := qtx.UpdateServiceDomainPrimary(ctx, sqlcgen.UpdateServiceDomainPrimaryParams{
		ID:        id,
		IsPrimary: isPrimary,
	})
	if err != nil {
		return ServiceDomain{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ServiceDomain{}, err
	}
	return domainFromRow(r), nil
}

func DeleteServiceDomain(ctx context.Context, id string) error {
	return Queries.DeleteServiceDomain(ctx, id)
}

func SetDomainReady(ctx context.Context, id string) error {
	return Queries.SetDomainReady(ctx, id)
}

func SetDomainError(ctx context.Context, id, lastError string) error {
	return Queries.SetDomainError(ctx, sqlcgen.SetDomainErrorParams{
		ID:        id,
		LastError: pgtype.Text{String: lastError, Valid: true},
	})
}

func ListUnresolvedDomains(ctx context.Context) ([]PendingDomain, error) {
	rows, err := Queries.ListUnresolvedDomains(ctx)
	if err != nil {
		return nil, err
	}
	domains := make([]PendingDomain, len(rows))
	for i, r := range rows {
		domains[i] = PendingDomain{
			ID:           r.ID,
			Domain:       r.Domain,
			ServiceID:    r.ServiceID,
			ServerID:     r.ServerID,
			Host:         r.Host,
			ServerPort:   r.ServerPort,
			SSHUser:      r.SshUser,
			EncryptedKey: r.PrivateKey,
		}
	}
	return domains, nil
}
