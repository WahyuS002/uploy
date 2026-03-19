package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type Membership struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

func membershipFromGen(m sqlcgen.WorkspaceMembership) Membership {
	return Membership{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		UserID:      m.UserID,
		Role:        m.Role,
		CreatedAt:   m.CreatedAt,
	}
}

func CreateMembershipTx(ctx context.Context, tx pgx.Tx, workspaceID, userID, role string) (Membership, error) {
	m, err := sqlcgen.New(tx).CreateMembership(ctx, sqlcgen.CreateMembershipParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
	})
	if err != nil {
		return Membership{}, err
	}
	return membershipFromGen(m), nil
}

func GetMembership(ctx context.Context, workspaceID, userID string) (Membership, error) {
	m, err := Queries.GetMembership(ctx, sqlcgen.GetMembershipParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		return Membership{}, err
	}
	return membershipFromGen(m), nil
}

func GetUserFirstWorkspace(ctx context.Context, userID string) (Workspace, Membership, error) {
	row, err := Queries.GetUserFirstWorkspace(ctx, userID)
	if err != nil {
		return Workspace{}, Membership{}, err
	}
	return Workspace{
			ID:          row.WID,
			Name:        row.WName,
			OwnerUserID: row.WOwnerUserID,
			CreatedAt:   row.WCreatedAt,
			UpdatedAt:   row.WUpdatedAt,
		}, Membership{
			ID:          row.WmID,
			WorkspaceID: row.WmWorkspaceID,
			UserID:      row.WmUserID,
			Role:        row.WmRole,
			CreatedAt:   row.WmCreatedAt,
		}, nil
}
