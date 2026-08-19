package runner

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
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
		slog.Warn("worker queue full, dropping job", "job_id", job.ID)
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
		slog.Error("failed to create execution", "job_id", job.ID, "error", err)
		return
	}
	slog.Info("executing job", "job_id", job.ID, "name", job.Name, "execution_id", exe.ID)

	args := []string{"run", "--rm", "-i"}
	args = append(args, "--name", fmt.Sprintf("kronize-job-%d-%d", job.ID, exe.ID))

	var envVars map[string]string
	if err := json.Unmarshal([]byte(job.EnvVars), &envVars); err == nil {
		for k, v := range envVars {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
	}

	if job.TokenFile != "" {
		args = append(args, "-v", job.TokenFile+":/root/.cspm_msal_token.json:ro")
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
	args = append(args, image)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = strings.NewReader(job.PythonCode)
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
		slog.Error("failed to complete execution", "execution_id", exe.ID, "error", err)
	}

	if status == "failed" {
		notifier.SendTeamsNotification(r.db, job, exe.ID, stderr.String(), duration)
	}

	slog.Info("job complete", "job_id", job.ID, "status", status, "duration_ms", duration, "exit_code", exitCode)
}

func (r *Runner) failExecution(execID int64, stdout, stderr string) {
	duration := int64(0)
	exitCode := -1
	if err := db.CompleteExecution(r.db, execID, "failed", stdout, stderr, exitCode, duration); err != nil {
		slog.Error("failed to record failure for execution", "execution_id", execID, "error", err)
	}
}
