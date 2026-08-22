package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"
)

func setupHandlerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	f, err := os.CreateTemp("", "kronize-handler-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.Close()

	d, err := db.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	if err := db.Migrate(d); err != nil {
		t.Fatal(err)
	}
	return d
}

func createHandlerTestUser(t *testing.T, d *sql.DB, username, password string) *model.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	u, err := db.CreateUser(d, model.CreateUserRequest{Username: username}, hash)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func authenticatedRequest(secret string, userID int64, username, role, method, target, body string) *http.Request {
	token, err := auth.GenerateToken(secret, userID, username, role)
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestChangePasswordRejectsShortNewPassword(t *testing.T) {
	d := setupHandlerTestDB(t)
	u := createHandlerTestUser(t, d, "shortpwd", "correct-password")

	req := authenticatedRequest(
		"secret", u.ID, u.Username, u.Role,
		http.MethodPost,
		"/api/auth/change-password",
		`{"current_password":"correct-password","new_password":"123"}`,
	)
	rr := httptest.NewRecorder()
	auth.Middleware("secret")(ChangePassword(d)).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}


func TestChangePasswordClearsMustChangePassword(t *testing.T) {
	d := setupHandlerTestDB(t)
	u := createHandlerTestUser(t, d, "validpwd", "old-password")
	if !u.MustChangePassword {
		t.Fatalf("new user MustChangePassword = false, want true")
	}

	req := authenticatedRequest(
		"secret", u.ID, u.Username, u.Role,
		http.MethodPost,
		"/api/auth/change-password",
		`{"current_password":"old-password","new_password":"new-password"}`,
	)
	rr := httptest.NewRecorder()
	auth.Middleware("secret")(ChangePassword(d)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	got, err := db.GetUserByID(d, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.MustChangePassword {
		t.Fatalf("MustChangePassword = true, want false after password change")
	}
	if !auth.CheckPassword("new-password", got.PasswordHash) {
		t.Fatalf("stored password hash does not match new password")
	}
}


func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func TestRequirePasswordChangedBlocksProtectedRoute(t *testing.T) {
	d := setupHandlerTestDB(t)
	u := createHandlerTestUser(t, d, "blockeduser", "correct-password")

	req := authenticatedRequest(
		"secret", u.ID, u.Username, u.Role,
		http.MethodGet,
		"/api/jobs",
		"",
	)
	rr := httptest.NewRecorder()
	chain := auth.Middleware("secret")(RequirePasswordChanged(d)(http.HandlerFunc(okHandler)))
	chain.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "password change required") {
		t.Fatalf("body = %s, want password change required", rr.Body.String())
	}
}

func TestRequirePasswordChangedAllowsRequiredAuthRoutes(t *testing.T) {
	d := setupHandlerTestDB(t)
	u := createHandlerTestUser(t, d, "alloweduser", "correct-password")

	paths := []string{
		"/api/auth/me",
		"/api/auth/change-password",
		"/api/auth/logout",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := authenticatedRequest(
				"secret", u.ID, u.Username, u.Role,
				http.MethodPost,
				path,
				"",
			)
			rr := httptest.NewRecorder()
			chain := auth.Middleware("secret")(RequirePasswordChanged(d)(http.HandlerFunc(okHandler)))
			chain.ServeHTTP(rr, req)

			if rr.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusNoContent, rr.Body.String())
			}
		})
	}
}

func TestRequirePasswordChangedAllowsAfterClear(t *testing.T) {
	d := setupHandlerTestDB(t)
	u := createHandlerTestUser(t, d, "cleareduser", "correct-password")

	if err := db.ClearMustChangePassword(d, u.ID); err != nil {
		t.Fatal(err)
	}

	req := authenticatedRequest(
		"secret", u.ID, u.Username, u.Role,
		http.MethodGet,
		"/api/jobs",
		"",
	)
	rr := httptest.NewRecorder()
	chain := auth.Middleware("secret")(RequirePasswordChanged(d)(http.HandlerFunc(okHandler)))
	chain.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
}

func TestRequirePasswordChangedAllowsLogin(t *testing.T) {
	d := setupHandlerTestDB(t)

	// login is unauthenticated — no token, no user in context
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	rr := httptest.NewRecorder()
	chain := auth.Middleware("secret")(RequirePasswordChanged(d)(http.HandlerFunc(okHandler)))
	chain.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("login must pass through middleware, got %d: %s", rr.Code, rr.Body.String())
	}
}
