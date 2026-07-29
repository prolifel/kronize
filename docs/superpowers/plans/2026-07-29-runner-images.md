# Runner Images Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Admin CRUD for runner Docker images + user dropdown in job form.

**Architecture:** New `runner_images` SQLite table, admin-only API routes with `GET /api/runners` public for dropdown, new CRUD page in frontend, dropdown in JobFormPage.

**Tech Stack:** Go 1.25, sqlite (modernc), chi router, React 19, TypeScript, Vite

## Global Constraints

- All dependency versions pinned exactly — no ranges
- Existing `jobs.image` field stays as free-text string (no FK)
- Admin-only routes use existing `auth.AdminOnly` middleware
- Match existing code style (snake_case JSON, chi router patterns, handler func factory pattern)

---

### Task 1: Backend — Model, DB migration, and DB CRUD

**Files:**
- Create: `internal/model/runner_image.go`
- Create: `internal/db/runner_images.go`
- Modify: `internal/db/db.go`

**Interfaces:**
- Consumes: existing `db` package conventions, `model` package patterns
- Produces:
  - `model.RunnerImage`, `model.CreateRunnerImageRequest`, `model.UpdateRunnerImageRequest` structs
  - `db.ListRunnerImages(database *sql.DB) ([]*model.RunnerImage, error)`
  - `db.GetRunnerImageByID(database *sql.DB, id int64) (*model.RunnerImage, error)`
  - `db.CreateRunnerImage(database *sql.DB, req model.CreateRunnerImageRequest) (*model.RunnerImage, error)`
  - `db.UpdateRunnerImage(database *sql.DB, id int64, req model.UpdateRunnerImageRequest) (*model.RunnerImage, error)`
  - `db.DeleteRunnerImage(database *sql.DB, id int64) error`

- [ ] **Step 1: Add `runner_images` table to migration**

In `internal/db/db.go`, add to existing `Migrate()` schema:

```sql
CREATE TABLE IF NOT EXISTS runner_images (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    image       TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

- [ ] **Step 2: Create model file**

`internal/model/runner_image.go`:

```go
package model

import "time"

