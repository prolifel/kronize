# Job Visibility & Sharing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace "admin sees all jobs" with explicit visibility — job owners share jobs to the `admin` role and/or specific users, and everyone (including admins) sees only what they own or are shared.

**Architecture:** New `job_visibility` join table stores one row per target (`user` or `role`). A single access rule (owner OR user-share OR role-share) gates every job-facing endpoint, execution endpoints, and stats. Scheduler keeps an unfiltered query so cron still runs all enabled jobs.

**Tech Stack:** Go 1.x + chi + modernc.org/sqlite (backend); React 19 + TypeScript + Tailwind (frontend). Tests via stdlib `testing`.

**Spec:** `docs/superpowers/specs/2026-08-21-job-visibility-design.md`

## Global Constraints

- Go code follows existing style in `internal/` — `gofmt`, snake_case filenames, explicit error handling. Do NOT run formatters manually (pre-commit hooks handle it).
- Frontend: PascalCase components, Tailwind only, typed props via interfaces in `frontend/src/types.ts`.
- Read failures for inaccessible jobs → 404; write failures → 403 (existing semantics).
- Existing jobs get NO visibility rows → owner-only. No backfill.
- Scheduler must still run every enabled job — `ListEnabledJobs` stays unfiltered.
- All commits follow observed prefixes: `feat:`, `fix:`, `docs:`.
- Run verification with `make test`, `go vet ./...`, `npx tsc --noEmit` (from `frontend/`), `make build`.

---
---

### Task 1: Model types for visibility

**Files:**
- Modify: `internal/model/job.go`
- Modify: `internal/model/user.go`

**Interfaces:**
- Consumes: nothing (foundation task).
- Produces: `model.VisibilityTarget{Type string, UserID int64, Username string, Role string}`, `model.Job.Visibility []VisibilityTarget`, `model.CreateJobRequest.Visibility []VisibilityTarget`, `model.UserSearchResult{ID int64, Username string}`.

- [ ] **Step 1: Add `VisibilityTarget` and wire into `Job` + `CreateJobRequest`**

In `internal/model/job.go`, add:

```go
type VisibilityTarget struct {
	Type     string `json:"type"` // "user" | "role"
	UserID   int64  `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role,omitempty"` // "admin"
}
```

Add field to `Job`:

```go
	CreatedByUsername string            `json:"created_by_username"`
	Visibility       []VisibilityTarget `json:"visibility,omitempty"`
```

Add field to `CreateJobRequest`:

```go
	ImageID        int64               `json:"image_id"`
	Visibility     []VisibilityTarget  `json:"visibility,omitempty"`
```

- [ ] **Step 2: Add `UserSearchResult`**

In `internal/model/user.go`, add:

```go
type UserSearchResult struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS (no new code references these yet, so only syntax matters).

- [ ] **Step 4: Commit**

```bash
git add internal/model/job.go internal/model/user.go
git commit -m "feat: add job visibility model types"
```

---
---

### Task 2: DB — schema + visibility helpers

**Files:**
- Modify: `internal/db/db.go` (add table to `Migrate` schema)
- Create: `internal/db/visibility.go`
- Test: `internal/db/visibility_test.go`

**Interfaces:**
- Consumes: `model.VisibilityTarget` (Task 1).
- Produces: `db.ReplaceVisibility(db, jobID, targets) error`, `db.GetVisibility(db, jobID) ([]model.VisibilityTarget, error)`, `db.UserCanAccessJob(db, jobID, userID, role) (bool, error)`, `db.VisibleJobSubquery(userID, role) (string, []interface{})`.

- [ ] **Step 1: Write the failing tests**

Create `internal/db/visibility_test.go` (same package as `setupDB`/`seedRunnerImage` in `db_test.go`):

```go
package db

import (
	"database/sql"
	"testing"

	"kronize/internal/auth"
	"kronize/internal/model"
)

func seedUser(t *testing.T, d *sql.DB, username, role string) *model.User {
	t.Helper()
	hash, err := auth.HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	u, err := CreateUser(d, model.CreateUserRequest{Username: username, Role: role}, hash)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func createTestJob(t *testing.T, d *sql.DB, ownerID, imageID int64) *model.Job {
	t.Helper()
	job, err := CreateJob(d, model.CreateJobRequest{
		Name:           "test-job",
		CronExpression: "0 * * * *",
		PythonCode:     "print('hi')",
		ImageID:        imageID,
	}, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	return job
}
```

(imports: `database/sql`, `testing`, `kronize/internal/auth`, `kronize/internal/model`)

Test 1 — replace is atomic and clears:

