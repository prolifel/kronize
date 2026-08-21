package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/logstream"
)

func StreamExecution(database *sql.DB, hub *logstream.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid execution id")
			return
		}

		exec, err := db.GetExecution(database, id)
		if err != nil {
			jsonError(w, http.StatusNotFound, "execution not found")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		if _, aerr := canAccessJob(database, userID, role, exec.JobID, false); aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
		if exec.Source != "manual" {
			jsonError(w, http.StatusForbidden, "live logs only available for manual executions")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			jsonError(w, http.StatusInternalServerError, "streaming unsupported")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch, snap, cancel, err := hub.SubscribeWithSnapshot(id, func() (logstream.Snapshot, error) {
			e, err := db.GetExecution(database, id)
			if err != nil {
				return logstream.Snapshot{}, err
			}
			return logstream.Snapshot{
				Status:     e.Status,
				Stdout:     e.Stdout,
				Stderr:     e.Stderr,
				ExitCode:   e.ExitCode,
				DurationMs: e.DurationMs,
			}, nil
		})
		if err != nil {
			jsonError(w, http.StatusNotFound, "execution not found")
			return
		}
		defer cancel()

		writeSSE(w, flusher, "init", snap)

		if snap.Status != "running" {
			writeSSE(w, flusher, "status", map[string]interface{}{
				"status":      snap.Status,
				"exit_code":   snap.ExitCode,
				"duration_ms": snap.DurationMs,
			})
			return
		}

		for {
			select {
			case <-r.Context().Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if ev.Type == "status" {
					var payload interface{}
					if err := json.Unmarshal([]byte(ev.Data), &payload); err != nil {
						return
					}
					writeSSE(w, flusher, "status", payload)
					return
				}
				writeSSE(w, flusher, ev.Type, ev.Data)
			}
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload interface{}) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	flusher.Flush()
}
