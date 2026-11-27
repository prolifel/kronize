package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"

	"github.com/go-chi/chi/v5"
)

func setupSettingsRouter(t *testing.T) (*chi.Mux, *dbTestUsers, string) {
	t.Helper()
	d := setupHandlerTestDB(t)
	secret := "test-secret"

	user := createHandlerTestUser(t, d, "regularuser", "password")
	hash, err := auth.HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := db.CreateUser(d, model.CreateUserRequest{Username: "adminuser", Role: "admin"}, hash)
	if err != nil {
		t.Fatal(err)
	}

	// Seed a legacy secret row directly in SQLite to verify it is never exposed.
	if err := db.SetSetting(d, "registry_password", "SECRET-PULL-TOKEN"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSetting(d, "smtp_password", "mailpass123"); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(auth.Middleware(secret))
	r.Group(func(r chi.Router) {
		r.Use(auth.AdminOnly)
		r.Get("/api/settings", GetSettings(d))
		r.Put("/api/settings", UpdateSettings(d))
	})

	return r, &dbTestUsers{user: user, admin: admin}, secret
}

type dbTestUsers struct {
	user  *model.User
	admin *model.User
}

func TestSettingsAuthorizationAndMasking(t *testing.T) {
	r, users, secret := setupSettingsRouter(t)

	// 1. Unauthenticated -> 401
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/settings", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s /api/settings = %d, want 401", method, rr.Code)
		}
	}

	// 2. Low-privilege role=user -> 403 on both GET and PUT
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, users.user.ID, users.user.Username, "user", http.MethodGet, "/api/settings", ""))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("user GET /api/settings = %d, want 403", rr.Code)
	}

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, users.user.ID, users.user.Username, "user", http.MethodPut, "/api/settings", `[{"key":"teams_webhook_url","value":"https://example.com/hook"}]`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("user PUT /api/settings = %d, want 403", rr.Code)
	}

	// 3. Admin GET omits legacy/secret rows and returns [] instead of null
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, users.admin.ID, users.admin.Username, "admin", http.MethodGet, "/api/settings", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("admin GET /api/settings = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "[]" {
		t.Fatalf("admin GET /api/settings = %s, want [] (legacy secrets omitted)", rr.Body.String())
	}

	// 4. Admin PUT rejects arbitrary/unallowlisted keys, oversized values, and invalid URLs
	badPayloads := []string{
		`[{"key":"","value":"x"}]`,
		`[{"key":"pwn_from_user","value":"x"}]`,
		`[{"key":"registry_password","value":"secret"}]`,
		`[{"key":"site_name","value":"Pwned"}]`,
		`[{"key":"teams_webhook_url","value":"javascript:alert(1)"}]`,
		`[{"key":"teams_webhook_url","value":"https://example.com/` + strings.Repeat("a", 2100) + `"}]`,
	}
	for _, payload := range badPayloads {
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, authenticatedRequest(secret, users.admin.ID, users.admin.Username, "admin", http.MethodPut, "/api/settings", payload))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("admin PUT invalid payload %s = %d, want 400: %s", payload, rr.Code, rr.Body.String())
		}
	}

	// 5. Admin PUT valid teams_webhook_url -> 200 and value is masked in response
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, users.admin.ID, users.admin.Username, "admin", http.MethodPut, "/api/settings", `[{"key":"teams_webhook_url","value":"https://hooks.example.com/secret-token"}]`))
	if rr.Code != http.StatusOK {
		t.Fatalf("admin PUT valid = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "secret-token") {
		t.Fatalf("response leaked cleartext webhook token: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"value":"********"`) {
		t.Fatalf("expected masked value in response, got: %s", rr.Body.String())
	}
}
