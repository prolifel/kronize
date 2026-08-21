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
