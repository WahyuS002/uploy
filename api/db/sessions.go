package db

import (
	"context"
	"time"
)

type Session struct {
	Token         string    `json:"token"`
	UserID        string    `json:"user_id"`
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceRole string    `json:"workspace_role"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func CreateSession(ctx context.Context, token, userID, workspaceID, workspaceRole string, expiresAt time.Time) error {
	_, err := Pool.Exec(ctx,
		`INSERT INTO sessions (token, user_id, workspace_id, workspace_role, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		token, userID, workspaceID, workspaceRole, expiresAt,
	)
	return err
}

func GetSession(ctx context.Context, token string) (Session, error) {
	var s Session
	err := Pool.QueryRow(ctx,
		`SELECT token, user_id, workspace_id, workspace_role, created_at, expires_at
		 FROM sessions WHERE token = $1 AND expires_at > NOW()`,
		token,
	).Scan(&s.Token, &s.UserID, &s.WorkspaceID, &s.WorkspaceRole, &s.CreatedAt, &s.ExpiresAt)
	return s, err
}

func DeleteSession(ctx context.Context, token string) error {
	_, err := Pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := Pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}
