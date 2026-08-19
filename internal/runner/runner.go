package runner

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"kronize/internal/db"
	"kronize/internal/model"
	"kronize/internal/notifier"
)

const runTimeout = 10 * time.Minute

type hostMapping struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	ReadOnly  bool   `json:"read_only"`
}

func ensureImage(ctx context.Context, cli *client.Client, ref string) error {
	if _, _, err := cli.ImageInspectWithRaw(ctx, ref); err == nil {
		return nil
	}
	rc, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("image pull: %w", err)
	}
	defer rc.Close()
	_, err = io.Copy(io.Discard, rc)
	return err
}

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

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		r.failExecution(exe.ID, "", "docker client: "+err.Error(), time.Since(start).Milliseconds())
		return
	}
	defer cli.Close()

	imageName := job.Image
	if imageName == "" {
		imageName = "kronize/python-runner"
	}

	var env []string
	var envVars map[string]string
	if err := json.Unmarshal([]byte(job.EnvVars), &envVars); err == nil {
		for k, v := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	var mounts []mount.Mount
	var mappings []hostMapping
	if err := json.Unmarshal([]byte(job.HostMappings), &mappings); err == nil {
		for _, m := range mappings {
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   m.Host,
				Target:   m.Container,
				ReadOnly: m.ReadOnly,
			})
		}
	}

	if err := ensureImage(ctx, cli, imageName); err != nil {
		r.failExecution(exe.ID, "", err.Error(), time.Since(start).Milliseconds())
		return
	}

	createResp, err := cli.ContainerCreate(ctx,
		&container.Config{Image: imageName, Env: env, OpenStdin: true, StdinOnce: true},
		&container.HostConfig{Mounts: mounts, AutoRemove: true},
		nil, nil,
		fmt.Sprintf("kronize-job-%d-%d", job.ID, exe.ID),
	)
	if err != nil {
		r.failExecution(exe.ID, "", "container create: "+err.Error(), time.Since(start).Milliseconds())
		return
	}

	attach, err := cli.ContainerAttach(ctx, createResp.ID, container.AttachOptions{
		Stream: true, Stdin: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		r.failExecution(exe.ID, "", "container attach: "+err.Error(), time.Since(start).Milliseconds())
		return
	}

	var stdout, stderr bytes.Buffer
	streamDone := make(chan struct{})
	go func() {
		stdcopy.StdCopy(&stdout, &stderr, attach.Reader)
		close(streamDone)
	}()

	io.WriteString(attach.Conn, job.PythonCode)
	attach.CloseWrite()

	if err := cli.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		attach.Close()
		r.failExecution(exe.ID, "", "container start: "+err.Error(), time.Since(start).Milliseconds())
		return
	}

	waitCh, errCh := cli.ContainerWait(ctx, createResp.ID, container.WaitConditionNotRunning)

	exitCode := 0
	status := "success"
	select {
	case err := <-errCh:
		if err != nil {
			status = "failed"
			exitCode = -1
			stderr.WriteString("container wait: " + err.Error())
		}
	case resp := <-waitCh:
		if resp.Error != nil {
			status = "failed"
			exitCode = -1
			stderr.WriteString(resp.Error.Message)
		} else {
			exitCode = int(resp.StatusCode)
			if exitCode != 0 {
				status = "failed"
			}
		}
	case <-ctx.Done():
		status = "failed"
		exitCode = 137
		stderr.WriteString("job timed out after " + runTimeout.String())
		killCtx, killCancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = cli.ContainerKill(killCtx, createResp.ID, "SIGKILL")
		_ = cli.ContainerRemove(killCtx, createResp.ID, container.RemoveOptions{Force: true})
		killCancel()
	}

	attach.Close()
	<-streamDone

	duration := time.Since(start).Milliseconds()
	if err := db.CompleteExecution(r.db, exe.ID, status, stdout.String(), stderr.String(), exitCode, duration); err != nil {
		slog.Error("failed to complete execution", "execution_id", exe.ID, "error", err)
	}

	if status == "failed" {
		notifier.SendTeamsNotification(r.db, job, exe.ID, stderr.String(), duration)
	}

	slog.Info("job complete", "job_id", job.ID, "status", status, "duration_ms", duration, "exit_code", exitCode)
}

func (r *Runner) failExecution(execID int64, stdout, stderr string, duration int64) {
	if err := db.CompleteExecution(r.db, execID, "failed", stdout, stderr, -1, duration); err != nil {
		slog.Error("failed to record failure for execution", "execution_id", execID, "error", err)
	}
}
