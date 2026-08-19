package db

import (
	"database/sql"
	"fmt"
	"kronize/internal/model"
)

const jobCols = "j.id, j.name, j.description, j.cron_expression, j.python_code, j.image_id, ri.image AS image, j.env_vars, j.log_level, j.enabled, j.created_by, COALESCE(u.username, '') AS created_by_username, j.created_at, j.updated_at"
const jobFrom = "FROM jobs j JOIN runner_images ri ON ri.id = j.image_id LEFT JOIN users u ON u.id = j.created_by"

func CreateJob(db *sql.DB, req model.CreateJobRequest, userID int64) (*model.Job, error) {
	logLevel := req.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}
	envVars := req.EnvVars
	if envVars == "" {
		envVars = "{}"
	}
	res, err := db.Exec(
		`INSERT INTO jobs (name, description, cron_expression, python_code, image_id, env_vars, log_level, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Description, req.CronExpression, req.PythonCode, req.ImageID, envVars, logLevel, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetJobByID(db, id)
}

func GetJobByID(db *sql.DB, id int64) (*model.Job, error) {
	row := db.QueryRow(
		"SELECT "+jobCols+" "+jobFrom+" WHERE j.id = ?", id,
	)
	j := &model.Job{}
	err := row.Scan(&j.ID, &j.Name, &j.Description, &j.CronExpression, &j.PythonCode,
		&j.ImageID, &j.Image, &j.EnvVars, &j.LogLevel, &j.Enabled, &j.CreatedBy, &j.CreatedByUsername, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}

func ListJobs(db *sql.DB, enabledOnly bool, userID int64, role string) ([]*model.Job, error) {
	query := "SELECT " + jobCols + " " + jobFrom
	var args []interface{}
	var clauses []string

	if role != "admin" {
		clauses = append(clauses, "j.created_by = ?")
		args = append(args, userID)
	}
	if enabledOnly {
		clauses = append(clauses, "j.enabled = 1")
	}
	if len(clauses) > 0 {
		query += " WHERE "
		for i, c := range clauses {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}
	query += " ORDER BY j.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*model.Job
	for rows.Next() {
		j := &model.Job{}
		if err := rows.Scan(&j.ID, &j.Name, &j.Description, &j.CronExpression, &j.PythonCode,
			&j.ImageID, &j.Image, &j.EnvVars, &j.LogLevel, &j.Enabled, &j.CreatedBy, &j.CreatedByUsername, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func UpdateJob(db *sql.DB, id int64, req model.UpdateJobRequest) (*model.Job, error) {
	fields := []string{}
	args := []interface{}{}

	if req.Name != nil {
		fields = append(fields, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		fields = append(fields, "description = ?")
		args = append(args, *req.Description)
	}
	if req.CronExpression != nil {
		fields = append(fields, "cron_expression = ?")
		args = append(args, *req.CronExpression)
	}
	if req.PythonCode != nil {
		fields = append(fields, "python_code = ?")
		args = append(args, *req.PythonCode)
	}
	if req.EnvVars != nil {
		fields = append(fields, "env_vars = ?")
		args = append(args, *req.EnvVars)
	}
	if req.LogLevel != nil {
		fields = append(fields, "log_level = ?")
		args = append(args, *req.LogLevel)
	}
	if req.Enabled != nil {
		fields = append(fields, "enabled = ?")
		args = append(args, *req.Enabled)
	}
	if req.ImageID != nil {
		fields = append(fields, "image_id = ?")
		args = append(args, *req.ImageID)
	}

	if len(fields) == 0 {
		return GetJobByID(db, id)
	}

	fields = append(fields, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE jobs SET %s WHERE id = ?", joinFields(fields))
	if _, err := db.Exec(query, args...); err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}
	return GetJobByID(db, id)
}

func DeleteJob(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM jobs WHERE id = ?", id)
	return err
}

func ListEnabledJobs(db *sql.DB) ([]*model.Job, error) {
	return ListJobs(db, true, 0, "admin")
}

func joinFields(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ", "
		}
		result += f
	}
	return result
}