```go
func TestReplaceAndGetVisibility(t *testing.T) {
	d := setupDB(t)
	owner := seedUser(t, d, "owner", "user")
	alice := seedUser(t, d, "alice", "user")
	img := seedRunnerImage(t, d)
	job := createTestJob(t, d, owner.ID, img)

	targets := []model.VisibilityTarget{
		{Type: "user", UserID: alice.ID},
		{Type: "role", Role: "admin"},
	}
	if err := ReplaceVisibility(d, job.ID, targets); err != nil {
		t.Fatal(err)
	}
	vis, err := GetVisibility(d, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(vis) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(vis))
	}
	// replacing with empty list clears everything
	if err := ReplaceVisibility(d, job.ID, nil); err != nil {
		t.Fatal(err)
	}
	vis, _ = GetVisibility(d, job.ID)
	if len(vis) != 0 {
		t.Fatalf("expected empty visibility after replace, got %d", len(vis))
	}
}
```

Test 2 — the access rule (owner / user share / role share / no access), including the key admin change:

```go
func TestUserCanAccessJob(t *testing.T) {
	d := setupDB(t)
	owner := seedUser(t, d, "owner", "user")
	alice := seedUser(t, d, "alice", "user")
	bob := seedUser(t, d, "bob", "user")
	admin := seedUser(t, d, "boss", "admin")
	img := seedRunnerImage(t, d)
	job := createTestJob(t, d, owner.ID, img)

	ok, err := UserCanAccessJob(d, job.ID, owner.ID, "user")
	if err != nil || !ok {
		t.Fatalf("owner should always access, ok=%v err=%v", ok, err)
	}
	ok, _ = UserCanAccessJob(d, job.ID, bob.ID, "user")
	if ok {
		t.Fatal("unrelated user must not access")
	}
	ok, _ = UserCanAccessJob(d, job.ID, admin.ID, "admin")
	if ok {
		t.Fatal("admin must NOT see unshared job — this is the core behavior change")
	}

	if err := ReplaceVisibility(d, job.ID, []model.VisibilityTarget{{Type: "user", UserID: alice.ID}}); err != nil {
		t.Fatal(err)
	}
	ok, _ = UserCanAccessJob(d, job.ID, alice.ID, "user")
	if !ok {
		t.Fatal("user-shared target must grant access")
	}

	if err := ReplaceVisibility(d, job.ID, []model.VisibilityTarget{{Type: "role", Role: "admin"}}); err != nil {
		t.Fatal(err)
	}
	ok, _ = UserCanAccessJob(d, job.ID, admin.ID, "admin")
	if !ok {
		t.Fatal("role-shared target must grant access to admins")
	}
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `go test ./internal/db/ -run 'TestReplaceAndGetVisibility|TestUserCanAccessJob' -v`
Expected: FAIL — `ReplaceVisibility`/`GetVisibility`/`UserCanAccessJob` undefined; `job_visibility` table missing.

- [ ] **Step 3: Add the table to the migration**

In `internal/db/db.go`, inside the `schema` string in `Migrate`, append after the `runner_images` table:

```sql
	CREATE TABLE IF NOT EXISTS job_visibility (
		job_id          INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
		target_type     TEXT NOT NULL CHECK (target_type IN ('user','role')),
		target_user_id  INTEGER REFERENCES users(id),
		target_role     TEXT,
		PRIMARY KEY (job_id, target_type,
		             COALESCE(target_user_id, -1),
		             COALESCE(target_role, ''))
	);
```

- [ ] **Step 4: Implement `internal/db/visibility.go`**

```go
package db

import (
	"database/sql"
	"fmt"

	"kronize/internal/model"
)

func ReplaceVisibility(db *sql.DB, jobID int64, targets []model.VisibilityTarget) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin visibility tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM job_visibility WHERE job_id = ?", jobID); err != nil {
		return fmt.Errorf("clear visibility: %w", err)
	}
	for _, t := range targets {
		switch t.Type {
		case "user":
			if _, err := tx.Exec(
				"INSERT INTO job_visibility (job_id, target_type, target_user_id) VALUES (?, 'user', ?)",
				jobID, t.UserID,
			); err != nil {
				return fmt.Errorf("insert user visibility: %w", err)
			}
		case "role":
			if _, err := tx.Exec(
				"INSERT INTO job_visibility (job_id, target_type, target_role) VALUES (?, 'role', ?)",
				jobID, t.Role,
			); err != nil {
				return fmt.Errorf("insert role visibility: %w", err)
			}
		}
	}
	return tx.Commit()
}

