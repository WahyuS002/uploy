package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/broker"
	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type Deployment struct {
	ID          string
	Status      string
	WorkspaceID string
	ServiceID   string
	CreatedAt   time.Time
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
	}
	return deps, nil
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
