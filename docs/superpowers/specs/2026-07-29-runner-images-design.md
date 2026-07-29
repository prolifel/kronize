# Runner Images — Admin CRUD + User Image Selection

## Overview

Add a "Runner" admin menu for managing a pre-defined list of Docker images that users can
pick when creating/editing jobs. The image field is currently hidden from the form and only
set via DB default; this spec exposes it as a dropdown.

## Backend

### New table: `runner_images`

```sql
CREATE TABLE IF NOT EXISTS runner_images (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    image       TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

No FK from `jobs.image` — keep as free-text string for backward compat. Existing jobs with
custom images continue to work.

### New API routes (admin-only group)

| Method | Path                  | Handler             | Purpose         |
|--------|-----------------------|---------------------|-----------------|
| GET    | /api/runners          | ListImages          | list all        |
| POST   | /api/runners          | CreateImage         | create          |
| PUT    | /api/runners/{id}     | UpdateImage         | update          |
| DELETE | /api/runners/{id}     | DeleteImage         | delete          |

`GET /api/runners` is not behind admin middleware — it goes in the public group so all
authenticated users can fetch the list for the dropdown. The other three routes are
admin-only.

### New files

- `internal/model/runner_image.go` — `RunnerImage` struct, `CreateRunnerImageRequest`, `UpdateRunnerImageRequest`
- `internal/db/runner_images.go` — CRUD functions (ListRunnerImages, CreateRunnerImage, UpdateRunnerImage, DeleteRunnerImage, GetRunnerImageByID)
- `internal/handler/runner_image_handler.go` — HTTP handlers

### Changes to existing files

- `internal/db/db.go` — add `runner_images` table to `Migrate` schema
- `internal/server/server.go` — mount `GET /api/runners` in public group, mount `POST/PUT/DELETE /api/runners/{id}` in admin group

### Runner model

```go
type RunnerImage struct {
    ID          int64     `json:"id"`
    Name        string    `json:"name"`
    Image       string    `json:"image"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
}
```

## Frontend

### New types

```ts
interface RunnerImage { id: number; name: string; image: string; description: string; created_at: string }
```

### New API methods

```ts
listRunners: () => request<RunnerImage[]>('/runners')
createRunner: (data: { name: string; image: string; description: string }) => request<RunnerImage>('/runners', ...)
updateRunner: (id: number, data: Partial<RunnerImage>) => request<RunnerImage>(...)  
deleteRunner: (id: number) => request<...>('/runners/{id}', DELETE)
```

### New page: `RunnersPage.tsx`

CRUD table patterned after UsersPage:

- Table columns: ID, Name, Image, Description, Created At, Actions
- Inline edit for name/image/description
- Delete with confirm
- "Add Runner" button at top

### Nav changes in `Layout.tsx`

Add "Runner" link next to "Users" in the admin-only block.

### Job form changes in `JobFormPage.tsx`

- Add image dropdown field, populated from `api.listRunners()` on mount
- Add `image` to `JobFormData` type
- On edit: pre-select the job's current image
- On create: default to blank or first option

### `api.ts`

Add methods listed above.

## Edge Cases

- **Runner menu shows "No images" empty state** when table is empty
- **User creates job without an image selected** — the dropdown has a "Default" option that maps to empty string, keeping backward compat
- **Admin deletes an image that's in use by a job** — allowed (no FK). The job still runs with whatever image string it has stored; the dropdown just won't have that option anymore
- **Migration order** — new table created in the existing `Migrate()` call, no separate migration needed

## Not In Scope (v1)

- Image usage tracking (which jobs use which image)
- Image tags/versions as separate fields (single `image` string only)
- Registry authentication or image pull configuration
- Sorting/filtering on the runner list
