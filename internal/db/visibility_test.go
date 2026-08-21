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
		t.Fatalf("alice should see only j2 (not unshared j1), got %d jobs", len(jobs))
	}
	_ = j1

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
