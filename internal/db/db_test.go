package db

import (
	"database/sql"
	"os"
	"testing"

	"kronize/internal/model"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	f, err := os.CreateTemp("", "kronize-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.Close()

	d, err := Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	if err := Migrate(d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestOpenAndMigrate(t *testing.T) {
	d := setupDB(t)

	// Verify tables exist
	rows, err := d.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}

	expected := []string{"executions", "jobs", "settings", "users"}
	for _, e := range expected {
		found := false
		for _, tbl := range tables {
			if tbl == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing table: %s", e)
		}
	}
}

func TestCreateAndGetUser(t *testing.T) {
	d := setupDB(t)

	u, err := CreateUser(d, model.CreateUserRequest{Username: "testuser"}, "hash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if u.Username != "testuser" {
		t.Errorf("username = %q, want %q", u.Username, "testuser")
	}
	if u.ID == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := GetUserByUsername(d, "testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("id = %d, want %d", got.ID, u.ID)
	}

	_, err = GetUserByUsername(d, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestCreateAndUpdateJob(t *testing.T) {
	d := setupDB(t)

	user, err := CreateUser(d, model.CreateUserRequest{Username: "jobtest"}, "hash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	req := model.CreateJobRequest{
		Name:           "test-job",
		Description:    "A test job",
		CronExpression: "0 * * * *",
		PythonCode:     "print('hello')",
		EnvVars:        `{"KEY": "val"}`,
		LogLevel:       "info",
	}
	j, err := CreateJob(d, req, user.ID)
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if j.Name != "test-job" {
		t.Errorf("name = %q, want %q", j.Name, "test-job")
	}
	if !j.Enabled {
		t.Error("expected enabled by default")
	}

	jobs, err := ListJobs(d, false)
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	enabled := false
	upd := model.UpdateJobRequest{Enabled: &enabled}
	updated, err := UpdateJob(d, j.ID, upd)
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if updated.Enabled {
		t.Error("expected job to be disabled after update")
	}

	enabledJobs, err := ListEnabledJobs(d)
	if err != nil {
		t.Fatalf("ListEnabledJobs() error = %v", err)
	}
	if len(enabledJobs) != 0 {
		t.Errorf("expected 0 enabled jobs, got %d", len(enabledJobs))
	}

	if err := DeleteJob(d, j.ID); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
	if _, err := GetJobByID(d, j.ID); err == nil {
		t.Error("expected error after delete")
	}
}
