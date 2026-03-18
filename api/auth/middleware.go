package auth

import (
	"context"
	"net/http"

	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/respond"
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
			respond.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		session, err := db.GetSession(r.Context(), cookie.Value)
		if err != nil {
			ClearSessionCookie(w)
			respond.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired session"})
			return
		}

		sc := SessionContext{
			UserID:        session.UserID,
			WorkspaceID:   session.WorkspaceID,
			WorkspaceRole: session.WorkspaceRole,
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
				respond.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}

			for _, role := range roles {
				if sc.WorkspaceRole == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			respond.JSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		})
	}
}

