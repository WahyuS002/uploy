package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrDeploymentInProgress = errors.New("deployment already in progress")
var ErrDeploymentSnapshotMissing = errors.New("deployment configuration snapshot missing")

type Deployment struct {
	ID          string
	Status      string
	WorkspaceID string
	ServiceID   string
	CreatedAt   time.Time
	Phase       string
	IsActive    bool
	IsDraining  bool
	IsRolling   bool
}

// newDeployment takes the columns rather than a row struct: the queries select
// the same five fields but no longer the whole table, so sqlc gives each one its
// own row type with nothing in common but these.
func newDeployment(id, status string, workspaceID pgtype.Text, serviceID string, createdAt time.Time) Deployment {
	dep := Deployment{
		ID:        id,
		Status:    status,
		ServiceID: serviceID,
		CreatedAt: createdAt,
	}
	if workspaceID.Valid {
		dep.WorkspaceID = workspaceID.String
	}
	return dep
}

// CreateDeployment records a deployment together with the config it is about to
// ship. The caller passes the very config it will render into the `docker run`,
// so what is stored and what is executed cannot come apart — reading the config
// back from the database when the deploy finishes would record any edit that
// landed while it was running as though the server had received it.
func CreateDeployment(ctx context.Context, workspaceID, serviceID string, cfg ServiceConfig) (Deployment, error) {
	snapshot, err := encodeSnapshot(cfg)
	if err != nil {
		return Deployment{}, fmt.Errorf("encode configuration snapshot: %w", err)
	}
	row, err := Queries.CreateDeployment(ctx, sqlcgen.CreateDeploymentParams{
		WorkspaceID:           pgtype.Text{String: workspaceID, Valid: true},
		ServiceID:             serviceID,
		ConfigurationSnapshot: pgtype.Text{String: snapshot, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "uq_deployments_in_progress_service" {
			return Deployment{}, ErrDeploymentInProgress
		}
		return Deployment{}, err
	}
	return newDeployment(row.ID, row.Status, row.WorkspaceID, row.ServiceID, row.CreatedAt), nil
}

func ListDeploymentsByService(ctx context.Context, serviceID string, limit int32) ([]Deployment, error) {
	rows, err := Queries.ListDeploymentsByService(ctx, sqlcgen.ListDeploymentsByServiceParams{
		ServiceID: serviceID,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	deps := make([]Deployment, len(rows))
	for i, r := range rows {
		deps[i] = newDeployment(r.ID, r.Status, r.WorkspaceID, r.ServiceID, r.CreatedAt)
		deps[i].Phase = r.Phase
		deps[i].IsActive = r.IsActive
		deps[i].IsDraining = r.IsDraining
		if r.ConfigurationSnapshot.Valid {
			cfg, decodeErr := decodeSnapshot(r.ConfigurationSnapshot.String)
			if decodeErr == nil {
				deps[i].IsRolling = len(cfg.Domains) > 0
			}
		}
	}
	return deps, nil
}

func ListInProgressDeploymentIDs(ctx context.Context) ([]string, error) {
	return Queries.ListInProgressDeploymentIDs(ctx)
}

func GetLatestDeploymentPhase(ctx context.Context, deploymentID string) (string, error) {
	return Queries.GetLatestDeploymentPhase(ctx, deploymentID)
}

func GetDeploymentConfig(ctx context.Context, deploymentID string) (ServiceConfig, error) {
	stored, err := Queries.GetDeploymentConfig(ctx, deploymentID)
	if err != nil {
		return ServiceConfig{}, err
	}
	if !stored.Valid {
		return ServiceConfig{}, fmt.Errorf("deployment %s has no configuration snapshot", deploymentID)
	}
	return decodeSnapshot(stored.String)
}

func GetLatestSuccessfulDeploymentConfig(ctx context.Context, serviceID string) (Deployment, ServiceConfig, error) {
	row, err := Queries.GetLatestSuccessfulDeploymentConfig(ctx, serviceID)
	if err != nil {
		return Deployment{}, ServiceConfig{}, err
	}
	dep := Deployment{ID: row.ID, ServiceID: serviceID, Status: "success"}
	if !row.ConfigurationSnapshot.Valid {
		return dep, ServiceConfig{}, fmt.Errorf("%w: successful deployment %s", ErrDeploymentSnapshotMissing, row.ID)
	}
	cfg, err := decodeSnapshot(row.ConfigurationSnapshot.String)
	if err != nil {
		return dep, ServiceConfig{}, fmt.Errorf("decode deployment %s snapshot: %w", row.ID, err)
	}
	return dep, cfg, nil
}

func GetActiveDeploymentConfig(ctx context.Context, serviceID string) (Deployment, ServiceConfig, error) {
	deps, err := ListDeploymentsByService(ctx, serviceID, 2)
	if err != nil {
		return Deployment{}, ServiceConfig{}, err
	}
	for _, dep := range deps {
		if !dep.IsActive {
			continue
		}
		cfg, cfgErr := GetDeploymentConfig(ctx, dep.ID)
		if cfgErr != nil {
			return Deployment{}, ServiceConfig{}, cfgErr
		}
		return dep, cfg, nil
	}
	return Deployment{}, ServiceConfig{}, fmt.Errorf("service %s has no active deployment", serviceID)
}

func SetDeploymentStatus(ctx context.Context, deploymentID, status string) error {
	return Queries.SetDeploymentStatus(ctx, sqlcgen.SetDeploymentStatusParams{
		Status: status,
		ID:     deploymentID,
	})
}

func GetDeployment(ctx context.Context, deploymentID string) (Deployment, error) {
	row, err := Queries.GetDeployment(ctx, deploymentID)
	if err != nil {
		return Deployment{}, err
	}
	return newDeployment(row.ID, row.Status, row.WorkspaceID, row.ServiceID, row.CreatedAt), err
}

func AppendLog(ctx context.Context, deploymentID, output, logType, phase string) error {
	row, err := Queries.InsertDeploymentLog(ctx, sqlcgen.InsertDeploymentLogParams{
		DeploymentID: deploymentID,
		Output:       output,
		Type:         logType,
		Phase:        phase,
	})
	if err != nil {
		return err
	}
	broker.PublishLog(deploymentID, row.ID, int(row.Order), row.CreatedAt, output, logType, phase)
	return nil
}

type LogEntry struct {
	ID        int64     `json:"id"`
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"created_at"`
	Output    string    `json:"output"`
	Type      string    `json:"type"`
	Phase     string    `json:"phase"`
}

func GetLogsAfter(ctx context.Context, deploymentID string, afterOrder int) ([]LogEntry, error) {
	rows, err := Queries.GetLogsAfter(ctx, sqlcgen.GetLogsAfterParams{
		DeploymentID: deploymentID,
		Order:        int32(afterOrder),
	})
	if err != nil {
		return nil, err
	}
	logs := make([]LogEntry, len(rows))
	for i, r := range rows {
		logs[i] = LogEntry{
			ID:        r.ID,
			Order:     int(r.Order),
			CreatedAt: r.CreatedAt,
			Output:    r.Output,
			Type:      r.Type,
			Phase:     r.Phase,
		}
	}
	return logs, nil
}
