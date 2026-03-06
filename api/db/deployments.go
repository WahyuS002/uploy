package db

import (
	"context"
	"fmt"
	"time"
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

func AppendLog(deploymentID, output string) error {
	_, err := Pool.Exec(context.Background(),
		`INSERT INTO deployment_logs (deployment_id, output) VALUES ($1, $2)`, deploymentID, output)

	return err
}

type LogEntry struct {
	CreatedAt time.Time `json:"created_at"`
	Output string `json:"output"`
}

func GetLogsAfter(deploymentID string, after time.Time) ([]LogEntry, error) {
    rows, err := Pool.Query(context.Background(),
        `SELECT created_at, output
				 FROM deployment_logs
         WHERE deployment_id=$1 AND created_at > $2
         ORDER BY created_at ASC`,
        deploymentID, after)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var logs []LogEntry
    for rows.Next() {
        var l LogEntry
				if err := rows.Scan(&l.CreatedAt, &l.Output); err != nil {
					return nil, err
				}
        logs = append(logs, l)
    }
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return logs, nil
}