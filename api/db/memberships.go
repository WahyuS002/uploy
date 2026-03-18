package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Membership struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

func CreateMembershipTx(ctx context.Context, tx pgx.Tx, workspaceID, userID, role string) (Membership, error) {
	id := fmt.Sprintf("wm-%d", time.Now().UnixNano())
	var m Membership
	err := tx.QueryRow(ctx,
		`INSERT INTO workspace_memberships (id, workspace_id, user_id, role) VALUES ($1, $2, $3, $4)
		 RETURNING id, workspace_id, user_id, role, created_at`,
		id, workspaceID, userID, role,
	).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.CreatedAt)
	return m, err
}

func GetMembership(ctx context.Context, workspaceID, userID string) (Membership, error) {
	var m Membership
	err := Pool.QueryRow(ctx,
		`SELECT id, workspace_id, user_id, role, created_at
		 FROM workspace_memberships WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.CreatedAt)
	return m, err
}

func GetUserFirstWorkspace(ctx context.Context, userID string) (Workspace, Membership, error) {
	var w Workspace
	var m Membership
	err := Pool.QueryRow(ctx,
		`SELECT w.id, w.name, w.owner_user_id, w.created_at, w.updated_at,
		        wm.id, wm.workspace_id, wm.user_id, wm.role, wm.created_at
		 FROM workspace_memberships wm
		 JOIN workspaces w ON w.id = wm.workspace_id
		 WHERE wm.user_id = $1
		 ORDER BY wm.created_at ASC
		 LIMIT 1`,
		userID,
	).Scan(&w.ID, &w.Name, &w.OwnerUserID, &w.CreatedAt, &w.UpdatedAt,
		&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.CreatedAt)
	return w, m, err
}
