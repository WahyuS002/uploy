package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/jackc/pgx/v5"
)

type contextKey string

const sessionContextKey contextKey = "session"

type SessionContext struct {
	UserID        string
	WorkspaceID   string
	WorkspaceRole string
}

func GetSessionContext(r *http.Request) (SessionContext, bool) {
	sc, ok := r.Context().Value(sessionContextKey).(SessionContext)
	return sc, ok
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			respond.JSON(w, http.StatusUnauthorized, gen.ErrorResponse{Error: "authentication required"})
			return
		}

		session, err := db.GetSession(r.Context(), cookie.Value)
		if err != nil {
			ClearSessionCookie(w)
			respond.JSON(w, http.StatusUnauthorized, gen.ErrorResponse{Error: "invalid or expired session"})
			return
		}

		// Sliding session: extend if remaining time < threshold
		remaining := time.Until(session.ExpiresAt)
		if remaining < RenewalThreshold {
			if newExpiry, err := db.ExtendSession(r.Context(), cookie.Value, IdleTimeout, AbsoluteLifetime); err == nil {
				SetSessionCookieWithExpiry(w, cookie.Value, newExpiry)
			}
		}

		membership, err := db.GetMembership(r.Context(), session.WorkspaceID, session.UserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				_ = db.DeleteSession(r.Context(), cookie.Value)
				ClearSessionCookie(w)
				respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "no longer a member of this workspace"})
				return
			}
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "internal error"})
			return
		}

		sc := SessionContext{
			UserID:        session.UserID,
			WorkspaceID:   session.WorkspaceID,
			WorkspaceRole: membership.Role,
		}
		ctx := context.WithValue(r.Context(), sessionContextKey, sc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sc, ok := GetSessionContext(r)
			if !ok {
				respond.JSON(w, http.StatusUnauthorized, gen.ErrorResponse{Error: "authentication required"})
				return
			}

			for _, role := range roles {
				if sc.WorkspaceRole == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			respond.JSON(w, http.StatusForbidden, gen.ErrorResponse{Error: "insufficient permissions"})
		})
	}
}

