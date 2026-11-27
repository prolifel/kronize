package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kronize/internal/auth"
	"kronize/internal/db"
	klog "kronize/internal/log"
	"kronize/internal/server"
)

func getAddr() string {
	if v := os.Getenv("ADDR"); v != "" {
		return v
	}
	return ":8080"
}

func generateRandomPassword() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func seedInitialAdmin(database *sql.DB, adminPass string) error {
	count, err := db.CountUsers(database)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	generated := false
	if adminPass == "" {
		adminPass, err = generateRandomPassword()
		if err != nil {
			return err
		}
		generated = true
	}
	adminHash, err := auth.HashPassword(adminPass)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	if err := db.SeedAdmin(database, adminHash); err != nil {
		return err
	}
	if generated {
		slog.Warn("ADMIN_PASSWORD not set; generated initial admin password (change on first login)", "username", "admin", "password", adminPass)
	}
	return nil
}
func getJWTSecret(secret string) string {
	if secret != "" {
		return secret
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return v
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("failed to generate JWT secret: %v", err)
	}
	slog.Info("generated random JWT secret")
	return hex.EncodeToString(b)
}

func main() {
	dbPath := flag.String("db", "/data/kronize.db", "Path to SQLite database")
	scriptsDir := flag.String("scripts", "/data/scripts", "Path to Python scripts directory")
	addr := flag.String("addr", getAddr(), "HTTP listen address (overrides ADDR env)")
	jwtSecret := flag.String("jwt-secret", "", "JWT signing secret (overrides JWT_SECRET env)")
	flag.Parse()

	klog.Init()

	secret := getJWTSecret(*jwtSecret)

	if err := os.MkdirAll(*scriptsDir, 0755); err != nil {
		log.Fatalf("failed to create scripts directory: %v", err)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := seedInitialAdmin(database, os.Getenv("ADMIN_PASSWORD")); err != nil {
		log.Fatalf("failed to seed admin: %v", err)
	}

	srv := server.New(database, *addr, secret, *scriptsDir)
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	slog.Info("kronize running", "addr", *addr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	slog.Info("shutting down...")
	srv.Scheduler.Stop()
}
