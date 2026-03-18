package config

import (
	"fmt"
	"os"
)

var C Config

type Config struct {
	DatabaseURL        string
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string
	OAuthRedirectBase  string
	CookieSecure       bool
}

func Load() error {
	C = Config{
		DatabaseURL:        envDefault("DATABASE_URL", "postgres://uploy:password@localhost:5432/uploy"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		OAuthRedirectBase:  os.Getenv("OAUTH_REDIRECT_BASE_URL"),
		CookieSecure:       os.Getenv("COOKIE_SECURE") != "false",
	}

	if (C.GitHubClientID == "") != (C.GitHubClientSecret == "") {
		return fmt.Errorf("GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must both be set")
	}
	if (C.GoogleClientID == "") != (C.GoogleClientSecret == "") {
		return fmt.Errorf("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must both be set")
	}

	hasProvider := C.GitHubClientID != "" || C.GoogleClientID != ""
	if hasProvider && C.OAuthRedirectBase == "" {
		return fmt.Errorf("OAUTH_REDIRECT_BASE_URL is required when OAuth providers are configured")
	}

	return nil
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
