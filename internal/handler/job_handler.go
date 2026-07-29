package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"
	"kronize/internal/scheduler"

	"github.com/go-chi/chi/v5"
)

type accessError struct {
	Code int
	Err  error
}

func (e *accessError) Error() string { return e.Err.Error() }

func canAccessJob(database *sql.DB, userID int64, role string, jobID int64, forWrite bool) (*model.Job, *accessError) {
	if role == "admin" {
		job, err := db.GetJobByID(database, jobID)
		if err != nil {
			return nil, &accessError{Code: http.StatusNotFound, Err: fmt.Errorf("job not found")}
		}
		return job, nil
	}
	job, err := db.GetJobByID(database, jobID)
	if err != nil {
		return nil, &accessError{Code: http.StatusNotFound, Err: fmt.Errorf("job not found")}
	}
	if job.CreatedBy != userID {
		if forWrite {
			return nil, &accessError{Code: http.StatusForbidden, Err: fmt.Errorf("forbidden")}
		}
		return nil, &accessError{Code: http.StatusNotFound, Err: fmt.Errorf("job not found")}
	}
	return job, nil
}

func accessErrorJSON(w http.ResponseWriter, err *accessError) {
	http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Err.Error()), err.Code)
}

func ListJobs(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enabledOnly := r.URL.Query().Get("enabled") == "true"
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		jobs, err := db.ListJobs(database, enabledOnly, userID, role)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to list jobs")
			return
		}
		if jobs == nil {
			jobs = []*model.Job{}
		}
		for _, j := range jobs {
			if j.EnvVars != "" && j.EnvVars != "{}" {
				j.EnvVars = `{"masked":true}`
			}
			j.PythonCode = ""
		}
		jsonResponse(w, http.StatusOK, jobs)
	}
}

func CreateJob(database *sql.DB, sched *scheduler.Scheduler, scriptsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		var req model.CreateJobRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" || req.CronExpression == "" || req.PythonCode == "" {
			jsonError(w, http.StatusBadRequest, "name, cron_expression, and python_code required")
			return
		}
		job, err := db.CreateJob(database, req, userID)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to create job")
			return
		}
		sched.AddJob(job)
		job.EnvVars = ""
		jsonResponse(w, http.StatusCreated, job)
	}
}

func GetJob(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid job id")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		job, aerr := canAccessJob(database, userID, role, id, false)
		if aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
		jsonResponse(w, http.StatusOK, job)
	}
}

func UpdateJob(database *sql.DB, sched *scheduler.Scheduler, scriptsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid job id")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		_, aerr := canAccessJob(database, userID, role, id, true)
		if aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
		var req model.UpdateJobRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		job, err := db.UpdateJob(database, id, req)
		if err != nil {
			jsonError(w, http.StatusNotFound, "job not found")
			return
		}
		sched.UpdateJob(job)
		job.PythonCode = ""
		job.EnvVars = ""
		jsonResponse(w, http.StatusOK, job)
	}
}

func DeleteJob(database *sql.DB, sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid job id")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		_, aerr := canAccessJob(database, userID, role, id, true)
		if aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
		if err := db.DeleteJob(database, id); err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to delete job")
			return
		}
		sched.RemoveJob(id)
		jsonResponse(w, http.StatusOK, map[string]string{"message": "deleted"})
	}
}

func RunJob(database *sql.DB, sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid job id")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		job, aerr := canAccessJob(database, userID, role, id, true)
		if aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
		sched.TriggerNow(job)
		jsonResponse(w, http.StatusAccepted, map[string]string{"message": "job triggered"})
	}
}

func ToggleJob(database *sql.DB, sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid job id")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		_, aerr := canAccessJob(database, userID, role, id, true)
		if aerr != nil {
			accessErrorJSON(w, aerr)
			return
		}
		current, err := db.GetJobByID(database, id)
		if err != nil {
			jsonError(w, http.StatusNotFound, "job not found")
			return
		}
		newEnabled := !current.Enabled
		upd := model.UpdateJobRequest{Enabled: &newEnabled}
		job, err := db.UpdateJob(database, id, upd)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to toggle job")
			return
		}
		if newEnabled {
			sched.AddJob(job)
		} else {
			sched.RemoveJob(id)
		}
		job.PythonCode = ""
		job.EnvVars = ""
		jsonResponse(w, http.StatusOK, job)
	}
}
