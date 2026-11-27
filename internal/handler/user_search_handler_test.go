package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kronize/internal/auth"
)

func TestSearchUsersRejectsEmptyAndWildcardQueries(t *testing.T) {
	d := setupHandlerTestDB(t)
	secret := "test-secret"
	u := createHandlerTestUser(t, d, "alice", "password-123")
	createHandlerTestUser(t, d, "bob", "password-123")
	createHandlerTestUser(t, d, "admin", "password-123")

	h := auth.Middleware(secret)(SearchUsers(d))

	// Absent, empty, whitespace, single-char, and wildcard-only queries must be 400.
	for _, target := range []string{
		"/api/users/search",
		"/api/users/search?q=",
		"/api/users/search?q=%20%20%20",
		"/api/users/search?q=a",
		"/api/users/search?q=_",
		"/api/users/search?q=%25",
		"/api/users/search?q=a%25",
		"/api/users/search?q=__",
	} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, authenticatedRequest(secret, u.ID, u.Username, "user", http.MethodGet, target, ""))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("GET %s = %d, want 400: %s", target, rr.Code, rr.Body.String())
		}
	}

	// Valid >=2 char query returns matching user only.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authenticatedRequest(secret, u.ID, u.Username, "user", http.MethodGet, "/api/users/search?q=al", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET ?q=al = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"alice"`) || strings.Contains(rr.Body.String(), `"bob"`) {
		t.Fatalf("unexpected search result: %s", rr.Body.String())
	}
}

func TestSearchUsersRateLimiting(t *testing.T) {
	d := setupHandlerTestDB(t)
	secret := "test-secret"
	u := createHandlerTestUser(t, d, "alice", "password-123")
	h := auth.Middleware(secret)(SearchUsers(d))

	for i := 0; i < searchRateLimit; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, authenticatedRequest(secret, u.ID, u.Username, "user", http.MethodGet, "/api/users/search?q=al", ""))
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i+1, rr.Code)
		}
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authenticatedRequest(secret, u.ID, u.Username, "user", http.MethodGet, "/api/users/search?q=al", ""))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited request = %d, want 429: %s", rr.Code, rr.Body.String())
	}
}
