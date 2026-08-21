package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"

	"github.com/go-chi/chi/v5"
)

func seedHandlerRunnerImage(t *testing.T, d *sql.DB) int64 {
	t.Helper()
	img, err := db.CreateRunnerImage(d, model.CreateRunnerImageRequest{
		Name:  "default",
		Image: "kronize/python-runner",
	})
	if err != nil {
		t.Fatal(err)
	}
	return img.ID
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

func TestUpdateJobVisibilityOwnerOnly(t *testing.T) {
	d := setupHandlerTestDB(t)
	secret := "test-secret"
	owner := createHandlerTestUser(t, d, "owner", "password")
	other := createHandlerTestUser(t, d, "other", "password")
	imgID := seedHandlerRunnerImage(t, d)
	if _, err := db.CreateJob(d, model.CreateJobRequest{
		Name: "j", CronExpression: "0 * * * *", PythonCode: "print(1)", ImageID: imgID,
	}, owner.ID); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(auth.Middleware(secret))
	r.Put("/api/jobs/{id}/visibility", UpdateJobVisibility(d))

	body := `[{"type":"user","user_id":` + itoa(other.ID) + `}]`
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, owner.ID, "owner", "user", "PUT", "/api/jobs/1/visibility", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("owner update should be 200, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, other.ID, "other", "user", "PUT", "/api/jobs/1/visibility", body))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("shared user must get 403, got %d", rr.Code)
	}

	// shared user CAN now read the job (full control)
	r2 := chi.NewRouter()
	r2.Use(auth.Middleware(secret))
	r2.Get("/api/jobs/{id}", GetJob(d))
	rr = httptest.NewRecorder()
	r2.ServeHTTP(rr, authenticatedRequest(secret, other.ID, "other", "user", "GET", "/api/jobs/1", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("shared user should read job, got %d", rr.Code)
	}
}

func TestGetExecutionVisibilityGate(t *testing.T) {
	d := setupHandlerTestDB(t)
	secret := "test-secret"
	owner := createHandlerTestUser(t, d, "owner", "password")
	alice := createHandlerTestUser(t, d, "alice", "password")
	imgID := seedHandlerRunnerImage(t, d)
	job, err := db.CreateJob(d, model.CreateJobRequest{
		Name: "j", CronExpression: "0 * * * *", PythonCode: "print(1)", ImageID: imgID,
	}, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	exec, err := db.CreateExecution(d, job.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(auth.Middleware(secret))
	r.Get("/api/executions/{id}", GetExecution(d))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, alice.ID, "alice", "user", "GET", "/api/executions/"+itoa(exec.ID), ""))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unrelated user must get 404, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, owner.ID, "owner", "user", "GET", "/api/executions/"+itoa(exec.ID), ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("owner should read execution, got %d", rr.Code)
	}
}

func TestGetStatsPerUser(t *testing.T) {
	d := setupHandlerTestDB(t)
	secret := "test-secret"
	owner := createHandlerTestUser(t, d, "owner", "password")
	alice := createHandlerTestUser(t, d, "alice", "password")
	imgID := seedHandlerRunnerImage(t, d)
	job, err := db.CreateJob(d, model.CreateJobRequest{
		Name: "j", CronExpression: "0 * * * *", PythonCode: "print(1)", ImageID: imgID,
	}, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateExecution(d, job.ID, "scheduled"); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(auth.Middleware(secret))
	r.Get("/api/stats", GetStats(d))

	// alice has no visible jobs
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, alice.ID, "alice", "user", "GET", "/api/stats", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("stats should be 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"total_jobs":0`) {
		t.Fatalf("alice should see total_jobs 0, got %s", rr.Body.String())
	}

	// owner sees their 1 job
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, authenticatedRequest(secret, owner.ID, "owner", "user", "GET", "/api/stats", ""))
	if !strings.Contains(rr.Body.String(), `"total_jobs":1`) {
		t.Fatalf("owner should see total_jobs 1, got %s", rr.Body.String())
	}
}
