package main

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"

	"kronize/internal/auth"
	"kronize/internal/db"
)

func TestSeedInitialAdminGeneratesRandomPasswordOnDemand(t *testing.T) {
	f, err := os.CreateTemp("", "kronize-main-test-*.db")
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

	var logBuf bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(oldLogger) })

	if err := seedInitialAdmin(d, ""); err != nil {
		t.Fatalf("seedInitialAdmin() error = %v", err)
	}

	admin, err := db.GetUserByUsername(d, "admin")
	if err != nil {
		t.Fatalf("GetUserByUsername(admin) error = %v", err)
	}
	if !admin.MustChangePassword {
		t.Fatalf("admin.MustChangePassword = false, want true")
	}
	for _, weak := range []string{"admin", "password", ""} {
		if auth.CheckPassword(weak, admin.PasswordHash) {
			t.Fatalf("admin password must not match weak default %q", weak)
		}
	}
	if !strings.Contains(logBuf.String(), "generated initial admin password") || !strings.Contains(logBuf.String(), "password=") {
		t.Fatalf("expected generated password to be logged for grep, got: %s", logBuf.String())
	}

	// Subsequent restarts when users already exist must not generate or log another password.
	logBuf.Reset()
	if err := seedInitialAdmin(d, ""); err != nil {
		t.Fatalf("second seedInitialAdmin() error = %v", err)
	}
	if logBuf.Len() != 0 {
		t.Fatalf("expected no log output on subsequent startup, got: %s", logBuf.String())
	}
}
