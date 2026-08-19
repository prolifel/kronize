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

func seedRunnerImage(t *testing.T, d *sql.DB) int64 {
	t.Helper()
	img, err := CreateRunnerImage(d, model.CreateRunnerImageRequest{
		Name:        "default",
		Image:       "kronize/python-runner",
		Description: "Default runner",
	})
	if err != nil {
		t.Fatal(err)
	}
	return img.ID
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

	expected := []string{"executions", "jobs", "runner_images", "settings", "users"}
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

	imageID := seedRunnerImage(t, d)

	req := model.CreateJobRequest{
		Name:           "test-job",
		Description:    "A test job",
		CronExpression: "0 * * * *",
		PythonCode:     "print('hello')",
		ImageID:        imageID,
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
	if j.ImageID != imageID {
		t.Errorf("image_id = %d, want %d", j.ImageID, imageID)
	}
	if j.Image != "kronize/python-runner" {
		t.Errorf("image = %q, want %q", j.Image, "kronize/python-runner")
	}
	if !j.Enabled {
		t.Error("expected enabled by default")
	}

	jobs, err := ListJobs(d, false, 0, "admin")
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

func TestCreateAndCompleteExecution(t *testing.T) {
	d := setupDB(t)

	user, err := CreateUser(d, model.CreateUserRequest{Username: "exectest"}, "hash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	imageID := seedRunnerImage(t, d)

	req := model.CreateJobRequest{
		Name:           "exec-test",
		CronExpression: "0 * * * *",
		PythonCode:     "print('x')",
		ImageID:        imageID,
	}
	j, err := CreateJob(d, req, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	e, err := CreateExecution(d, j.ID)
	if err != nil {
		t.Fatalf("CreateExecution() error = %v", err)
	}
	if e.Status != "running" {
		t.Errorf("status = %q, want %q", e.Status, "running")
	}

	err = CompleteExecution(d, e.ID, "success", "output", "", 0, 1500)
	if err != nil {
		t.Fatalf("CompleteExecution() error = %v", err)
	}

	got, err := GetExecution(d, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "success" {
		t.Errorf("status = %q, want %q", got.Status, "success")
	}
	if got.Stdout != "output" {
		t.Errorf("stdout = %q, want %q", got.Stdout, "output")
	}

	execs, err := ListExecutionsByJob(d, j.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(execs) != 1 {
		t.Errorf("expected 1 execution, got %d", len(execs))
	}
}

func TestSettings(t *testing.T) {
	d := setupDB(t)

	if err := SetSetting(d, "teams_webhook_url", "https://example.com/webhook"); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	val, err := GetSetting(d, "teams_webhook_url")
	if err != nil {
		t.Fatal(err)
	}
	if val != "https://example.com/webhook" {
		t.Errorf("value = %q, want %q", val, "https://example.com/webhook")
	}

	val, err = GetSetting(d, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Errorf("expected empty, got %q", val)
	}

	all, err := GetAllSettings(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 setting, got %d", len(all))
	}
}

func TestJobCreatedByUsername(t *testing.T) {
	d := setupDB(t)

	user, err := CreateUser(d, model.CreateUserRequest{Username: "creator"}, "hash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	imageID := seedRunnerImage(t, d)

	j, err := CreateJob(d, model.CreateJobRequest{
		Name:           "creator-job",
		CronExpression: "0 * * * *",
		ImageID:        imageID,
	}, user.ID)
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if j.CreatedByUsername != "creator" {
		t.Errorf("CreateJob() CreatedByUsername = %q, want %q", j.CreatedByUsername, "creator")
	}

	jobs, err := ListJobs(d, false, 0, "admin")
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].CreatedByUsername != "creator" {
		t.Errorf("ListJobs() CreatedByUsername = %q, want %q", jobs[0].CreatedByUsername, "creator")
	}
}

func TestJobHostMappings(t *testing.T) {
	d := setupDB(t)

	user, err := CreateUser(d, model.CreateUserRequest{Username: "mapuser"}, "hash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	imageID := seedRunnerImage(t, d)

	mappings := `[{"host":"/Users/clement/.cspm_msal_token.json","container":"/root/.cspm_msal_token.json","read_only":true}]`
	j, err := CreateJob(d, model.CreateJobRequest{
		Name:           "map-job",
		CronExpression: "0 * * * *",
		ImageID:        imageID,
		HostMappings:   mappings,
	}, user.ID)
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if j.HostMappings != mappings {
		t.Errorf("CreateJob() HostMappings = %q, want %q", j.HostMappings, mappings)
	}

	jobs, err := ListJobs(d, false, 0, "admin")
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 1 || jobs[0].HostMappings != mappings {
		t.Errorf("ListJobs() HostMappings = %q, want %q", jobs[0].HostMappings, mappings)
	}

	empty := ""
	updated, err := UpdateJob(d, j.ID, model.UpdateJobRequest{HostMappings: &empty})
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if updated.HostMappings != "" {
		t.Errorf("UpdateJob() HostMappings = %q, want empty", updated.HostMappings)
	}
}
