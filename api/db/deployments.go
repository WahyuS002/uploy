package db

import (
	"context"
	"fmt"
	"time"

	"github.com/WahyuS002/uploy/broker"
)

type Deployment struct {
	ID     string
	Status string
}

func CreateDeployment(ctx context.Context) (Deployment, error) {
	deploymentID := fmt.Sprintf("dep-%d", time.Now().UnixNano())

	_, err := Pool.Exec(ctx,
		`INSERT INTO deployments (id, status) VALUES ($1, 'in_progress')`,
		deploymentID,
	)
	if err != nil {
		return Deployment{}, err
	}

	return Deployment{
		ID:     deploymentID,
		Status: "in_progress",
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
	err := Pool.QueryRow(ctx,
		`SELECT id, status FROM deployments WHERE id=$1`,
		deploymentID,
	).Scan(&d.ID, &d.Status)

	return d, err
}

func AppendLog(ctx context.Context, deploymentID, output string) error {
	var id int64
	var createdAt time.Time
	err := Pool.QueryRow(ctx,
		`INSERT INTO deployment_logs (deployment_id, output) VALUES ($1, $2)
		 RETURNING id, created_at`,
		deploymentID, output).Scan(&id, &createdAt)
	if err != nil {
		return err
	}

	broker.PublishLog(deploymentID, id, createdAt, output)
	return nil
}

type LogEntry struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Output    string    `json:"output"`
}

func GetLogsAfter(ctx context.Context, deploymentID string, afterID int64) ([]LogEntry, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id, created_at, output
		 FROM deployment_logs
		 WHERE deployment_id=$1 AND id > $2
		 ORDER BY id ASC`,
		deploymentID, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.CreatedAt, &l.Output); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}