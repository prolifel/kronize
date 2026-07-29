# Kronize — Context File

## Overview

Kronize is a self-hosted cron management system ("lambda on baremetal"). Users write Python code via a web dashboard, schedule it with cron expressions, and the system executes jobs in isolated Docker containers. Targeted at infrastructure automation (e.g., cleansing stale Google Workspace users, rotating GCP/AWS service account keys).

## Tech Stack

- **Backend**: Go 1.25 (`chi` router, `robfig/cron/v3`, `golang-jwt/v5`, `modernc.org/sqlite`, `golang.org/x/crypto`)
- **Frontend**: React 19, React Router 7, Tailwind CSS 3, TypeScript 5.5, Vite 8
- **Deployment**: Docker multi-stage build → Proxmox LXC (via compose), GitHub Actions CI
- **Python Runner**: `kronize/python-runner` image (Alpine + Python 3 + requests/bs4/psycopg2)

## Architecture

```
kronize (single Go binary)
├── HTTP server :8080
│   ├── Vite static files (SPA frontend)
│   └── REST API (/api/*)
├── Cron scheduler (goroutine)
│   ├── Load enabled jobs from SQLite on start
│   ├── Register cron expressions with robfig/cron
│   └── On tick → enqueue job to worker channel
├── Job runner (worker pool, 4 goroutines)
│   ├── Read job from channel
│   ├── Create Execution record in SQLite
│   ├── docker run --rm with job env vars + script mount
│   ├── Capture stdout/stderr/exit code/duration
│   ├── Record results
│   └── On failure → Teams webhook notification
└── Cleanup ticker (every 1h)
    └── DELETE executions WHERE created_at < now - 30d
```

## API Routes

All routes under `/api` with JWT auth middleware:

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/auth/login | Login |
| POST | /api/auth/logout | Logout |
| GET | /api/auth/me | Current user |
| POST | /api/auth/change-password | Change password |
| GET | /api/jobs | List jobs (?enabled=true) |
| POST | /api/jobs | Create job |
| GET | /api/jobs/{id} | Get job |
| PUT | /api/jobs/{id} | Update job |
| DELETE | /api/jobs/{id} | Delete job |
| POST | /api/jobs/{id}/run | Trigger immediate run |
| PUT | /api/jobs/{id}/toggle | Toggle enabled |
| GET | /api/jobs/{id}/executions | List executions |
| GET | /api/executions/{id} | Get execution detail |
| GET | /api/stats | Dashboard stats |
| GET | /api/settings | Get settings |
| PUT | /api/settings | Update settings |
| GET | /api/runners | List runner images (admin) |
| POST | /api/runners | Create runner image (admin) |
| PUT | /api/runners/{id} | Update runner image (admin) |
| DELETE | /api/runners/{id} | Delete runner image (admin) |
| GET | /api/users | List users (admin) |
| POST | /api/users | Create user (admin) |
| PUT | /api/users/{id}| Update user (admin) |
| DELETE | /api/users/{id}| Delete user (admin) |

## Data Model (SQLite)

### users
`id`, `username` (UNIQUE), `password_hash`, `role` (user|admin), `must_change_password`, `created_at`

### jobs
`id`, `name`, `description`, `cron_expression`, `python_code`, `image` (docker image), `env_vars` (JSON object), `log_level`, `enabled`, `created_by` (FK→users), `created_at`, `updated_at`

### executions
`id`, `job_id` (FK→jobs, CASCADE), `status` (running|success|failed), `stdout`, `stderr`, `exit_code`, `duration_ms`, `started_at`, `finished_at`

### settings
`key` (PK), `value` — stores config like `teams_webhook_url`

### runner_images
`id`, `name`, `image`, `description`, `created_at`

## Frontend Routes

| Route | Page | Description |
|-------|------|-------------|
| /login | LoginPage | Auth |
| /change-password | ChangePasswordPage | Force password change |
| /dashboard | DashboardPage | Stats cards |
| /jobs | JobListPage | Job table |
| /jobs/new | JobFormPage | Create job |
| /jobs/:id | JobDetailPage | Job detail + executions |
| /jobs/:id/edit | JobFormPage | Edit job |
| /executions | ExecutionsPage | All executions (aggregated) |
| /users | UsersPage | User CRUD (admin) |
| /runners | RunnersPage | Runner image CRUD (admin) |

## Key Components (Frontend)

- `Layout` — nav bar, auth guard, route-based nav links (admin gets Users + Runner)
- `CodeEditor` — styled textarea with macOS-style traffic light dots
- `CronHelper` — cron input with preset dropdown
- `EnvVarEditor` — key/value editor with Simple (table) + Advanced (KEY=VALUE text) modes
- `ExecutionLog` — stdout (green) / stderr (red) display
- `StatsCards` — 4 stat cards (total jobs, active, success rate, failures)
- `StatusBadge` — enabled/disabled pill

## Auth Flow

- JWT tokens, 72h expiry, stored in memory (not localStorage)
- Credentials: `include` for cookie-based session
- Admin-only routes gated by `auth.AdminOnly` middleware
- First-time users forced to change password via `must_change_password` flag

## Build & Run

```sh
make build        # frontend build + Go build
make dev          # build + run with dev-secret
make dev-api      # Go only (API), for frontend hot-reload
make dev-ui       # Vite dev server
make test         # go test ./internal/db/ -v
make check        # test + vet + frontend build
make docker-build # python-runner image only
```

## Deployment

- Single binary, deployed via `docker compose` on Proxmox LXC
- Docker socket bind-mounted for job execution (`/var/run/docker.sock`)
- CI: GitHub Actions builds and pushes on tags `kronize-v*` / `python-runner-v*`
- Registry: `cr.prolifel.com`
- Redeploy via `make redeploy` (SSH → pct exec → compose pull + up -d)

## Env Vars

- `JWT_SECRET` — JWT signing secret
- `ADMIN_PASSWORD` — initial admin password on first run
- `REGISTRY_URL` — Docker registry
- `ADDR` — listen address (default :8080)

## File Layout

```
kronize/
├── main.go                 # Entry point, flag parsing, init
├── internal/
│   ├── auth/               # JWT + bcrypt + middleware
│   ├── db/                 # SQLite CRUD for all models
│   ├── handler/            # HTTP handlers (chi)
│   ├── model/              # Go structs + request/response types
│   ├── notifier/           # Teams webhook notifier
│   ├── runner/             # Docker job executor (worker pool)
│   ├── scheduler/          # Cron scheduler (robfig/cron)
│   └── server/             # Server wiring, route registration
├── frontend/
│   ├── src/
│   │   ├── pages/          # One page per route
│   │   ├── components/     # Reusable UI components
│   │   ├── api.ts          # Typed API client
│   │   ├── types.ts        # TypeScript interfaces
│   │   ├── AuthContext.tsx  # Auth state provider
│   │   └── App.tsx         # Route definitions
│   └── ...
├── docker/python-runner/   # Python runner Dockerfile
├── deploy/compose.yml      # Docker compose deployment
├── scripts/                # Utility scripts (SA key rotation, user suspension)
└── Makefile
```
