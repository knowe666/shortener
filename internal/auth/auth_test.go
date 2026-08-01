package auth

import (
	"net/http/httptest"
	"testing"
)

func TestNewAuthenticator_UsesInjectedSecret(t *testing.T) {
	auth, err := NewAuthenticator("test-secret")
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}

	w := httptest.NewRecorder()
	auth.SetUserCookie(w, "user-123")

	r := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range w.Result().Cookies() {
		r.AddCookie(cookie)
	}

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		t.Fatalf("GetUserIDFromCookie() error = %v", err)
	}

	if userID != "user-123" {
		t.Fatalf("GetUserIDFromCookie() = %q, want %q", userID, "user-123")
	}
}
