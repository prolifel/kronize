package logstream

import (
	"database/sql"
	"os"
	"testing"

	"kronize/internal/db"
	"kronize/internal/model"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	f, err := os.CreateTemp("", "kronize-logstream-test-*.db")
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

func TestHubSnapshotAndAppend(t *testing.T) {
	d := setupDB(t)

	user, err := db.CreateUser(d, model.CreateUserRequest{Username: "hubtest"}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	img, err := db.CreateRunnerImage(d, model.CreateRunnerImageRequest{
		Name: "default", Image: "kronize/python-runner",
	})
	if err != nil {
		t.Fatal(err)
	}
	job, err := db.CreateJob(d, model.CreateJobRequest{
		Name: "hub-test", CronExpression: "0 * * * *", PythonCode: "print('x')", ImageID: img.ID,
	}, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	exe, err := db.CreateExecution(d, job.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	hub := New(d)
	ch, snap, cancel, err := hub.SubscribeWithSnapshot(exe.ID, func() (Snapshot, error) {
		e, err := db.GetExecution(d, exe.ID)
		if err != nil {
			return Snapshot{}, err
		}
		return Snapshot{
			Status:     e.Status,
			Stdout:     e.Stdout,
			Stderr:     e.Stderr,
			ExitCode:   e.ExitCode,
			DurationMs: e.DurationMs,
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	if snap.Stdout != "" || snap.Stderr != "" {
		t.Fatalf("initial snapshot should be empty, got stdout=%q stderr=%q", snap.Stdout, snap.Stderr)
	}

	if err := hub.Append(exe.ID, "stdout", "live\n"); err != nil {
		t.Fatal(err)
	}

	ev := <-ch
	if ev.Type != "stdout" || ev.Data != "live\n" {
		t.Fatalf("event = %#v", ev)
	}
}
