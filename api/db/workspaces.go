package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OwnerUserID string    `json:"owner_user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func workspaceFromGen(w sqlcgen.Workspace) Workspace {
	return Workspace{
		ID:          w.ID,
		Name:        w.Name,
		OwnerUserID: w.OwnerUserID,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}

func CreateWorkspaceTx(ctx context.Context, tx pgx.Tx, name, ownerUserID string) (Workspace, error) {
	w, err := sqlcgen.New(tx).CreateWorkspace(ctx, sqlcgen.CreateWorkspaceParams{
		Name:        name,
		OwnerUserID: ownerUserID,
	})
	if err != nil {
		return Workspace{}, err
	}
	return workspaceFromGen(w), nil
}

func GetWorkspace(ctx context.Context, id string) (Workspace, error) {
	w, err := Queries.GetWorkspace(ctx, id)
	if err != nil {
		return Workspace{}, err
	}
	return workspaceFromGen(w), nil
}
