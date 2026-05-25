package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	dbPath := flag.String("db", "/data/kronize.db", "Path to SQLite database")
	scriptsDir := flag.String("scripts", "/data/scripts", "Path to Python scripts directory")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	jwtSecret := flag.String("jwt-secret", "", "JWT signing secret (default: auto-generated)")
	_ = jwtSecret // used later for JWT middleware
	flag.Parse()

	log.Printf("kronize starting - db=%s scripts=%s addr=%s", *dbPath, *scriptsDir, *addr)

	// TODO: init DB, load scheduler, start HTTP server

	// Wait for SIGINT/SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("kronize shutting down")
}
