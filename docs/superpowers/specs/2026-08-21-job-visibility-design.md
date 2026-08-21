# Job Visibility & Sharing

**Date:** 2026-08-21
**Status:** Approved (design)

## Goal

Replace "admin sees all jobs" with explicit visibility. Job creator chooses who can access a job: the `admin` role, specific users, or a combination. Admins no longer see everything by default.

## Requirements

- A job is visible to its owner (creator) always.
- Creator can share the job to the `admin` role (all admin users) and/or to specific users, in any combination.
- Shared users get **full control**: view, run, edit, delete, toggle.
- Only the owner manages the share list; shared users cannot re-share or change ownership.
- Admins see only: their own jobs, jobs shared to them personally, and jobs shared to the `admin` role.
- Existing jobs become owner-only (no implicit admin visibility).
- Visibility applies everywhere job data is exposed: list, get, update, delete, run, toggle, executions, execution stream, stats.
- Scheduler behavior unchanged: cron still runs every enabled job regardless of visibility.

## Design

### Data Model

```sql
CREATE TABLE IF NOT EXISTS job_visibility (
    job_id          INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    target_type     TEXT NOT NULL CHECK (target_type IN ('user','role')),
    target_user_id  INTEGER REFERENCES users(id),   -- set when target_type='user'
    target_role     TEXT,                           -- set when target_type='role' ('admin')
    PRIMARY KEY (job_id, target_type,
                 COALESCE(target_user_id, -1),
                 COALESCE(target_role, ''))
);
```

- No changes to the `jobs` table. Existing jobs have no `job_visibility` rows → visible to owner only.
- Rows: `('user', 5, NULL)` = user id 5; `('role', NULL, 'admin')` = all admins.
- `ON DELETE CASCADE` cleans up shares when a job is deleted.

### Access Rule

User U with role R can access job J if any of:

1. `J.created_by = U.id` (owner)
2. `job_visibility` row: `job_id = J.id AND target_type = 'user' AND target_user_id = U.id`
3. `job_visibility` row: `job_id = J.id AND target_type = 'role' AND target_role = R`

Read access failure → 404 (existing behavior). Write access failure → 403 (existing behavior).

### Backend

1. `internal/db/jobs.go`:
   - New `jobVisibility` helpers: set/replace visibility for a job, fetch visibility list for a job.
   - `ListJobs` drops the `role == "admin"` bypass; the visibility filter (owner OR shared) applies to every user.
   - `GetJobByID` unchanged; callers that need access checks use the shared `canAccessJob` logic.
   - `ListEnabledJobs` (scheduler path) keeps querying all enabled jobs without the visibility filter.
2. `internal/db/executions.go`:
   - `ListExecutionsByJob`, `GetExecution` unchanged at DB level; access gating happens in handlers.
3. `internal/handler/job_handler.go`:
   - `canAccessJob` applies the access rule above; `forWrite` keeps 404 vs 403 semantics.
   - `CreateJob` accepts optional `visibility` array.
   - `GetJob` returns `visibility` list with usernames resolved.
   - New `UpdateJobVisibility` handler: owner-only, replaces the whole list in one transaction.
4. `internal/handler/execution_handler.go`, `execution_stream_handler.go`:
   - `ListExecutions`, `GetExecution`, `StreamExecution` resolve the job from the execution and check access first; failures → 404.
5. `internal/handler/stats_handler.go`:
   - Stats become per-user: jobs visible to the caller (owner or shared) and their executions. No global leak.
6. New `GET /api/users/search?q=`:
   - Any authenticated user; returns `[{id, username}]` only (no roles, no hashes), username LIKE filter for autocomplete.
   - Existing admin-only `GET /api/users` unchanged.
7. `internal/model/job.go`:
   - Add `Visibility []VisibilityTarget` to `Job` and `CreateJobRequest`; new `VisibilityTarget {Type, UserID, Username, Role}` and `UpdateVisibilityRequest []VisibilityTarget`.
8. `internal/server/server.go`:
   - Register `PUT /api/jobs/{id}/visibility` and `GET /api/users/search`.

### Frontend / UX

- `frontend/src/api.ts`: `searchUsers(q)`, `updateJobVisibility(jobId, targets)`, types for `VisibilityTarget`.
- Job form/edit page: **Sharing** section with user autocomplete (from `/api/users/search`), "Share with admins" toggle, list of current shares with remove buttons.
- Job detail page: shows current shares (users + admin role).
- Job list: small shared indicator (e.g. share count) on shared jobs.

### Error Handling

- Visibility endpoint: owner only; anyone else → 403. Invalid target (unknown user id, bad type) → 400.
- Whole-list replace is atomic: one transaction (delete all rows for job, insert new ones).

### Testing

- DB: visibility rows grant access to the correct user; role row grants all admins; owner always passes; unrelated user gets nothing.
- Handler: visibility update owner OK / shared user 403; invalid target 400.
- List: user sees own + shared; admin sees own + shared-to-them + admin-role jobs only.
- Scheduler: `ListEnabledJobs` still returns all enabled jobs.
- Migration: existing jobs visible to owner only (no visibility rows created).
