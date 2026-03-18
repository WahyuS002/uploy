package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/respond"
	"github.com/jackc/pgx/v5/pgconn"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		respond.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email format"})
		return
	}
	if len(req.Password) < 8 {
		respond.JSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx := r.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer tx.Rollback(ctx)

	user, err := db.CreateUserTx(ctx, tx, email, hash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == db.PgUniqueViolation {
			respond.JSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	wsName := strings.Split(email, "@")[0]
	workspace, err := db.CreateWorkspaceTx(ctx, tx, wsName, user.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	_, err = db.CreateMembershipTx(ctx, tx, workspace.ID, user.ID, "owner")
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, err := auth.GenerateSessionToken()
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := db.CreateSession(ctx, token, user.ID, workspace.ID, "owner", expiresAt); err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	auth.SetSessionCookie(w, token)
	respond.JSON(w, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":            user.ID,
			"email":         user.Email,
			"platform_role": user.PlatformRole,
		},
		"workspace": map[string]any{
			"id":   workspace.ID,
			"name": workspace.Name,
		},
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		respond.JSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	if auth.IsLoginRateLimited(email) {
		respond.JSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many login attempts, please try again later"})
		return
	}

	ctx := r.Context()
	user, err := db.GetUserByEmail(ctx, email)
	if err != nil {
		respond.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	ok, err := auth.VerifyPassword(user.PasswordHash, req.Password)
	if err != nil || !ok {
		auth.RecordFailedLogin(email)
		respond.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	if user.Status != "active" {
		respond.JSON(w, http.StatusForbidden, map[string]string{"error": "account is not active"})
		return
	}

	_ = db.DeleteUserSessions(ctx, user.ID)

	workspace, membership, err := db.GetUserFirstWorkspace(ctx, user.ID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, err := auth.GenerateSessionToken()
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := db.CreateSession(ctx, token, user.ID, workspace.ID, membership.Role, expiresAt); err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	auth.ClearFailedLogins(email)
	auth.SetSessionCookie(w, token)
	respond.JSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":            user.ID,
			"email":         user.Email,
			"platform_role": user.PlatformRole,
		},
		"workspace": map[string]any{
			"id":   workspace.ID,
			"name": workspace.Name,
			"role": membership.Role,
		},
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.CookieName)
	if err != nil {
		respond.JSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	_ = db.DeleteSession(r.Context(), cookie.Value)
	auth.ClearSessionCookie(w)
	respond.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	sc, ok := auth.GetSessionContext(r)
	if !ok {
		respond.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	ctx := r.Context()
	user, err := db.GetUserByID(ctx, sc.UserID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	workspace, err := db.GetWorkspace(ctx, sc.WorkspaceID)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	respond.JSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":            user.ID,
			"email":         user.Email,
			"platform_role": user.PlatformRole,
		},
		"workspace": map[string]any{
			"id":   workspace.ID,
			"name": workspace.Name,
			"role": sc.WorkspaceRole,
		},
	})
}


