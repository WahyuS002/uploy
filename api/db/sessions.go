package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type Session struct {
	Token         string    `json:"token"`
	UserID        string    `json:"user_id"`
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceRole string    `json:"workspace_role"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func sessionFromGen(s sqlcgen.Session) Session {
	return Session{
		Token:         s.Token,
		UserID:        s.UserID,
		WorkspaceID:   s.WorkspaceID,
		WorkspaceRole: s.WorkspaceRole,
		CreatedAt:     s.CreatedAt,
		ExpiresAt:     s.ExpiresAt,
	}
}

func CreateSession(ctx context.Context, token, userID, workspaceID, workspaceRole string, expiresAt time.Time) error {
	return Queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		Token:         token,
		UserID:        userID,
		WorkspaceID:   workspaceID,
		WorkspaceRole: workspaceRole,
		ExpiresAt:     expiresAt,
	})
}

func GetSession(ctx context.Context, token string) (Session, error) {
	s, err := Queries.GetSession(ctx, token)
	if err != nil {
		return Session{}, err
	}
	return sessionFromGen(s), nil
}

func ExtendSession(ctx context.Context, token string, idleTimeout, absoluteLifetime time.Duration) (time.Time, error) {
	return Queries.ExtendSession(ctx, sqlcgen.ExtendSessionParams{
		Token: token,
		IdleTimeout: pgtype.Interval{
			Microseconds: int64(idleTimeout / time.Microsecond),
			Valid:        true,
		},
		AbsoluteLifetime: pgtype.Interval{
			Microseconds: int64(absoluteLifetime / time.Microsecond),
			Valid:        true,
		},
	})
}

func DeleteSession(ctx context.Context, token string) error {
	return Queries.DeleteSession(ctx, token)
}

func DeleteUserSessions(ctx context.Context, userID string) error {
	return Queries.DeleteUserSessions(ctx, userID)
}