func GetVisibility(db *sql.DB, jobID int64) ([]model.VisibilityTarget, error) {
	rows, err := db.Query(
		`SELECT v.target_type, COALESCE(v.target_user_id, 0), COALESCE(v.target_role, ''),
		        COALESCE(u.username, '')
		 FROM job_visibility v
		 LEFT JOIN users u ON u.id = v.target_user_id
		 WHERE v.job_id = ?
		 ORDER BY v.target_type, v.target_user_id`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("get visibility: %w", err)
	}
	defer rows.Close()

	var targets []model.VisibilityTarget
	for rows.Next() {
		var t model.VisibilityTarget
		if err := rows.Scan(&t.Type, &t.UserID, &t.Role, &t.Username); err != nil {
			return nil, fmt.Errorf("scan visibility: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func UserCanAccessJob(db *sql.DB, jobID, userID int64, role string) (bool, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM jobs j
		 WHERE j.id = ?
		   AND (j.created_by = ?
		        OR EXISTS (SELECT 1 FROM job_visibility v
		                   WHERE v.job_id = j.id
		                     AND ((v.target_type = 'user' AND v.target_user_id = ?)
		                          OR (v.target_type = 'role' AND v.target_role = ?))))`,
		jobID, userID, userID, role,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("access check: %w", err)
	}
	return count > 0, nil
}

func VisibleJobSubquery(userID int64, role string) (string, []interface{}) {
	return `(SELECT j.id FROM jobs j
	         WHERE j.created_by = ?
	            OR EXISTS (SELECT 1 FROM job_visibility v
	                       WHERE v.job_id = j.id
	                         AND ((v.target_type = 'user' AND v.target_user_id = ?)
	                              OR (v.target_type = 'role' AND v.target_role = ?))))`,
		[]interface{}{userID, userID, role}
}
```

- [ ] **Step 5: Run tests, verify they pass**

Run: `go test ./internal/db/ -run 'TestReplaceAndGetVisibility|TestUserCanAccessJob' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/db/db.go internal/db/visibility.go internal/db/visibility_test.go
git commit -m "feat: add job_visibility table and access helpers"
```

---
---

### Task 3: DB — visibility-aware job queries

**Files:**
- Modify: `internal/db/jobs.go`
- Test: `internal/db/visibility_test.go` (append)

**Interfaces:**
- Consumes: `db.VisibleJobSubquery` (Task 2).
- Produces: `db.ListJobs` now filters by visibility for ALL roles; `db.ListEnabledJobs` returns all enabled jobs unfiltered; `db.CreateJob` persists optional initial visibility.

- [ ] **Step 1: Write the failing test**

Append to `internal/db/visibility_test.go`:

```go
func TestListJobsVisibilityFilter(t *testing.T) {
	d := setupDB(t)
	owner := seedUser(t, d, "owner", "user")
	alice := seedUser(t, d, "alice", "user")
	admin := seedUser(t, d, "boss", "admin")
	img := seedRunnerImage(t, d)
	j1 := createTestJob(t, d, owner.ID, img)
	j2 := createTestJob(t, d, owner.ID, img)
	j3 := createTestJob(t, d, owner.ID, img)

	if err := ReplaceVisibility(d, j2.ID, []model.VisibilityTarget{{Type: "user", UserID: alice.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceVisibility(d, j3.ID, []model.VisibilityTarget{{Type: "role", Role: "admin"}}); err != nil {
		t.Fatal(err)
	}

	jobs, err := ListJobs(d, false, alice.ID, "user")
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != j2.ID {
		t.Fatalf("alice should see only j2, got %d jobs", len(jobs))
	}

	jobs, err = ListJobs(d, false, admin.ID, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != j3.ID {
		t.Fatalf("admin should see only j3, got %d jobs", len(jobs))
	}

	all, err := ListEnabledJobs(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("scheduler path must see all 3 enabled jobs, got %d", len(all))
	}
}

func TestCreateJobWithVisibility(t *testing.T) {
	d := setupDB(t)
	owner := seedUser(t, d, "owner", "user")
	alice := seedUser(t, d, "alice", "user")
	img := seedRunnerImage(t, d)

	job, err := CreateJob(d, model.CreateJobRequest{
		Name:           "shared-job",
		CronExpression: "0 * * * *",
		PythonCode:     "print(1)",
		ImageID:        img,
		Visibility:     []model.VisibilityTarget{{Type: "user", UserID: alice.ID}},
	}, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	vis, err := GetVisibility(d, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(vis) != 1 || vis[0].Type != "user" || vis[0].UserID != alice.ID {
		t.Fatalf("expected alice share, got %+v", vis)
	}
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `go test ./internal/db/ -run 'TestListJobsVisibilityFilter|TestCreateJobWithVisibility' -v`
Expected: FAIL — admin still sees all 3 jobs; create ignores visibility.

- [ ] **Step 3: Filter `ListJobs` for every role**

In `internal/db/jobs.go`, replace the body of `ListJobs` (keep signature) with:

```go
func ListJobs(db *sql.DB, enabledOnly bool, userID int64, role string) ([]*model.Job, error) {
	sub, args := VisibleJobSubquery(userID, role)
	clauses := []string{"j.id IN " + sub}
	if enabledOnly {
		clauses = append(clauses, "j.enabled = 1")
	}
	query := "SELECT " + jobCols + " " + jobFrom + " WHERE " + joinClauses(clauses) + " ORDER BY j.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*model.Job
	for rows.Next() {
		j := &model.Job{}
		if err := rows.Scan(&j.ID, &j.Name, &j.Description, &j.CronExpression, &j.PythonCode,
			&j.ImageID, &j.Image, &j.EnvVars, &j.HostMappings, &j.LogLevel, &j.Enabled, &j.CreatedBy, &j.CreatedByUsername, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
```

Add helper `joinClauses` next to `joinFields`:

```go
func joinClauses(clauses []string) string {
	result := ""
	for i, c := range clauses {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}
```

- [ ] **Step 4: Make `ListEnabledJobs` unfiltered**

Replace `ListEnabledJobs` (it currently delegates to `ListJobs` with the admin bypass, which no longer bypasses):

```go
func ListEnabledJobs(db *sql.DB) ([]*model.Job, error) {
	rows, err := db.Query(
		"SELECT " + jobCols + " " + jobFrom + " WHERE j.enabled = 1 ORDER BY j.created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*model.Job
	for rows.Next() {
		j := &model.Job{}
		if err := rows.Scan(&j.ID, &j.Name, &j.Description, &j.CronExpression, &j.PythonCode,
			&j.ImageID, &j.Image, &j.EnvVars, &j.HostMappings, &j.LogLevel, &j.Enabled, &j.CreatedBy, &j.CreatedByUsername, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
```

- [ ] **Step 5: Persist initial visibility in `CreateJob`**

In `CreateJob`, after the `id, _ := res.LastInsertId()` line and before `return GetJobByID(db, id)`:

```go
	if len(req.Visibility) > 0 {
		if err := ReplaceVisibility(db, id, req.Visibility); err != nil {
			return nil, fmt.Errorf("create job visibility: %w", err)
		}
	}
	return GetJobByID(db, id)
```

(remove the old `return GetJobByID(db, id)` line)

- [ ] **Step 6: Run tests, verify they pass**

Run: `go test ./internal/db/ -v`
Expected: all PASS (existing + new).

- [ ] **Step 7: Commit**

```bash
git add internal/db/jobs.go internal/db/visibility_test.go
git commit -m "feat: filter job list by visibility, keep scheduler unfiltered"
```

---
---

### Task 4: DB — user search

**Files:**
- Modify: `internal/db/users.go`
- Test: `internal/db/visibility_test.go` (append)

**Interfaces:**
- Consumes: `model.UserSearchResult` (Task 1).
- Produces: `db.SearchUsers(db, q string) ([]model.UserSearchResult, error)` — empty `q` returns all users (id + username only), else username LIKE filter, max 20 rows.

- [ ] **Step 1: Write the failing test**

Append to `internal/db/visibility_test.go`:

```go
func TestSearchUsers(t *testing.T) {
	d := setupDB(t)
	seedUser(t, d, "alice", "user")
	seedUser(t, d, "bob", "user")
	seedUser(t, d, "carol", "user")

	all, err := SearchUsers(d, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("empty query should return all users, got %d", len(all))
	}

	matches, err := SearchUsers(d, "al")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Username != "alice" {
		t.Fatalf("expected only alice, got %+v", matches)
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `go test ./internal/db/ -run TestSearchUsers -v`
Expected: FAIL — `SearchUsers` undefined.

- [ ] **Step 3: Implement `SearchUsers`**

Append to `internal/db/users.go`:

```go
func SearchUsers(db *sql.DB, q string) ([]model.UserSearchResult, error) {
	rows, err := db.Query(
		"SELECT id, username FROM users WHERE username LIKE '%' || ? || '%' ORDER BY username LIMIT 20",
		q,
	)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var users []model.UserSearchResult
	for rows.Next() {
		var u model.UserSearchResult
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}
```

- [ ] **Step 4: Run test, verify it passes**

Run: `go test ./internal/db/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/db/users.go internal/db/visibility_test.go
git commit -m "feat: add user search for share picker"
```

---
---

### Task 5: Handlers — access control + visibility endpoint + search

**Files:**
- Modify: `internal/handler/job_handler.go` (`canAccessJob`, `GetJob`, new `UpdateJobVisibility`)
- Modify: `internal/handler/execution_handler.go` (`ListExecutions`, `GetExecution`)
- Modify: `internal/handler/execution_stream_handler.go` (`StreamExecution`)
- Modify: `internal/handler/stats_handler.go` (`GetStats`)
- Create: `internal/handler/user_search_handler.go`
- Test: `internal/handler/job_visibility_test.go`

**Interfaces:**
- Consumes: `db.UserCanAccessJob`, `db.GetVisibility`, `db.ReplaceVisibility`, `db.VisibleJobSubquery`, `db.SearchUsers`, `db.GetUserByID`, `model.VisibilityTarget` (Tasks 1-4).
- Produces: `handler.UpdateJobVisibility(db) http.HandlerFunc` (owner-only, bare JSON array body, returns updated visibility), `handler.SearchUsers(db) http.HandlerFunc`, and visibility-gated versions of existing handlers.

- [ ] **Step 1: Write the failing handler tests**

Create `internal/handler/job_visibility_test.go` (same package `handler`, reusing `setupHandlerTestDB`, `createHandlerTestUser`, `authenticatedRequest` from `auth_handler_test.go`):

```go
package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

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

func TestUpdateJobVisibilityOwnerOnly(t *testing.T) {
	d := setupHandlerTestDB(t)
	secret := "test-secret"
	owner := createHandlerTestUser(t, d, "owner", "password")
	other := createHandlerTestUser(t, d, "other", "password")
	imgID := seedHandlerRunnerImage(t, d)
	job, err := db.CreateJob(d, model.CreateJobRequest{
		Name: "j", CronExpression: "0 * * * *", PythonCode: "print(1)", ImageID: imgID,
	}, owner.ID)
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
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
	r2.Get("/api/jobs/{id}", GetJob(d))
	rr = httptest.NewRecorder()
	r2.ServeHTTP(rr, authenticatedRequest(secret, other.ID, "other", "user", "GET", "/api/jobs/1", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("shared user should read job, got %d", rr.Code)
	}
}
```

Helper `itoa` (add to the test file):

```go
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
```

(import `strconv`)

Second test — executions gated by visibility:

```go
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
```

Third test — stats are per-user:

```go
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
```

(import `strings`)

- [ ] **Step 2: Run tests, verify they fail**

Run: `go test ./internal/handler/ -run 'TestUpdateJobVisibilityOwnerOnly|TestGetExecutionVisibilityGate|TestGetStatsPerUser' -v`
Expected: FAIL — `UpdateJobVisibility` undefined; execution/stats not gated (404/`total_jobs:1` failures).

- [ ] **Step 3: Rework `canAccessJob` in `internal/handler/job_handler.go`**

Replace the whole `canAccessJob` function:

```go
func canAccessJob(database *sql.DB, userID int64, role string, jobID int64, forWrite bool) (*model.Job, *accessError) {
	job, err := db.GetJobByID(database, jobID)
	if err != nil {
		return nil, &accessError{Code: http.StatusNotFound, Err: fmt.Errorf("job not found")}
	}
	ok, err := db.UserCanAccessJob(database, jobID, userID, role)
	if err != nil {
		return nil, &accessError{Code: http.StatusInternalServerError, Err: fmt.Errorf("access check failed")}
	}
	if !ok {
		if forWrite {
			return nil, &accessError{Code: http.StatusForbidden, Err: fmt.Errorf("forbidden")}
		}
		return nil, &accessError{Code: http.StatusNotFound, Err: fmt.Errorf("job not found")}
	}
	return job, nil
}
```

- [ ] **Step 4: Attach visibility to `GetJob` response**

In `GetJob`, after the `canAccessJob` check and before `jsonResponse`:

```go
		vis, err := db.GetVisibility(database, id)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to load visibility")
			return
		}
		job.Visibility = vis
		jsonResponse(w, http.StatusOK, job)
```

(remove the old `jsonResponse(w, http.StatusOK, job)` line)

- [ ] **Step 5: Add `UpdateJobVisibility` handler**

Append to `internal/handler/job_handler.go`:

```go
func UpdateJobVisibility(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid job id")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		job, err := db.GetJobByID(database, id)
		if err != nil {
			jsonError(w, http.StatusNotFound, "job not found")
			return
		}
		if job.CreatedBy != userID {
			jsonError(w, http.StatusForbidden, "only the job owner can manage visibility")
			return
		}
		var targets []model.VisibilityTarget
		if err := decodeJSON(r, &targets); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		for _, t := range targets {
			switch t.Type {
			case "user":
				if t.UserID <= 0 {
					jsonError(w, http.StatusBadRequest, "invalid user target")
					return
				}
				if _, err := db.GetUserByID(database, t.UserID); err != nil {
					jsonError(w, http.StatusBadRequest, "unknown user")
					return
				}
			case "role":
				if t.Role != "admin" {
					jsonError(w, http.StatusBadRequest, "unsupported role target")
					return
				}
			default:
				jsonError(w, http.StatusBadRequest, "invalid target type")
				return
			}
		}
		if err := db.ReplaceVisibility(database, id, targets); err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to update visibility")
			return
		}
		vis, err := db.GetVisibility(database, id)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to load visibility")
			return
		}
		jsonResponse(w, http.StatusOK, vis)
	}
}
```

- [ ] **Step 6: Gate executions in `internal/handler/execution_handler.go`**

In `ListExecutions`, after parsing `jobID` and before the `db.ListExecutionsByJob` call:

```go
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		if _, aerr := canAccessJob(database, userID, role, jobID, false); aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
```

In `GetExecution`, after `db.GetExecution` succeeds and before `jsonResponse`:

```go
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		if _, aerr := canAccessJob(database, userID, role, exec.JobID, false); aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
```

- [ ] **Step 7: Gate stream in `internal/handler/execution_stream_handler.go`**

In `StreamExecution`, after `exec, err := db.GetExecution(...)` succeeds and before the `exec.Source != "manual"` check:

```go
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		if _, aerr := canAccessJob(database, userID, role, exec.JobID, false); aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
```

- [ ] **Step 8: Make stats per-user in `internal/handler/stats_handler.go`**

Replace the whole `GetStats` handler body:

```go
func GetStats(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		sub, args := db.VisibleJobSubquery(userID, role)

		totalJobs := 0
		database.QueryRow("SELECT COUNT(*) FROM jobs WHERE id IN "+sub, args...).Scan(&totalJobs)
		enabledJobs := 0
		database.QueryRow("SELECT COUNT(*) FROM jobs WHERE enabled = 1 AND id IN "+sub, args...).Scan(&enabledJobs)
		total24h := 0
		database.QueryRow("SELECT COUNT(*) FROM executions WHERE started_at > datetime('now', '-1 day') AND job_id IN "+sub, args...).Scan(&total24h)
		failuresToday := 0
		database.QueryRow("SELECT COUNT(*) FROM executions WHERE status = 'failed' AND started_at > datetime('now', '-1 day') AND job_id IN "+sub, args...).Scan(&failuresToday)
		successRate := 0.0
		if total24h > 0 {
			successRate = float64(total24h-failuresToday) / float64(total24h) * 100
		}
		jsonResponse(w, http.StatusOK, StatsResponse{
			TotalJobs:      totalJobs,
			EnabledJobs:    enabledJobs,
			SuccessRate24h: successRate,
			FailuresToday:  failuresToday,
		})
	}
}
```

Add import `"kronize/internal/auth"` to `stats_handler.go`.

- [ ] **Step 9: Add `SearchUsers` handler**

Create `internal/handler/user_search_handler.go`:

```go
package handler

import (
	"database/sql"
	"net/http"

	"kronize/internal/db"
	"kronize/internal/model"
)

func SearchUsers(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := db.SearchUsers(database, r.URL.Query().Get("q"))
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to search users")
			return
		}
		if users == nil {
			users = []model.UserSearchResult{}
		}
		jsonResponse(w, http.StatusOK, users)
	}
}
```

- [ ] **Step 10: Run handler tests**

Run: `go test ./internal/handler/ -v`
Expected: all PASS (existing auth tests + new).

- [ ] **Step 11: Commit**

```bash
git add internal/handler/
git commit -m "feat: gate jobs, executions, and stats by visibility"
```

---
---

### Task 6: Server — route registration

**Files:**
- Modify: `internal/server/server.go`

**Interfaces:**
- Consumes: `handler.UpdateJobVisibility`, `handler.SearchUsers` (Task 5).
- Produces: `PUT /api/jobs/{id}/visibility` (auth, any role; owner check inside), `GET /api/users/search` (auth, any role).

- [ ] **Step 1: Register the routes**

In `internal/server/server.go`, inside the auth-protected `/api` route group, add near the job routes:

```go
		r.Put("/jobs/{id}/visibility", handler.UpdateJobVisibility(s.DB))
```

and add near `r.Get("/settings", ...)`:

```go
		r.Get("/users/search", handler.SearchUsers(s.DB))
```

`/users/search` stays OUTSIDE the `auth.AdminOnly` group. chi matches the static segment `/users/search` before the `/users/{id}` param route in the admin group, so there is no conflict.

- [ ] **Step 2: Verify build + tests**

Run: `make test && go vet ./...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/server/server.go
git commit -m "feat: register visibility and user search routes"
```

---
---

### Task 7: Frontend — types + API client

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`

**Interfaces:**
- Consumes: backend JSON shapes from Tasks 1 and 5.
- Produces: `VisibilityTarget`, `UserSearchResult`, `Job.visibility`, `JobFormData.visibility`, `api.searchUsers(q)`, `api.updateJobVisibility(id, targets)`.

- [ ] **Step 1: Update types**

In `frontend/src/types.ts`:

```ts
export interface VisibilityTarget {
  type: 'user' | 'role'
  user_id?: number
  username?: string
  role?: string
}

export interface UserSearchResult {
  id: number
  username: string
}
```

Add to `Job`:

```ts
  visibility: VisibilityTarget[]
```

Add to `JobFormData`:

```ts
  visibility: VisibilityTarget[]
```

- [ ] **Step 2: Keep the type-check green**

In `frontend/src/pages/JobFormPage.tsx`, the initial `form` state must include the new required field — add `visibility: [],` next to `image_id: 0`:

```tsx
    image_id: 0,
    visibility: [],
```

- [ ] **Step 3: Update API client**

In `frontend/src/api.ts`, after `toggleJob`:

```ts
  updateJobVisibility: (id: number, targets: VisibilityTarget[]) =>
    request<VisibilityTarget[]>(`/jobs/${id}/visibility`, { method: 'PUT', body: JSON.stringify(targets) }),

  searchUsers: (q: string) =>
    request<UserSearchResult[]>(`/users/search${q ? `?q=${encodeURIComponent(q)}` : ''}`),
```

Update the import line:

```ts
import type { Job, JobFormData, Execution, Stats, Setting, LoginResponse, User, RunnerImage, VisibilityTarget, UserSearchResult } from './types'
```

- [ ] **Step 4: Type-check**

Run: `cd frontend && npx tsc --noEmit`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types.ts frontend/src/api.ts
git commit -m "feat: add visibility types and API client methods"
```

---
---

### Task 8: Frontend — sharing editor on job form

**Files:**
- Create: `frontend/src/components/JobVisibilityEditor.tsx`
- Modify: `frontend/src/pages/JobFormPage.tsx`

**Interfaces:**
- Consumes: `api.searchUsers`, `api.updateJobVisibility`, `VisibilityTarget`, `UserSearchResult` (Task 7).
- Produces: `<JobVisibilityEditor targets={VisibilityTarget[]} onChange={(t) => void} />`; job form saves visibility on create (in payload) and on edit (separate endpoint call).

- [ ] **Step 1: Create the editor component**

`frontend/src/components/JobVisibilityEditor.tsx`:

```tsx
import { useEffect, useRef, useState } from 'react'
import { api } from '../api'
import type { VisibilityTarget, UserSearchResult } from '../types'

interface Props {
  targets: VisibilityTarget[]
  onChange: (targets: VisibilityTarget[]) => void
}

export default function JobVisibilityEditor({ targets, onChange }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<UserSearchResult[]>([])
  const [open, setOpen] = useState(false)
  const searchSeq = useRef(0)

  useEffect(() => {
    const seq = ++searchSeq.current
    const t = setTimeout(() => {
      api.searchUsers(query).then((users) => {
        if (seq === searchSeq.current) setResults(users)
      }).catch(() => {})
    }, 200)
    return () => clearTimeout(t)
  }, [query])

  const hasAdmin = targets.some((t) => t.type === 'role')
  const userTargets = targets.filter((t) => t.type === 'user')

  const toggleAdmin = () => {
    if (hasAdmin) {
      onChange(targets.filter((t) => t.type !== 'role'))
    } else {
      onChange([...targets, { type: 'role', role: 'admin' }])
    }
  }

  const addUser = (u: UserSearchResult) => {
    if (userTargets.some((t) => t.user_id === u.id)) return
    onChange([...targets, { type: 'user', user_id: u.id, username: u.username }])
    setQuery('')
    setOpen(false)
  }

  const removeUser = (userId: number) => {
    onChange(targets.filter((t) => !(t.type === 'user' && t.user_id === userId)))
  }

  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input
          type="checkbox"
          checked={hasAdmin}
          onChange={toggleAdmin}
          className="h-4 w-4"
        />
        Share with admin role (all admins)
      </label>

      <div className="relative">
        <input
          type="text"
          value={query}
          onChange={(e) => { setQuery(e.target.value); setOpen(true) }}
          onFocus={() => setOpen(true)}
          placeholder="Search users to share with..."
          className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
        />
        {open && results.length > 0 && (
          <ul className="absolute z-10 mt-1 w-full bg-white border border-gray-300 rounded-md shadow-lg max-h-48 overflow-auto">
            {results.map((u) => (
              <li key={u.id}>
                <button
                  type="button"
                  onClick={() => addUser(u)}
                  className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
                >
                  {u.username}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {userTargets.length > 0 && (
        <ul className="flex flex-wrap gap-2">
          {userTargets.map((t) => (
            <li key={t.user_id} className="flex items-center gap-2 bg-gray-100 rounded-full px-3 py-1 text-sm">
              {t.username}
              <button
                type="button"
                onClick={() => removeUser(t.user_id!)}
                className="text-gray-500 hover:text-red-600"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Integrate into `JobFormPage`**

In `frontend/src/pages/JobFormPage.tsx`:

1. Imports — add:

```tsx
import JobVisibilityEditor from '../components/JobVisibilityEditor'
import type { JobFormData, RunnerImage, VisibilityTarget } from '../types'
```

2. Form state — add `visibility: []` to the initial `form`:

```tsx
    image_id: 0,
    visibility: [],
```

3. Edit load — in the `api.getJob` `.then`, add:

```tsx
          visibility: job.visibility || [],
```

4. Render — after `<HostMappingsEditor ... />` and before the submit buttons, add:

```tsx
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Sharing</label>
          <JobVisibilityEditor
            targets={form.visibility}
            onChange={(v) => setForm((prev) => ({ ...prev, visibility: v }))}
          />
        </div>
```

5. Submit — in `handleSubmit`:

```tsx
      if (isEdit) {
        await api.updateJob(Number(id), form)
        await api.updateJobVisibility(Number(id), form.visibility)
      } else {
        await api.createJob(form)
      }
```

- [ ] **Step 3: Type-check + build**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/JobVisibilityEditor.tsx frontend/src/pages/JobFormPage.tsx
git commit -m "feat: add sharing editor to job form"
```

---
---

### Task 9: Frontend — shared indicators + detail view

**Files:**
- Modify: `frontend/src/pages/JobListPage.tsx`
- Modify: `frontend/src/pages/JobDetailPage.tsx`

**Interfaces:**
- Consumes: `useAuth()` (already used in `JobDetailPage`), `Job.visibility`, `Job.created_by` (Task 7).
- Produces: "Shared" badge on list rows where `job.created_by !== currentUser.id`; read-only "Shared with" line on the detail page.

- [ ] **Step 1: List page badge**

In `frontend/src/pages/JobListPage.tsx`:

1. Import `useAuth`:

```tsx
import { useAuth } from '../AuthContext'
```

2. In the component, add:

```tsx
  const { user: currentUser } = useAuth()
```

3. In the job row name cell (currently `<div className="text-sm font-medium text-gray-900">{job.name}</div>`), add a badge when shared:

```tsx
                    <div className="flex items-center gap-2">
                      <div className="text-sm font-medium text-gray-900">{job.name}</div>
                      {currentUser && job.created_by !== currentUser.id && (
                        <span className="text-xs bg-purple-100 text-purple-700 rounded-full px-2 py-0.5">
                          Shared
                        </span>
                      )}
                    </div>
```

- [ ] **Step 2: Detail page shared-with line**

In `frontend/src/pages/JobDetailPage.tsx`, in the header area where job info renders, add a read-only summary:

```tsx
      {job && job.visibility && job.visibility.length > 0 && (
        <p className="text-sm text-gray-500">
          Shared with:{' '}
          {job.visibility.map((t) =>
            t.type === 'role' ? 'all admins' : t.username
          ).join(', ')}
        </p>
      )}
```

Place it directly under the job title heading.

- [ ] **Step 3: Type-check + build**

Run: `cd frontend && npx tsc --noEmit && npm run build`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/JobListPage.tsx frontend/src/pages/JobDetailPage.tsx
git commit -m "feat: show shared indicators in job list and detail"
```

---
---

### Task 10: Full verification

**Files:** none (verification only).

- [ ] **Step 1: Full test suite**

Run: `make test`
Expected: all PASS.

- [ ] **Step 2: Vet + frontend build**

Run: `make check`
Expected: PASS (test + vet + tsc + frontend build).

- [ ] **Step 3: Manual smoke (documented, not automated)**

With `make dev` running:

1. Log in as `admin` → job list shows only admin's own jobs (previously all).
2. As owner, open a job → Sharing section → toggle "Share with admin role" → save → log in as another admin → job visible.
3. As owner, share to user `alice` → `alice` sees the job, can run/edit it, but the Sharing section is not rendered for her (owner-only).
4. Verify stats cards reflect only visible jobs.
5. Verify `GET /api/users/search?q=al` returns id + username only.
6. Verify a job with no shares is invisible to admins and other users (404 on direct `GET /api/jobs/{id}`).

- [ ] **Step 4: Final commit (if anything drifted)**

```bash
git status
```

Commit any leftover fixes with an appropriate `fix:` or `feat:` message.
