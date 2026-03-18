package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/WahyuS002/uploy/config"
)

const (
	CookieName       = "session"
	cookieMaxAge     = 7 * 24 * 60 * 60 // 7 days
	tokenByteSize    = 32
	IdleTimeout      = 7 * 24 * time.Hour  // 7 days
	AbsoluteLifetime = 30 * 24 * time.Hour // 30 days
	RenewalThreshold = IdleTimeout / 2     // extend when remaining < 3.5 days
)

func GenerateSessionToken() (string, error) {
	b := make([]byte, tokenByteSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isSecureCookie() bool {
	return config.C.CookieSecure
}

func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   cookieMaxAge,
	})
}

func SetSessionCookieWithExpiry(w http.ResponseWriter, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
