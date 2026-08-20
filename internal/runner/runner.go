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
	"kronize/internal/logstream"
	"kronize/internal/model"
	"kronize/internal/notifier"
)

const runTimeout = 10 * time.Minute

type logWriter struct {
	stream string
	execID int64
	hub    *logstream.Hub
	buf    *bytes.Buffer
}

func (w *logWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if err != nil {
		return n, err
	}
	if w.hub != nil && n > 0 {
		if err := w.hub.Append(w.execID, w.stream, string(p[:n])); err != nil {
			slog.Error("failed to append live log chunk", "execution_id", w.execID, "stream", w.stream, "error", err)
		}
	}
	return n, nil
}

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

type workItem struct {
	job    *model.Job
	execID int64
	source string
}

type Runner struct {
	db         *sql.DB
	scriptsDir string
	hub        *logstream.Hub
	workChan   chan workItem
}

func New(database *sql.DB, scriptsDir string, hub *logstream.Hub) *Runner {
	r := &Runner{
		db:         database,
		scriptsDir: scriptsDir,
		hub:        hub,
		workChan:   make(chan workItem, 100),
	}
	for i := 0; i < 4; i++ {
		go r.worker()
	}
	return r
}

func (r *Runner) Enqueue(job *model.Job, source string) (int64, error) {
	exe, err := db.CreateExecution(r.db, job.ID, source)
	if err != nil {
		return 0, err
	}
	item := workItem{job: job, execID: exe.ID, source: source}
	select {
	case r.workChan <- item:
	default:
		_ = db.CompleteExecution(r.db, exe.ID, "failed", "", "worker queue full", -1, 0)
		slog.Warn("worker queue full, failed job", "job_id", job.ID, "execution_id", exe.ID)
		return exe.ID, fmt.Errorf("worker queue full")
	}
	return exe.ID, nil
}

func (r *Runner) worker() {
	for item := range r.workChan {
		r.runJob(item.job, item.execID, item.source)
	}
}

func (r *Runner) runJob(job *model.Job, execID int64, source string) {
	start := time.Now()

	slog.Info("executing job", "job_id", job.ID, "name", job.Name, "execution_id", execID, "source", source)

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		r.failExecution(execID, "", "docker client: "+err.Error(), time.Since(start).Milliseconds())
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
		r.failExecution(execID, "", err.Error(), time.Since(start).Milliseconds())
		return
	}

	createResp, err := cli.ContainerCreate(ctx,
		&container.Config{Image: imageName, Env: env, OpenStdin: true, StdinOnce: true},
		&container.HostConfig{Mounts: mounts, AutoRemove: true},
		nil, nil,
		fmt.Sprintf("kronize-job-%d-%d", job.ID, execID),
	)
	if err != nil {
		r.failExecution(execID, "", "container create: "+err.Error(), time.Since(start).Milliseconds())
		return
	}

	attach, err := cli.ContainerAttach(ctx, createResp.ID, container.AttachOptions{
		Stream: true, Stdin: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		r.failExecution(execID, "", "container attach: "+err.Error(), time.Since(start).Milliseconds())
		return
	}

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	stdoutW := &logWriter{stream: "stdout", execID: execID, hub: r.hub, buf: stdoutBuf}
	stderrW := &logWriter{stream: "stderr", execID: execID, hub: r.hub, buf: stderrBuf}
	streamDone := make(chan struct{})
	go func() {
		stdcopy.StdCopy(stdoutW, stderrW, attach.Reader)
		close(streamDone)
	}()

	io.WriteString(attach.Conn, job.PythonCode)
	attach.CloseWrite()

	if err := cli.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		attach.Close()
		r.failExecution(execID, "", "container start: "+err.Error(), time.Since(start).Milliseconds())
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
			stderrBuf.WriteString("container wait: " + err.Error())
		}
	case resp := <-waitCh:
		if resp.Error != nil {
			status = "failed"
			exitCode = -1
			stderrBuf.WriteString(resp.Error.Message)
		} else {
			exitCode = int(resp.StatusCode)
			if exitCode != 0 {
				status = "failed"
			}
		}
	case <-ctx.Done():
		status = "failed"
		exitCode = 137
		stderrBuf.WriteString("job timed out after " + runTimeout.String())
		killCtx, killCancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = cli.ContainerKill(killCtx, createResp.ID, "SIGKILL")
		_ = cli.ContainerRemove(killCtx, createResp.ID, container.RemoveOptions{Force: true})
		killCancel()
	}

	attach.Close()
	<-streamDone

	duration := time.Since(start).Milliseconds()
	if err := db.CompleteExecution(r.db, execID, status, stdoutBuf.String(), stderrBuf.String(), exitCode, duration); err != nil {
		slog.Error("failed to complete execution", "execution_id", execID, "error", err)
	}

	if r.hub != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"status":      status,
			"exit_code":   exitCode,
			"duration_ms": duration,
		})
		r.hub.PublishStatus(execID, string(data))
	}

	if status == "failed" {
		notifier.SendTeamsNotification(r.db, job, execID, stderrBuf.String(), duration)
	}

	slog.Info("job complete", "job_id", job.ID, "status", status, "duration_ms", duration, "exit_code", exitCode)
}

func (r *Runner) failExecution(execID int64, stdout, stderr string, duration int64) {
	if err := db.CompleteExecution(r.db, execID, "failed", stdout, stderr, -1, duration); err != nil {
		slog.Error("failed to record failure for execution", "execution_id", execID, "error", err)
	}
	if r.hub != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"status":      "failed",
			"exit_code":   -1,
			"duration_ms": duration,
		})
		r.hub.PublishStatus(execID, string(data))
	}
}
