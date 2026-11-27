package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"

	"github.com/go-chi/chi/v5"
)

func TestRunnerImageHandlerRejectsSSRFRefs(t *testing.T) {
	d := setupHandlerTestDB(t)
	secret := "test-secret"
	hash, err := auth.HashPassword("password-123")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := db.CreateUser(d, model.CreateUserRequest{Username: "adminuser", Role: "admin"}, hash)
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(auth.Middleware(secret), auth.AdminOnly)
	r.Post("/api/runners", CreateRunnerImage(d))
	r.Put("/api/runners/{id}", UpdateRunnerImage(d))

	for _, img := range []string{
		"127.0.0.1:8080/x:v1",
		"host.docker.internal:8080/x:v1",
		"172.19.0.1:2375/x:v1",
		"localhost:5000/x:v1",
	} {
		rr := httptest.NewRecorder()
		body := `{"name":"ssrf","image":"` + img + `","description":"test"}`
		r.ServeHTTP(rr, authenticatedRequest(secret, admin.ID, admin.Username, "admin", http.MethodPost, "/api/runners", body))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("POST /api/runners with %s = %d, want 400: %s", img, rr.Code, rr.Body.String())
		}
	}

	// Valid image creation succeeds.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, admin.ID, admin.Username, "admin", http.MethodPost, "/api/runners", `{"name":"valid","image":"cr.prolifel.com/kronize/python-runner:v1","description":"ok"}`))
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /api/runners valid = %d, want 201: %s", rr.Code, rr.Body.String())
	}

	// Updating runner image to SSRF ref is also rejected.
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, admin.ID, admin.Username, "admin", http.MethodPut, "/api/runners/1", `{"image":"host.docker.internal:22/x:v1"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("PUT /api/runners/1 SSRF = %d, want 400: %s", rr.Code, rr.Body.String())
	}
}
