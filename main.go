package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/server"
)

func getAddr() string {
	if v := os.Getenv("ADDR"); v != "" {
		return v
	}
	return ":8080"
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
	log.Println("generated random JWT secret")
	return hex.EncodeToString(b)
}

func main() {
	dbPath := flag.String("db", "/data/kronize.db", "Path to SQLite database")
	scriptsDir := flag.String("scripts", "/data/scripts", "Path to Python scripts directory")
	addr := flag.String("addr", getAddr(), "HTTP listen address (overrides ADDR env)")
	jwtSecret := flag.String("jwt-secret", "", "JWT signing secret (overrides JWT_SECRET env)")
	flag.Parse()

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

	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin"
		log.Println("ADMIN_PASSWORD not set, using default: admin")
	}
	adminHash, err := auth.HashPassword(adminPass)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}
	if err := db.SeedAdmin(database, adminHash); err != nil {
		log.Fatalf("failed to seed admin: %v", err)
	}

	srv := server.New(database, *addr, secret, *scriptsDir)
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Printf("kronize running on %s", *addr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down...")
	srv.Scheduler.Stop()
}
