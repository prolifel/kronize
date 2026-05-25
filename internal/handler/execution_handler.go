package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"kronize/internal/db"
	"kronize/internal/model"

	"github.com/go-chi/chi/v5"
)

func ListExecutions(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid job id")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 20
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		execs, err := db.ListExecutionsByJob(database, jobID, limit, offset)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to list executions")
			return
		}
		if execs == nil {
			execs = []*model.Execution{}
		}
		for _, e := range execs {
			if len(e.Stdout) > 200 {
				e.Stdout = e.Stdout[:200] + "..."
			}
			if len(e.Stderr) > 200 {
				e.Stderr = e.Stderr[:200] + "..."
			}
		}
		jsonResponse(w, http.StatusOK, execs)
	}
}

func GetExecution(database *sql.DB) http.HandlerFunc {
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
		jsonResponse(w, http.StatusOK, exec)
	}
}
