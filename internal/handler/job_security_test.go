package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/logstream"
	"kronize/internal/model"
	"kronize/internal/scheduler"

	"github.com/go-chi/chi/v5"
)

func setupJobHandlerRouter(t *testing.T) (*chi.Mux, int64, *model.User, *model.User, string) {
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
	imgID := seedHandlerRunnerImage(t, d)

	hub := logstream.New(d)
	sched := scheduler.New(d, t.TempDir(), hub)

	r := chi.NewRouter()
	r.Use(auth.Middleware(secret))
	r.Post("/api/jobs", CreateJob(d, sched, t.TempDir()))
	r.Put("/api/jobs/{id}", UpdateJob(d, sched, t.TempDir()))

	return r, imgID, user, admin, secret
}

func TestCreateJobHostMappingsAndEnvRestrictions(t *testing.T) {
	r, imgID, user, admin, secret := setupJobHandlerRouter(t)
	imgStr := itoa(imgID)

	// 1. Regular user trying to mount '/' -> 400 Bad Request
	rr := httptest.NewRecorder()
	body := `{"name":"poc","cron_expression":"0 0 1 * *","image_id":` + imgStr + `,"python_code":"print(1)","env_vars":"{}","host_mappings":"[{\"host\":\"/\",\"container\":\"/hostroot\",\"read_only\":false}]"}`
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPost, "/api/jobs", body))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("user mounting '/' got %d, want 400: %s", rr.Code, rr.Body.String())
	}

	// 2. Regular user trying to mount even a safe path -> 403 Forbidden
	rr = httptest.NewRecorder()
	body = `{"name":"poc","cron_expression":"0 0 1 * *","image_id":` + imgStr + `,"python_code":"print(1)","env_vars":"{}","host_mappings":"[{\"host\":\"/mnt/shared\",\"container\":\"/workspace\",\"read_only\":true}]"}`
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPost, "/api/jobs", body))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("user mounting safe path got %d, want 403: %s", rr.Code, rr.Body.String())
	}

	// 3. Regular user trying to set custom env_vars -> 403 Forbidden
	rr = httptest.NewRecorder()
	body = `{"name":"poc","cron_expression":"0 0 1 * *","image_id":` + imgStr + `,"python_code":"print(1)","env_vars":"{\"FOO\":\"bar\"}","host_mappings":"[]"}`
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPost, "/api/jobs", body))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("user setting env_vars got %d, want 403: %s", rr.Code, rr.Body.String())
	}

	// 4. Regular user with empty host_mappings and empty env_vars -> 201 Created
	rr = httptest.NewRecorder()
	body = `{"name":"normal","cron_expression":"0 0 1 * *","image_id":` + imgStr + `,"python_code":"print(1)","env_vars":"{}","host_mappings":"[]"}`
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPost, "/api/jobs", body))
	if rr.Code != http.StatusCreated {
		t.Fatalf("user creating normal job got %d, want 201: %s", rr.Code, rr.Body.String())
	}

	// 5. Admin trying to mount '/' or docker.sock -> 400 Bad Request
	for _, hostPath := range []string{"/", "/var/run/docker.sock", "/etc"} {
		rr = httptest.NewRecorder()
		body = `{"name":"admin-bad","cron_expression":"0 0 1 * *","image_id":` + imgStr + `,"python_code":"print(1)","env_vars":"{}","host_mappings":"[{\"host\":\"` + hostPath + `\",\"container\":\"/hostroot\",\"read_only\":true}]"}`
		r.ServeHTTP(rr, authenticatedRequest(secret, admin.ID, admin.Username, "admin", http.MethodPost, "/api/jobs", body))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("admin mounting %s got %d, want 400: %s", hostPath, rr.Code, rr.Body.String())
		}
	}

	// 6. Admin with safe host_mappings and env_vars -> 201 Created
	rr = httptest.NewRecorder()
	body = `{"name":"admin-ok","cron_expression":"0 0 1 * *","image_id":` + imgStr + `,"python_code":"print(1)","env_vars":"{\"APP_ENV\":\"prod\"}","host_mappings":"[{\"host\":\"/mnt/shared\",\"container\":\"/workspace\",\"read_only\":true}]"}`
	r.ServeHTTP(rr, authenticatedRequest(secret, admin.ID, admin.Username, "admin", http.MethodPost, "/api/jobs", body))
	if rr.Code != http.StatusCreated {
		t.Fatalf("admin creating safe job got %d, want 201: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateJobHostMappingsAndEnvRestrictions(t *testing.T) {
	r, imgID, user, _, secret := setupJobHandlerRouter(t)
	imgStr := itoa(imgID)

	// Create normal job as user (id=1).
	rr := httptest.NewRecorder()
	body := `{"name":"normal","cron_expression":"0 0 1 * *","image_id":` + imgStr + `,"python_code":"print(1)","env_vars":"{}","host_mappings":"[]"}`
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPost, "/api/jobs", body))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create job got %d: %s", rr.Code, rr.Body.String())
	}

	// User updating python_code with empty host_mappings/env_vars -> 200 OK
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPut, "/api/jobs/1", `{"python_code":"print(2)","host_mappings":"[]","env_vars":"{}"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("user updating normal fields got %d, want 200: %s", rr.Code, rr.Body.String())
	}

	// User updating host_mappings to safe path -> 403 Forbidden
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPut, "/api/jobs/1", `{"host_mappings":"[{\"host\":\"/mnt/shared\",\"container\":\"/workspace\",\"read_only\":true}]"}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("user updating host_mappings got %d, want 403: %s", rr.Code, rr.Body.String())
	}

	// User updating env_vars -> 403 Forbidden
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, user.ID, user.Username, "user", http.MethodPut, "/api/jobs/1", `{"env_vars":"{\"SECRET\":\"1\"}"}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("user updating env_vars got %d, want 403: %s", rr.Code, rr.Body.String())
	}
}
