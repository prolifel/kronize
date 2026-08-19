# Job Token File Mount

**Date:** 2026-08-19
**Status:** Approved (design)

## Goal

Let a job mount one host file into the runner container so scripts can read persistent secrets/tokens (e.g. `~/.cspm_msal_token.json`), which are otherwise absent in the ephemeral `docker run --rm` container.

## Requirements

- Job stores an optional host-side token file path.
- When set, the runner mounts that file read-only into the container at `/root/.cspm_msal_token.json`.
- Unset token file (default) → no mount, existing behavior unchanged.
- Admin-only? No — jobs already execute arbitrary code in containers; one read-only host file adds no meaningful escalation.

## Design

### Backend

1. Migration: add `token_file TEXT DEFAULT ''` to `jobs`. Follow the existing pragma-table-info pattern in `internal/db/db.go` (used for the `image` → `image_id` migration): check column exists, `ALTER TABLE jobs ADD COLUMN token_file TEXT DEFAULT ''` if missing.
2. `internal/model/job.go`: add `TokenFile string \`json:"token_file"\`` to `Job`, `CreateJobRequest`, `UpdateJobRequest`.
3. `internal/db/jobs.go`: add `j.token_file` to `jobCols`, include in `CreateJob` INSERT, handle pointer field in `UpdateJob`, scan in both scan sites.
4. `internal/runner/runner.go`: after env-var args, if `job.TokenFile != ""` append `-v`, `<token_file>:/root/.cspm_msal_token.json:ro`.

### Frontend

1. `frontend/src/types.ts`: `token_file: string` on `Job` and `JobFormData`.
2. `frontend/src/pages/JobFormPage.tsx`: optional text input "Token file (host path)" wired into form state and submit payload.

## Verification

- `make test` — DB test: create job with `token_file`, assert round-trip via `GetJobByID`/`ListJobs`; update clears it.
- `make check` — vet + TS type-check + frontend build.

## Out of Scope

- Multiple volumes / general `volumes` list.
- Configurable in-container destination path.
- Detail-page display of token file path (can be added later; field is in API).
