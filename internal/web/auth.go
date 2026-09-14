package web

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

const authCookie = "ns_auth"

func cookieValue(password string) string {
	sum := sha256.Sum256([]byte("ns-apns|" + password))
	return hex.EncodeToString(sum[:])
}

func passwordOK(got, want string) bool {
	if want == "" {
		return true
	}
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func authorized(r *http.Request, password string) bool {
	if strings.TrimSpace(password) == "" {
		return true
	}
	c, err := r.Cookie(authCookie)
	if err != nil || c.Value == "" {
		return false
	}
	want := cookieValue(password)
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(want)) == 1
}

func setAuthCookie(w http.ResponseWriter, password string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookie,
		Value:    cookieValue(password),
		Path:     "/",
		MaxAge:   30 * 24 * 3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
