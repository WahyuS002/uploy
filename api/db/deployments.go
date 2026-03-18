package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/broker"
)

type Deployment struct {
	ID          string
	Status      string
	WorkspaceID string
}

func CreateDeployment(ctx context.Context, workspaceID string) (Deployment, error) {
	deploymentID := fmt.Sprintf("dep-%d", time.Now().UnixNano())

	_, err := Pool.Exec(ctx,
		`INSERT INTO deployments (id, status, workspace_id) VALUES ($1, 'in_progress', $2)`,
		deploymentID, workspaceID,
	)
	if err != nil {
		return Deployment{}, err
	}

	return Deployment{
		ID:          deploymentID,
		Status:      "in_progress",
		WorkspaceID: workspaceID,
	}, nil
}

func SetDeploymentStatus(ctx context.Context, deploymentID, status string) error {
	_, err := Pool.Exec(ctx,
		`UPDATE deployments SET status=$1 WHERE id=$2`,
		status, deploymentID,
	)
	return err
}

func GetDeployment(ctx context.Context, deploymentID string) (Deployment, error) {
	var d Deployment
	var wsID *string
	err := Pool.QueryRow(ctx,
		`SELECT id, status, workspace_id FROM deployments WHERE id=$1`,
		deploymentID,
	).Scan(&d.ID, &d.Status, &wsID)
	if wsID != nil {
		d.WorkspaceID = *wsID
	}

	return d, err
}

func AppendLog(ctx context.Context, deploymentID, output, logType string) error {
	var id int64
	var order int
	var createdAt time.Time
	err := Pool.QueryRow(ctx,
		`INSERT INTO deployment_logs (deployment_id, "order", output, type)
		 VALUES ($1, (SELECT COALESCE(MAX("order"), 0) + 1 FROM deployment_logs WHERE deployment_id=$1), $2, $3)
		 RETURNING id, "order", created_at`,
		deploymentID, output, logType).Scan(&id, &order, &createdAt)
	if err != nil {
		return err
	}

	broker.PublishLog(deploymentID, id, order, createdAt, output, logType)
	return nil
}

type LogEntry struct {
	ID        int64     `json:"id"`
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"created_at"`
	Output    string    `json:"output"`
	Type      string    `json:"type"`
}

func GetLogsAfter(ctx context.Context, deploymentID string, afterOrder int) ([]LogEntry, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id, "order", created_at, output, type
		 FROM deployment_logs
		 WHERE deployment_id=$1 AND "order" > $2
		 ORDER BY "order" ASC`,
		deploymentID, afterOrder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.Order, &l.CreatedAt, &l.Output, &l.Type); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}
