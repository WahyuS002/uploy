package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
)

const (
	CookieName    = "session"
	cookieMaxAge  = 7 * 24 * 60 * 60 // 7 days
	tokenByteSize = 32
)

func GenerateSessionToken() (string, error) {
	b := make([]byte, tokenByteSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isSecureCookie() bool {
	return os.Getenv("COOKIE_SECURE") != "false"
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
