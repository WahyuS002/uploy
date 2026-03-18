package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/jackc/pgx/v5"
)

func GitHubLoginHandler(w http.ResponseWriter, r *http.Request) {
	cfg := auth.GitHubOAuthConfig()
	state, err := auth.GenerateOAuthState()
	if err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}
	http.Redirect(w, r, cfg.AuthCodeURL(state), http.StatusFound)
}

func GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	cfg := auth.GoogleOAuthConfig()
	state, err := auth.GenerateOAuthState()
	if err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}
	http.Redirect(w, r, cfg.AuthCodeURL(state), http.StatusFound)
}

func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !auth.ValidateOAuthState(state) {
		http.Redirect(w, r, "/login?error=invalid_state", http.StatusFound)
		return
	}

	errParam := r.URL.Query().Get("error")
	if errParam != "" {
		http.Redirect(w, r, "/login?error=oauth_denied", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	info, err := auth.FetchGitHubUser(r.Context(), auth.GitHubOAuthConfig(), code)
	if err != nil {
		log.Printf("GitHub OAuth error: %v", err)
		http.Redirect(w, r, "/login?error=oauth_failed", http.StatusFound)
		return
	}

	handleOAuthLogin(w, r, "github", info)
}

func GoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !auth.ValidateOAuthState(state) {
		http.Redirect(w, r, "/login?error=invalid_state", http.StatusFound)
		return
	}

	errParam := r.URL.Query().Get("error")
	if errParam != "" {
		http.Redirect(w, r, "/login?error=oauth_denied", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	info, err := auth.FetchGoogleUser(r.Context(), auth.GoogleOAuthConfig(), code)
	if err != nil {
		log.Printf("Google OAuth error: %v", err)
		http.Redirect(w, r, "/login?error=oauth_failed", http.StatusFound)
		return
	}

	handleOAuthLogin(w, r, "google", info)
}

func handleOAuthLogin(w http.ResponseWriter, r *http.Request, provider string, info auth.ProviderUserInfo) {
	ctx := r.Context()

	// 1. Check if OAuth identity already exists
	identity, err := db.GetOAuthIdentity(ctx, provider, info.ID)
	if err == nil {
		// Identity exists — login as that user
		loginExistingUser(w, r, identity.UserID)
		return
	}
	if err != pgx.ErrNoRows {
		log.Printf("OAuth identity lookup error: %v", err)
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	// 2. Check if email matches an existing user
	email := strings.ToLower(strings.TrimSpace(info.Email))
	user, err := db.GetUserByEmail(ctx, email)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("User email lookup error: %v", err)
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}
	if err == nil {
		// Link identity to existing user
		tx, err := db.Pool.Begin(ctx)
		if err != nil {
			http.Redirect(w, r, "/login?error=internal", http.StatusFound)
			return
		}
		defer tx.Rollback(ctx)

		if _, err := db.CreateOAuthIdentityTx(ctx, tx, user.ID, provider, info.ID, email); err != nil {
			http.Redirect(w, r, "/login?error=internal", http.StatusFound)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			http.Redirect(w, r, "/login?error=internal", http.StatusFound)
			return
		}

		loginExistingUser(w, r, user.ID)
		return
	}

	// 3. No account — create user, workspace, membership, and identity
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}
	defer tx.Rollback(ctx)

	newUser, err := db.CreateUserTx(ctx, tx, email, "")
	if err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	wsName := strings.Split(email, "@")[0]
	workspace, err := db.CreateWorkspaceTx(ctx, tx, wsName, newUser.ID)
	if err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	if _, err := db.CreateMembershipTx(ctx, tx, workspace.ID, newUser.ID, "owner"); err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	if _, err := db.CreateOAuthIdentityTx(ctx, tx, newUser.ID, provider, info.ID, email); err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	createSessionAndRedirect(w, r, newUser.ID, workspace.ID, "owner")
}

func loginExistingUser(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()

	user, err := db.GetUserByID(ctx, userID)
	if err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}
	if user.Status != "active" {
		http.Redirect(w, r, "/login?error=account_inactive", http.StatusFound)
		return
	}

	_ = db.DeleteUserSessions(ctx, userID)

	workspace, membership, err := db.GetUserFirstWorkspace(ctx, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Redirect(w, r, "/login?error=no_workspace", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	createSessionAndRedirect(w, r, userID, workspace.ID, membership.Role)
}

func createSessionAndRedirect(w http.ResponseWriter, r *http.Request, userID, workspaceID, role string) {
	token, err := auth.GenerateSessionToken()
	if err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := db.CreateSession(r.Context(), token, userID, workspaceID, role, expiresAt); err != nil {
		http.Redirect(w, r, "/login?error=internal", http.StatusFound)
		return
	}

	auth.SetSessionCookie(w, token)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}
