package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type ApplicationDomain struct {
	ID               string     `json:"id"`
	ApplicationID    string     `json:"application_id"`
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
	ID            string
	Domain        string
	ApplicationID string
	ServerID      string
	Host          string
	ServerPort    int32
	SSHUser       string
	EncryptedKey  string
}

func domainFromRow(r sqlcgen.ApplicationDomain) ApplicationDomain {
	return ApplicationDomain{
		ID:               r.ID,
		ApplicationID:    r.ApplicationID,
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

func CreateApplicationDomain(ctx context.Context, applicationID, domain string, isPrimary bool) (ApplicationDomain, error) {
	r, err := Queries.CreateApplicationDomain(ctx, sqlcgen.CreateApplicationDomainParams{
		ApplicationID: applicationID,
		Domain:        domain,
		IsPrimary:     isPrimary,
	})
	if err != nil {
		return ApplicationDomain{}, err
	}
	return domainFromRow(r), nil
}

func GetApplicationDomainByID(ctx context.Context, id string) (ApplicationDomain, error) {
	r, err := Queries.GetApplicationDomainByID(ctx, id)
	if err != nil {
		return ApplicationDomain{}, err
	}
	return domainFromRow(r), nil
}

func ListDomainsByApplication(ctx context.Context, applicationID string) ([]ApplicationDomain, error) {
	rows, err := Queries.ListDomainsByApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	domains := make([]ApplicationDomain, len(rows))
	for i, r := range rows {
		domains[i] = domainFromRow(r)
	}
	return domains, nil
}

// UpdateApplicationDomainPrimary sets is_primary on a domain. When promoting to
// primary, it first demotes any existing primary for the same application inside
// a transaction so the partial unique index is never violated.
func UpdateApplicationDomainPrimary(ctx context.Context, applicationID, id string, isPrimary bool) (ApplicationDomain, error) {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return ApplicationDomain{}, err
	}
	defer tx.Rollback(ctx)

	qtx := Queries.WithTx(tx)

	if isPrimary {
		if err := qtx.ClearPrimaryByApplication(ctx, applicationID); err != nil {
			return ApplicationDomain{}, err
		}
	}

	r, err := qtx.UpdateApplicationDomainPrimary(ctx, sqlcgen.UpdateApplicationDomainPrimaryParams{
		ID:        id,
		IsPrimary: isPrimary,
	})
	if err != nil {
		return ApplicationDomain{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ApplicationDomain{}, err
	}
	return domainFromRow(r), nil
}

func DeleteApplicationDomain(ctx context.Context, id string) error {
	return Queries.DeleteApplicationDomain(ctx, id)
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
			ID:            r.ID,
			Domain:        r.Domain,
			ApplicationID: r.ApplicationID,
			ServerID:      r.ServerID,
			Host:          r.Host,
			ServerPort:    r.ServerPort,
			SSHUser:       r.SshUser,
			EncryptedKey:  r.PrivateKey,
		}
	}
	return domains, nil
}
