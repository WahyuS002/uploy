package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
)

type Environment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ProjectID string    `json:"project_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CreateEnvironment(ctx context.Context, name, projectID string) (Environment, error) {
	r, err := Queries.CreateEnvironment(ctx, sqlcgen.CreateEnvironmentParams{
		Name:      name,
		ProjectID: projectID,
	})
	if err != nil {
		return Environment{}, err
	}
	return Environment{
		ID: r.ID, Name: r.Name, ProjectID: r.ProjectID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func GetEnvironmentByID(ctx context.Context, id string) (Environment, error) {
	r, err := Queries.GetEnvironmentByID(ctx, id)
	if err != nil {
		return Environment{}, err
	}
	return Environment{
		ID: r.ID, Name: r.Name, ProjectID: r.ProjectID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func ListEnvironmentsByProject(ctx context.Context, projectID string) ([]Environment, error) {
	rows, err := Queries.ListEnvironmentsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	envs := make([]Environment, len(rows))
	for i, r := range rows {
		envs[i] = Environment{
			ID: r.ID, Name: r.Name, ProjectID: r.ProjectID,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return envs, nil
}

func UpdateEnvironment(ctx context.Context, id, name string) (Environment, error) {
	r, err := Queries.UpdateEnvironment(ctx, sqlcgen.UpdateEnvironmentParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return Environment{}, err
	}
	return Environment{
		ID: r.ID, Name: r.Name, ProjectID: r.ProjectID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func DeleteEnvironment(ctx context.Context, id string) error {
	return Queries.DeleteEnvironment(ctx, id)
}
