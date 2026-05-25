package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"kronize/internal/db"
	"kronize/internal/server"
)

func main() {
	dbPath := flag.String("db", "/data/kronize.db", "Path to SQLite database")
	scriptsDir := flag.String("scripts", "/data/scripts", "Path to Python scripts directory")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	jwtSecret := flag.String("jwt-secret", "", "JWT signing secret (default: auto-generated)")
	flag.Parse()

	secret := *jwtSecret
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatalf("failed to generate JWT secret: %v", err)
		}
		secret = hex.EncodeToString(b)
		log.Println("generated random JWT secret")
	}

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
