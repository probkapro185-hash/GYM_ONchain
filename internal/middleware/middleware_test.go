package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowedOrigin(t *testing.T) {
	called := false
	h := CORS([]string{"https://gym.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/test", nil)
	req.Header.Set("Origin", "https://gym.example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK || !called {
		t.Fatalf("status=%d called=%v, want 200/true", rr.Code, called)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://gym.example.com" {
		t.Fatalf("allow-origin=%q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow-credentials=%q", got)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	called := false
	h := CORS([]string{"https://gym.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/test", nil)
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d called=%v, want 403/false", rr.Code, called)
	}
}

func TestCORSWildcardDoesNotEnableCredentials(t *testing.T) {
	h := CORS([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/test", nil)
	req.Header.Set("Origin", "https://any.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin=%q, want *", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("wildcard must not enable credentials, got %q", got)
	}
}

func TestCORSSameOriginIsAlwaysAllowed(t *testing.T) {
	called := false
	h := CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK || !called {
		t.Fatalf("status=%d called=%v, want 200/true", rr.Code, called)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("same-origin request does not need CORS response header, got %q", got)
	}
}
