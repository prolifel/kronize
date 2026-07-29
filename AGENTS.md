# Repository Guidelines

## Project Structure & Module Organization

- **`main.go`** — entry point, flag parsing, dependency wiring
- **`internal/auth/`** — JWT token generation/validation, bcrypt hashing, middleware (auth.AdminOnly, auth.RequireAuth)
- **`internal/db/`** — SQLite CRUD for all five models, test helpers (`setupDB`), migrations
- **`internal/handler/`** — HTTP handlers per resource (job, execution, user, runner, settings, auth, stats)
- **`internal/model/`** — Go structs matching SQLite schema and JSON request/response types
- **`internal/notifier/`** — Teams webhook notification for failed jobs
- **`internal/runner/`** — Docker job executor with 4-goroutine worker pool
- **`internal/scheduler/`** — Cron scheduler using `robfig/cron/v3`
- **`internal/server/`** — chi router wiring, route registration, CORS, server lifecycle
- **`frontend/`** — Vite + React 19 + TypeScript SPA (`src/pages/`, `src/components/`, `src/api.ts`)
- **`docker/python-runner/`** — Python runner Dockerfile (python:3.12-slim + boto3, google-api-client, bs4)
- **`deploy/compose.yml`** — Docker Compose deployment for Proxmox LXC
- **`scripts/`** — utility scripts (SA key rotation, user suspension)

## Architecture Overview

Kronize is a single Go binary (chi router) serving a React SPA and REST API on `:8080`.

On startup, the cron scheduler loads all enabled jobs from SQLite and registers them with `robfig/cron/v3`. On each tick, the job is enqueued to a buffered channel consumed by a 4-goroutine worker pool. Each worker writes the job's Python code to a file, executes it via `docker run --rm` using the configured runner image, captures stdout/stderr/exit code/duration, and records the result in the `executions` table. On failure, a Teams webhook notifier fires.

A background ticker runs every hour, purging executions older than 30 days.

### Data Model (SQLite)

| Table | Key Columns |
|-------|-------------|
| `users` | id, username (UNIQUE), password_hash, role (user\|admin), must_change_password, created_at |
| `jobs` | id, name, cron_expression, python_code, image, env_vars (JSON), enabled, created_by (FK→users) |
| `executions` | id, job_id (FK→jobs, CASCADE), status (running\|success\|failed), stdout, stderr, exit_code, duration_ms |
| `settings` | key (PK), value |
| `runner_images` | id, name, image, description, created_at |

### API Routes

All under `/api` with JWT auth middleware. Admin-only routes (users, runners) are gated by `auth.AdminOnly`. See `internal/server/server.go` for the full route table.

## Build, Test, and Development Commands

All commands from repository root via `make`:

| Command | Purpose |
|---------|---------|
| `make build` | Build frontend + Go binary |
| `make dev` | `make build` + run locally with dev-secret |
| `make dev-api` | Go only (API), use alongside `make dev-ui` for Vite hot-reload |
| `make dev-ui` | Vite dev server (port 5173, proxies /api to :8080) |
| `make test` | `go test ./internal/db/ -v` |
| `make check` | `make test` + `go vet` + TypeScript type-check + frontend build |
| `make docker-build` | Build python-runner Docker image |
| `make redeploy` | SSH to Proxmox LXC → `docker compose pull + up -d` |
| `make clean` | Remove binary, `frontend/dist/`, `data/` |

## Coding Style & Naming Conventions

- **Go**: `gofmt` style. Snake_case for filenames (`job_handler.go`). Exported symbols get doc comments. Error handling is explicit — no silent swallows.
- **TypeScript/React**: PascalCase for components and their files (`JobFormPage.tsx`). camelCase for functions and variables. Tailwind CSS for all styling — no separate CSS modules. Props are typed via interfaces in the same file or `src/types.ts`.
- **Python runner scripts**: snake_case. Single-file scripts. stdout/stderr captured by runner — no interactive input.
- Pre-commit hooks handle formatting — do not run formatters manually.

## Testing Guidelines

- **Go tests** use the standard `testing` package. A `setupDB(t *testing.T)` helper in `internal/db/db_test.go` provisions a temp SQLite database with migrations applied.
- Run `make test` to execute all tests. New DB queries or model changes should include a corresponding test.
- Test naming: `Test` prefix + function or feature under test (`TestOpenAndMigrate`, `TestGetEnabledJobs`).
- Pattern for DB tests: call `setupDB(t)`, execute queries, assert results. Cleanup via `t.Cleanup`.
- Frontend testing is not yet established in this project.

## Commit & Pull Request Guidelines

- **Prefixes** observed in history: `feat:`, `fix:`, `bump:`, `plan:`, `spec:`, `docs:`.
- PR descriptions should explain the *what* and *why*, link any design docs or issues, and note migration steps if the schema changes.
- Tags trigger CI builds: `kronize-v*` builds and pushes the Kronize binary image; `python-runner-v*` builds the runner image. Pushes to `cr.prolifel.com`.
- Format: `<prefix>: <short description>` — keep the first line under 72 characters.

## Security & Configuration Tips

- **`JWT_SECRET`** and **`ADMIN_PASSWORD`** are required env vars — never hardcode, commit, or log them. The binary falls back to a random JWT secret only if neither flag nor env var is set (logged at startup), which invalidates all sessions on restart.
- **`ADMIN_PASSWORD`** sets the initial admin password. On first run, the admin user is seeded with `must_change_password = true`, forcing a password change on login. Default is `admin` if unset — change in production.
- Docker socket (`/var/run/docker.sock`) is bind-mounted at runtime for job container execution. The kronize container needs `DOCKER_GID` matching the host's Docker group ID to avoid permission errors.
- Tokens expire after 72 hours and are stored in memory (not localStorage) on the frontend — page reload requires re-login.
- Database file at `/data/kronize.db` inside the container. Mount a persistent volume in production.
- Job `python_code` is written to disk in `--scripts` directory before execution — ensure that directory is not web-accessible.

## Agent-Specific Instructions

When modifying this repository as an AI agent:

- **Read before writing**: Before editing a Go handler or DB query, read the corresponding model struct in `internal/model/` first — response types and DB schema must stay in sync.
- **Frontend-backend contract**: Any new API endpoint needs entries in three places: handler (`internal/handler/`), route registration (`internal/server/server.go`), and API client (`frontend/src/api.ts`). Missing any one breaks the feature.
- **SQLite concurrency**: SQLite does not handle concurrent writes well. Batch mutations in transactions and avoid long-held write locks. See `6f61b46` for a previous fix — `database/sql` with `?mode=rw` and `_journal_mode=WAL` can help.
- **Job execution flow**: Adding pre- or post-execution hooks? Modify `internal/runner/runner.go`. Adding notification channels? Modify `internal/notifier/notifier.go`. Do not mix the two.
- **Version bumps**: Update the version in `main.go` (via a constant or tag), tag with the appropriate prefix (`kronize-v*` or `python-runner-v*`), and let CI handle the rest. Docker Compose in `deploy/compose.yml` references exact tags — update those too.
- **Dependencies**: Pin all versions exactly — no `^`, `~`, or `latest`. Go modules are managed via `go.mod`; frontend deps via `package.json`. Check both when adding a dependency.
