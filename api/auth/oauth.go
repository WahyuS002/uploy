package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/WahyuS002/uploy/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

const (
	stateExpiry          = 10 * time.Minute
	stateCleanupInterval = 2 * time.Minute
)

type ProviderUserInfo struct {
	ID    string
	Email string
}

var (
	oauthStates   = make(map[string]time.Time)
	oauthStatesMu sync.Mutex
)

func init() {
	go oauthStatesCleanupLoop()
}

func GitHubOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.C.GitHubClientID,
		ClientSecret: config.C.GitHubClientSecret,
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
		RedirectURL:  config.C.OAuthRedirectBase + "/api/auth/github/callback",
	}
}

func GoogleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.C.GoogleClientID,
		ClientSecret: config.C.GoogleClientSecret,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
		RedirectURL:  config.C.OAuthRedirectBase + "/api/auth/google/callback",
	}
}

func GenerateOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)

	oauthStatesMu.Lock()
	oauthStates[state] = time.Now().Add(stateExpiry)
	oauthStatesMu.Unlock()

	return state, nil
}

func ValidateOAuthState(state string) bool {
	oauthStatesMu.Lock()
	defer oauthStatesMu.Unlock()

	expiry, ok := oauthStates[state]
	if !ok {
		return false
	}
	delete(oauthStates, state)
	return time.Now().Before(expiry)
}

func oauthStatesCleanupLoop() {
	ticker := time.NewTicker(stateCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		oauthStatesMu.Lock()
		now := time.Now()
		for state, expiry := range oauthStates {
			if now.After(expiry) {
				delete(oauthStates, state)
			}
		}
		oauthStatesMu.Unlock()
	}
}

func FetchGitHubUser(ctx context.Context, cfg *oauth2.Config, code string) (ProviderUserInfo, error) {
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return ProviderUserInfo{}, fmt.Errorf("code exchange failed: %w", err)
	}

	client := cfg.Client(ctx, token)

	// Fetch user profile for ID only
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return ProviderUserInfo{}, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var ghUser struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return ProviderUserInfo{}, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Always resolve email from /user/emails to ensure it's verified
	email, err := fetchGitHubPrimaryEmail(client)
	if err != nil {
		return ProviderUserInfo{}, fmt.Errorf("failed to get verified email: %w", err)
	}

	return ProviderUserInfo{
		ID:    fmt.Sprintf("%d", ghUser.ID),
		Email: email,
	}, nil
}

func fetchGitHubPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Prefer primary+verified
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	// Fallback to any verified
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}

func FetchGoogleUser(ctx context.Context, cfg *oauth2.Config, code string) (ProviderUserInfo, error) {
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return ProviderUserInfo{}, fmt.Errorf("code exchange failed: %w", err)
	}

	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return ProviderUserInfo{}, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var gUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return ProviderUserInfo{}, fmt.Errorf("failed to decode user info: %w", err)
	}

	if gUser.Email == "" {
		return ProviderUserInfo{}, fmt.Errorf("no email returned from Google")
	}
	if !gUser.VerifiedEmail {
		return ProviderUserInfo{}, fmt.Errorf("Google email is not verified")
	}

	return ProviderUserInfo{
		ID:    gUser.ID,
		Email: gUser.Email,
	}, nil
}
