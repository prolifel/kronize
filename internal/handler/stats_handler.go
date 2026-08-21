package handler

import (
	"database/sql"
	"net/http"

	"kronize/internal/auth"
	"kronize/internal/db"
)

type StatsResponse struct {
	TotalJobs      int     `json:"total_jobs"`
	EnabledJobs    int     `json:"enabled_jobs"`
	SuccessRate24h float64 `json:"success_rate_24h"`
	FailuresToday  int     `json:"failures_today"`
}

func GetStats(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())
		sub, args := db.VisibleJobSubquery(userID, role)

		totalJobs := 0
		database.QueryRow("SELECT COUNT(*) FROM jobs WHERE id IN "+sub, args...).Scan(&totalJobs)
		enabledJobs := 0
		database.QueryRow("SELECT COUNT(*) FROM jobs WHERE enabled = 1 AND id IN "+sub, args...).Scan(&enabledJobs)
		total24h := 0
		database.QueryRow("SELECT COUNT(*) FROM executions WHERE started_at > datetime('now', '-1 day') AND job_id IN "+sub, args...).Scan(&total24h)
		failuresToday := 0
		database.QueryRow("SELECT COUNT(*) FROM executions WHERE status = 'failed' AND started_at > datetime('now', '-1 day') AND job_id IN "+sub, args...).Scan(&failuresToday)
		successRate := 0.0
		if total24h > 0 {
			successRate = float64(total24h-failuresToday) / float64(total24h) * 100
		}
		jsonResponse(w, http.StatusOK, StatsResponse{
			TotalJobs:      totalJobs,
			EnabledJobs:    enabledJobs,
			SuccessRate24h: successRate,
			FailuresToday:  failuresToday,
		})
	}
}