type RunnerImage struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateRunnerImageRequest struct {
	Name        string `json:"name"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type UpdateRunnerImageRequest struct {
	Name        *string `json:"name"`
	Image       *string `json:"image"`
	Description *string `json:"description"`
}
```

- [ ] **Step 3: Create DB CRUD file**

`internal/db/runner_images.go` with `ListRunnerImages`, `GetRunnerImageByID`, `CreateRunnerImage`, `UpdateRunnerImage`, `DeleteRunnerImage`.

Follow the same patterns as `internal/db/users.go` (scanning rows, returning model pointers, using `database/sql` directly).

- [ ] **Step 4: Build and verify no compile errors**

```bash
cd /Users/clement/kantor/kronize && go build ./...
```

Expected: clean build.

### Task 2: Backend — HTTP handlers and routes

**Files:**
- Create: `internal/handler/runner_image_handler.go`
- Modify: `internal/server/server.go`

**Interfaces:**
- Consumes:
  - `db.ListRunnerImages`, `db.GetRunnerImageByID`, `db.CreateRunnerImage`, `db.UpdateRunnerImage`, `db.DeleteRunnerImage`
  - `model.CreateRunnerImageRequest`, `model.UpdateRunnerImageRequest`
- Produces: HTTP handlers `ListRunnerImages`, `CreateRunnerImage`, `UpdateRunnerImage`, `DeleteRunnerImage` mounted in server routes

- [ ] **Step 1: Create handler file**

`internal/handler/runner_image_handler.go` with factory functions following the pattern in `internal/handler/user_handler.go`:
- `ListRunnerImages(database *sql.DB)` — returns all images (no auth check, just list)
- `CreateRunnerImage(database *sql.DB)` — validate name + image required, create, return 201
- `UpdateRunnerImage(database *sql.DB)` — parse ID from URL, validate at least one field, update, return 200
- `DeleteRunnerImage(database *sql.DB)` — parse ID, delete, return 200

- [ ] **Step 2: Mount routes in server.go**

In `internal/server/server.go`, add to public routes:

```go
r.Get("/runners", handler.ListRunnerImages(s.DB))
```

Add to admin routes (inside `r.Group(func(r chi.Router) { r.Use(auth.AdminOnly) ... })`):

```go
r.Post("/runners", handler.CreateRunnerImage(s.DB))
r.Put("/runners/{id}", handler.UpdateRunnerImage(s.DB))
r.Delete("/runners/{id}", handler.DeleteRunnerImage(s.DB))
```

- [ ] **Step 3: Build and verify**

```bash
cd /Users/clement/kantor/kronize && go build ./...
```

Expected: clean build.

### Task 3: Frontend — Types, API client, and RunnersPage

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Create: `frontend/src/pages/RunnersPage.tsx`
- Modify: `frontend/src/components/Layout.tsx`

**Interfaces:**
- Consumes: Existing `api.request<T>()` helper, existing Layout pattern
- Produces:
  - `RunnerImage` interface in types.ts
  - `api.listRunners()`, `api.createRunner()`, `api.updateRunner()`, `api.deleteRunner()` in api.ts
  - `RunnersPage` component rendered at `/runners`
  - "Runner" nav link in admin nav

- [ ] **Step 1: Add `RunnerImage` type**

Add to `frontend/src/types.ts`:

```ts
export interface RunnerImage {
  id: number
  name: string
  image: string
  description: string
  created_at: string
}
```

- [ ] **Step 2: Add API methods**

Add to `frontend/src/api.ts`:

```ts
listRunners: () =>
  request<RunnerImage[]>('/runners'),

createRunner: (data: { name: string; image: string; description: string }) =>
  request<RunnerImage>('/runners', { method: 'POST', body: JSON.stringify(data) }),

updateRunner: (id: number, data: { name?: string; image?: string; description?: string }) =>
  request<RunnerImage>(`/runners/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

deleteRunner: (id: number) =>
  request<{ message: string }>(`/runners/${id}`, { method: 'DELETE' }),
```

- [ ] **Step 3: Create RunnersPage**

`frontend/src/pages/RunnersPage.tsx` — CRUD table patterned after `UsersPage.tsx`:

- Admin role gate at top (same pattern)
- "Add Runner" button toggles form
- Create form with 3 fields: name, image (e.g. `python:3.12-slim`), description
- Table with columns: Name, Image, Description, Created At, Actions
- Inline edit for name/image/description
- Delete with confirm
- Loading state

- [ ] **Step 4: Add "Runner" nav link**

In `frontend/src/components/Layout.tsx`, inside the admin nav block, add after Users:

```tsx
{ user?.role === 'admin' && navLinks.push({ to: '/runners', label: 'Runner' }) }
```

Also verify the route table in `App.tsx` needs a new route entry (Task 4 handles this).

- [ ] **Step 5: Add route in App.tsx**

Add import and route in `frontend/src/App.tsx`:

```tsx
import RunnersPage from './pages/RunnersPage'
// ...
<Route path="/runners" element={<RunnersPage />} />
```

### Task 4: Frontend — Job form image dropdown

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/pages/JobFormPage.tsx`

- [ ] **Step 1: Add `image` to JobFormData**

In `frontend/src/types.ts`, add field to `JobFormData`:

```ts
export interface JobFormData {
  // ...existing fields...
  image: string
}
```

- [ ] **Step 2: Update JobFormPage state and fetch**

In `frontend/src/pages/JobFormPage.tsx`:

- Add `image: ''` to initial form state
- On mount, fetch runners via `api.listRunners()` and store in local state
- When editing, job.image is already loaded from `api.getJob()`

- [ ] **Step 3: Add image dropdown to form**

Add after the log level select in the grid:

```tsx
<div>
  <label className="block text-sm font-medium text-gray-700 mb-1">Image</label>
  <select
    value={form.image}
    onChange={(e) => setForm({ ...form, image: e.target.value })}
    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
  >
    <option value="">Default</option>
    {runners.map((r) => (
      <option key={r.id} value={r.image}>{r.name}</option>
    ))}
  </select>
</div>
```

- [ ] **Step 4: Verify build**

```bash
cd /Users/clement/kantor/kronize/frontend && npx tsc --noEmit
```

Expected: clean compile.

### Task 5: Build and smoke test

**Files:** (no changes — verify)

- [ ] **Step 1: Full Go build**

```bash
cd /Users/clement/kantor/kronize && go build ./...
```

- [ ] **Step 2: Full frontend build**

```bash
cd /Users/clement/kantor/kronize/frontend && npm run build
```

- [ ] **Step 3: Run any existing tests**

```bash
cd /Users/clement/kantor/kronize && go test ./...
```
