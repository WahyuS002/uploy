package db

import (
	"context"
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

func deploymentFromGen(d sqlcgen.Deployment) Deployment {
	dep := Deployment{
		ID:        d.ID,
		Status:    d.Status,
		ServiceID: d.ServiceID,
		CreatedAt: d.CreatedAt,
	}
	if d.WorkspaceID.Valid {
		dep.WorkspaceID = d.WorkspaceID.String
	}
	return dep
}

func CreateDeployment(ctx context.Context, workspaceID, serviceID string) (Deployment, error) {
	row, err := Queries.CreateDeployment(ctx, sqlcgen.CreateDeploymentParams{
		WorkspaceID: pgtype.Text{String: workspaceID, Valid: true},
		ServiceID:   serviceID,
	})
	if err != nil {
		return Deployment{}, err
	}
	return deploymentFromGen(row), nil
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
		deps[i] = deploymentFromGen(r)
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
	return deploymentFromGen(row), err
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
