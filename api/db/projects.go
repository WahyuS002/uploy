package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
)

// DefaultEnvironmentName is the name of the environment automatically created
// alongside every new project.
const DefaultEnvironmentName = "production"

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

// CreateProjectWithDefaultEnvironment creates a project and its default
// `production` environment atomically. If `name` is empty, a unique
// Railway-style `adjective-noun` name is generated for the workspace. If the
// environment insert fails the project insert is rolled back so callers never
// observe a project without its default environment.
func CreateProjectWithDefaultEnvironment(ctx context.Context, name, workspaceID string) (Project, Environment, error) {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return Project{}, Environment{}, err
	}
	defer tx.Rollback(ctx)

	q := Queries.WithTx(tx)

	// Serialize project-name allocation per workspace so two concurrent
	// create transactions cannot both observe the same candidate as free.
	// The advisory lock is held for the rest of the transaction and
	// released on commit/rollback. We always acquire it, even for explicit
	// client-supplied names, so an auto-generated name in a parallel tx
	// cannot collide with a manual insert that committed first.
	if err := q.LockWorkspaceProjectNames(ctx, workspaceID); err != nil {
		return Project{}, Environment{}, err
	}

	if name == "" {
		generated, err := generateUniqueProjectName(ctx, workspaceNameExists(q, workspaceID))
		if err != nil {
			return Project{}, Environment{}, err
		}
		name = generated
	}

	pr, err := q.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Name:        name,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return Project{}, Environment{}, err
	}

	er, err := q.CreateEnvironment(ctx, sqlcgen.CreateEnvironmentParams{
		Name:      DefaultEnvironmentName,
		ProjectID: pr.ID,
	})
	if err != nil {
		return Project{}, Environment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Project{}, Environment{}, err
	}

	return Project{
			ID: pr.ID, Name: pr.Name, WorkspaceID: pr.WorkspaceID,
			CreatedAt: pr.CreatedAt, UpdatedAt: pr.UpdatedAt,
		}, Environment{
			ID: er.ID, Name: er.Name, ProjectID: er.ProjectID,
			CreatedAt: er.CreatedAt, UpdatedAt: er.UpdatedAt,
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
