package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OwnerUserID string    `json:"owner_user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func CreateWorkspaceTx(ctx context.Context, tx pgx.Tx, name, ownerUserID string) (Workspace, error) {
	var w Workspace
	err := tx.QueryRow(ctx,
		`INSERT INTO workspaces (name, owner_user_id) VALUES ($1, $2)
		 RETURNING id, name, owner_user_id, created_at, updated_at`,
		name, ownerUserID,
	).Scan(&w.ID, &w.Name, &w.OwnerUserID, &w.CreatedAt, &w.UpdatedAt)
	return w, err
}

func GetWorkspace(ctx context.Context, id string) (Workspace, error) {
	var w Workspace
	err := Pool.QueryRow(ctx,
		`SELECT id, name, owner_user_id, created_at, updated_at
		 FROM workspaces WHERE id = $1`,
		id,
	).Scan(&w.ID, &w.Name, &w.OwnerUserID, &w.CreatedAt, &w.UpdatedAt)
	return w, err
}
