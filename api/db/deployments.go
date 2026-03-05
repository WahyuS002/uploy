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
