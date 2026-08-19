# Host Mappings + Docker SDK Refactor

**Date:** 2026-08-19
**Status:** Approved (design)

## Goal

Generalize the job-to-container file mounting (replacing the single `token_file` concept) with a configurable list of read-only host-to-container mappings, and replace the shell-out `docker run` execution with the official Docker Engine SDK for Go.

**Supersedes:** `docs/superpowers/specs/2026-08-19-job-token-file-mount-design.md` (token_file was committed but never shipped to a running binary).

## Requirements

- Job stores an optional JSON list of host mappings: host path, container path, read-only flag.
- Runner creates/mounts all listed mappings when starting the container.
- Runner executes containers via `github.com/docker/docker` client (pinned) — no `docker` CLI shell-out.
- Observable behavior preserved: stdin python code, captured stdout/stderr, real exit code, duration, 10-minute timeout, auto-remove on exit, auto-pull on image-not-found.
- Unset mappings (default `[]`) -> no mounts, existing behavior unchanged.

## Design

### Model/DB

1. Migration (in `internal/db/db.go`): if `host_mappings` column missing, `ALTER TABLE jobs ADD COLUMN host_mappings TEXT DEFAULT '[]'`; if `token_file` column exists, drop it via `ALTER TABLE jobs DROP COLUMN token_file`.
2. `internal/model/job.go`: replace `TokenFile` with `HostMappings string \`json:"host_mappings"\`` on `Job`, `CreateJobRequest`, `UpdateJobRequest` (JSON string mirroring the `env_vars` pattern). No new struct for the array — the runner unmarshals.
3. `internal/db/jobs.go`: `jobCols`, `CreateJob`, `UpdateJob`, and both scans swap `token_file` -> `host_mappings`.
4. Tests: `TestJobHostMappings` round-trip (create/list/update-clear), mirroring `TestJobTokenFile` (which is replaced).

### Runner (Docker SDK)

`internal/runner/runner.go` `runJob` rewrite:

- `client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())` per run.
- Unmarshal `job.HostMappings` into `[]struct{ Host, Container string; ReadOnly bool }`; build `mounts.Mount{Type: "bind", Source, Target, ReadOnly}`.
- `ImagePull` first — if it fails with image-not-found semantics, report in stderr; else use existing local image.
- `ContainerCreate` with `config.Config{Image, Env, OpenStdin, StdinOnce}` + `hostconfig.HostConfig{Mounts, AutoRemove: true}`; name `kronize-job-<jobID>-<execID>`.
- `ContainerAttach` (stream) to write `job.PythonCode` to stdin and stream stdout/stderr into buffers.
- `ContainerStart`; `ContainerWait` for exit; enforce 10-minute timeout via context — on timeout `ContainerKill`; `ContainerRemove` fallback if auto-remove did not fire.
- Exit code from wait result — signal-killed shows 137, never `-1`.

### Frontend

1. `frontend/src/types.ts`: `host_mappings: string` on `Job` and `JobFormData` (replacing `token_file`).
2. `frontend/src/pages/JobFormPage.tsx`: replace token-file input with a Host Mappings editor — rows of host path, container path, read-only checkbox, add/remove buttons; mirrors the existing `EnvVarEditor` pattern. State kept as JSON string in the form.

## Dependency

- Add pinned `github.com/docker/docker` to `go.mod` (exact version, no range). No ARCHITECTURE.md exists in the repo — the plan adds a minimal Dependencies table section.

## Verification

- `make test` — DB migration + host_mappings round-trip.
- `make check` — vet + TS type-check + frontend build.
- Docker daemon is DOWN on the dev machine — live container run is a manual verification step, not part of automated gates.

## Out of Scope

- Multi-image registry auth flows beyond image pull.
- Per-mapping non-read-only writes (all mappings read-only).
- Container networking, ports, resource limits.
