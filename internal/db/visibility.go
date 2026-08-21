package db

import (
	"database/sql"
	"fmt"

	"kronize/internal/model"
)

func ReplaceVisibility(db *sql.DB, jobID int64, targets []model.VisibilityTarget) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin visibility tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM job_visibility WHERE job_id = ?", jobID); err != nil {
		return fmt.Errorf("clear visibility: %w", err)
	}
	for _, t := range targets {
		switch t.Type {
		case "user":
			if _, err := tx.Exec(
				"INSERT INTO job_visibility (job_id, target_type, target_user_id) VALUES (?, 'user', ?)",
				jobID, t.UserID,
			); err != nil {
				return fmt.Errorf("insert user visibility: %w", err)
			}
		case "role":
			if _, err := tx.Exec(
				"INSERT INTO job_visibility (job_id, target_type, target_role) VALUES (?, 'role', ?)",
				jobID, t.Role,
			); err != nil {
				return fmt.Errorf("insert role visibility: %w", err)
			}
		}
	}
	return tx.Commit()
}

func GetVisibility(db *sql.DB, jobID int64) ([]model.VisibilityTarget, error) {
	rows, err := db.Query(
		`SELECT v.target_type, COALESCE(v.target_user_id, 0), COALESCE(v.target_role, ''),
		        COALESCE(u.username, '')
		 FROM job_visibility v
		 LEFT JOIN users u ON u.id = v.target_user_id
		 WHERE v.job_id = ?
		 ORDER BY v.target_type, v.target_user_id`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("get visibility: %w", err)
	}
	defer rows.Close()

	var targets []model.VisibilityTarget
	for rows.Next() {
		var t model.VisibilityTarget
		if err := rows.Scan(&t.Type, &t.UserID, &t.Role, &t.Username); err != nil {
			return nil, fmt.Errorf("scan visibility: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func UserCanAccessJob(db *sql.DB, jobID, userID int64, role string) (bool, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM jobs j
		 WHERE j.id = ?
		   AND (j.created_by = ?
		        OR EXISTS (SELECT 1 FROM job_visibility v
		                   WHERE v.job_id = j.id
		                     AND ((v.target_type = 'user' AND v.target_user_id = ?)
		                          OR (v.target_type = 'role' AND v.target_role = ?))))`,
		jobID, userID, userID, role,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("access check: %w", err)
	}
	return count > 0, nil
}

func VisibleJobSubquery(userID int64, role string) (string, []interface{}) {
	return `(SELECT j.id FROM jobs j
	         WHERE j.created_by = ?
	            OR EXISTS (SELECT 1 FROM job_visibility v
	                       WHERE v.job_id = j.id
	                         AND ((v.target_type = 'user' AND v.target_user_id = ?)
	                              OR (v.target_type = 'role' AND v.target_role = ?))))`,
		[]interface{}{userID, userID, role}
}
