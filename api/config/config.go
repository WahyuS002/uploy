package config

import (
	"fmt"
	"os"

	"github.com/WahyuS002/uploy/crypto"
)

var C Config

type Config struct {
	DatabaseURL        string
	EncryptionKey      string
	MonitoringImage    string
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
		EncryptionKey:      os.Getenv("ENCRYPTION_KEY"),
		MonitoringImage:    envDefault("MONITORING_IMAGE", "ghcr.io/wahyus002/uploy-monitor:0.1.0"),
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

	if C.EncryptionKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required (64 hex chars for AES-256)")
	}
	if !pinnedImage(C.MonitoringImage) {
		return fmt.Errorf("MONITORING_IMAGE must use a version tag or digest")
	}
	if err := crypto.Init(C.EncryptionKey); err != nil {
		return err
	}

	return nil
}

func pinnedImage(image string) bool {
	for index := len(image) - 1; index >= 0; index-- {
		switch image[index] {
		case '@':
			return index < len(image)-1
		case ':':
			return index < len(image)-1 && image[index+1:] != "latest"
		case '/':
			return false
		}
	}
	return false
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
