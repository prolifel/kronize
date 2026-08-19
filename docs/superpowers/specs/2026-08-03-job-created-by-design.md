# Show Job Creator Username

**Date:** 2026-08-03 (updated 2026-08-19)
**Status:** Approved (design)

## Goal

Display the job creator's username instead of the raw `created_by` user ID — on the job detail page and as a "Created By" column in the job list page.

## Requirements

- API returns creator username alongside existing job data.
- Job detail page shows "Created by <username>" in the info card.
- Job list page shows a "Created By" column with the creator username.
- Existing jobs stay visible if the creator user is deleted (username falls back to empty; list page renders `-`).

## Design

### Backend

1. Add `CreatedByUsername string \`json:"created_by_username"\`` to `internal/model/job.go` `Job` struct.
2. Change `jobCols` in `internal/db/jobs.go` to `LEFT JOIN users u ON u.id = j.created_by` and select `u.username`; scan into `CreatedByUsername`. Applies to `ListJobs`, `GetJobByID`, and `GetEnabledJobs` (shared `jobCols`/scan helper).
3. No schema migration — `users` table already exists; join is read-only.

### Frontend

1. Add `created_by_username: string` to `Job` in `frontend/src/types.ts`.
2. In `frontend/src/pages/JobDetailPage.tsx` info card, add third `dl` entry: `Created by` → `{job.created_by_username}`.
3. In `frontend/src/pages/JobListPage.tsx` table, add `Created By` column after Status: `{job.created_by_username || '-'}`.

## Verification

- `make test` (DB query regression).
- `make check` (vet + TS type-check + frontend build).

## Out of Scope

- Permission changes — `canAccessJob` logic untouched.
