package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID string    `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func CreateProject(ctx context.Context, name, workspaceID string) (Project, error) {
	r, err := Queries.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Name:        name,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return Project{}, err
	}
	return Project{
		ID: r.ID, Name: r.Name, WorkspaceID: r.WorkspaceID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func GetProjectByID(ctx context.Context, id string) (Project, error) {
	r, err := Queries.GetProjectByID(ctx, id)
	if err != nil {
		return Project{}, err
	}
	return Project{
		ID: r.ID, Name: r.Name, WorkspaceID: r.WorkspaceID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func ListProjectsByWorkspace(ctx context.Context, workspaceID string) ([]Project, error) {
	rows, err := Queries.ListProjectsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	projects := make([]Project, len(rows))
	for i, r := range rows {
		projects[i] = Project{
			ID: r.ID, Name: r.Name, WorkspaceID: r.WorkspaceID,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return projects, nil
}

func UpdateProject(ctx context.Context, id, name string) (Project, error) {
	r, err := Queries.UpdateProject(ctx, sqlcgen.UpdateProjectParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return Project{}, err
	}
	return Project{
		ID: r.ID, Name: r.Name, WorkspaceID: r.WorkspaceID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func DeleteProject(ctx context.Context, id string) error {
	return Queries.DeleteProject(ctx, id)
}
