package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizedEmptyPassword(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if !authorized(r, "") {
		t.Fatal("empty password should skip auth")
	}
}

func TestCookieMatchesPassword(t *testing.T) {
	w := httptest.NewRecorder()
	setAuthCookie(w, "secret")
	res := w.Result()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range res.Cookies() {
		req.AddCookie(c)
	}
	if !authorized(req, "secret") {
		t.Fatal("expected ok")
	}
	if authorized(req, "other") {
		t.Fatal("old cookie must fail after password change")
	}
}
