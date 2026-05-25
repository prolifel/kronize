package db

import (
	"os"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	f, err := os.CreateTemp("", "kronize-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	d, err := Open(f.Name())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer d.Close()

	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

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
