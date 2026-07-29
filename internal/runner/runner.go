package runner

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"kronize/internal/db"
	"kronize/internal/model"
	"kronize/internal/notifier"
)

type Runner struct {
	db         *sql.DB
	scriptsDir string
	workChan   chan *model.Job
}

func New(database *sql.DB, scriptsDir string) *Runner {
	r := &Runner{
		db:         database,
		scriptsDir: scriptsDir,
		workChan:   make(chan *model.Job, 100),
	}
	for i := 0; i < 4; i++ {
		go r.worker()
	}
	return r
}

func (r *Runner) ExecuteJob(job *model.Job) {
	select {
	case r.workChan <- job:
	default:
		log.Printf("worker queue full, dropping job %d", job.ID)
	}
}

func (r *Runner) worker() {
	for job := range r.workChan {
		r.runJob(job)
	}
}

func (r *Runner) runJob(job *model.Job) {
	start := time.Now()

	exe, err := db.CreateExecution(r.db, job.ID)
	if err != nil {
		log.Printf("failed to create execution for job %d: %v", job.ID, err)
		return
	}
	log.Printf("executing job %d (%s): execution=%d", job.ID, job.Name, exe.ID)

	scriptPath := filepath.Join(r.scriptsDir, fmt.Sprintf("%d", job.ID), "main.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		r.failExecution(exe.ID, "", fmt.Sprintf("failed to create script dir: %v", err))
		return
	}
	if err := os.WriteFile(scriptPath, []byte(job.PythonCode), 0644); err != nil {
		r.failExecution(exe.ID, "", fmt.Sprintf("failed to write script: %v", err))
		return
	}

	scriptDir := filepath.Dir(scriptPath)
	absScriptDir, err := filepath.Abs(scriptDir)
	if err != nil {
		r.failExecution(exe.ID, "", fmt.Sprintf("failed to resolve absolute script dir: %v", err))
		return
	}

	args := []string{"run", "--rm"}
	args = append(args, "-v", fmt.Sprintf("%s:/code:ro", absScriptDir))
	args = append(args, "--name", fmt.Sprintf("kronize-job-%d-%d", job.ID, exe.ID))

	var envVars map[string]string
	if err := json.Unmarshal([]byte(job.EnvVars), &envVars); err == nil {
		for k, v := range envVars {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
	}

	image := job.Image
	if image == "" {
		reg := os.Getenv("REGISTRY_URL")
		if reg == "" {
			image = "kronize/python-runner"
		} else {
			image = reg + "/kronize/python-runner:latest"
		}
	}
	args = append(args, image, "/code/main.py")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	duration := time.Since(start).Milliseconds()

	exitCode := 0
	status := "success"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		status = "failed"
	}

	if err := db.CompleteExecution(r.db, exe.ID, status, stdout.String(), stderr.String(), exitCode, duration); err != nil {
		log.Printf("failed to complete execution %d: %v", exe.ID, err)
	}

	if status == "failed" {
		notifier.SendTeamsNotification(r.db, job, exe.ID, stderr.String(), duration)
	}

	log.Printf("job %d complete: status=%s duration=%dms exit=%d", job.ID, status, duration, exitCode)
}

func (r *Runner) failExecution(execID int64, stdout, stderr string) {
	duration := int64(0)
	exitCode := -1
	if err := db.CompleteExecution(r.db, execID, "failed", stdout, stderr, exitCode, duration); err != nil {
		log.Printf("failed to record failure for execution %d: %v", execID, err)
	}
}
