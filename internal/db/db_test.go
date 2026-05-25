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
