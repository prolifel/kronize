package db

import (
	"database/sql"
	"fmt"
	"kronize/internal/model"
)

const executionCols = "id, job_id, status, stdout, stderr, exit_code, duration_ms, started_at, finished_at, source"

func CreateExecution(db *sql.DB, jobID int64, source string) (*model.Execution, error) {
	if source == "" {
		source = "scheduled"
	}
	res, err := db.Exec(
		"INSERT INTO executions (job_id, status, source) VALUES (?, 'running', ?)",
		jobID, source,
	)
	if err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetExecution(db, id)
}

func GetExecution(db *sql.DB, id int64) (*model.Execution, error) {
	e := &model.Execution{}
	err := db.QueryRow(
		`SELECT `+executionCols+`
		 FROM executions WHERE id = ?`, id,
	).Scan(&e.ID, &e.JobID, &e.Status, &e.Stdout, &e.Stderr, &e.ExitCode, &e.DurationMs, &e.StartedAt, &e.FinishedAt, &e.Source)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	return e, nil
}

func CompleteExecution(db *sql.DB, id int64, status string, stdout, stderr string, exitCode int, durationMs int64) error {
	_, err := db.Exec(
		`UPDATE executions SET status = ?, stdout = ?, stderr = ?, exit_code = ?, duration_ms = ?, finished_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		status, stdout, stderr, exitCode, durationMs, id,
	)
	return err
}

func ListExecutionsByJob(db *sql.DB, jobID int64, limit, offset int) ([]*model.Execution, error) {
	rows, err := db.Query(
		`SELECT `+executionCols+`
		 FROM executions WHERE job_id = ? ORDER BY started_at DESC LIMIT ? OFFSET ?`,
		jobID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}
	defer rows.Close()

	var execs []*model.Execution
	for rows.Next() {
		e := &model.Execution{}
		if err := rows.Scan(&e.ID, &e.JobID, &e.Status, &e.Stdout, &e.Stderr, &e.ExitCode, &e.DurationMs, &e.StartedAt, &e.FinishedAt, &e.Source); err != nil {
			return nil, fmt.Errorf("scan execution: %w", err)
		}
		execs = append(execs, e)
	}
	return execs, nil
}

func AppendExecutionOutput(db *sql.DB, id int64, stream, chunk string) error {
	if chunk == "" {
		return nil
	}
	var column string
	switch stream {
	case "stdout":
		column = "stdout"
	case "stderr":
		column = "stderr"
	default:
		return fmt.Errorf("invalid stream %q", stream)
	}
	query := fmt.Sprintf("UPDATE executions SET %s = %s || ? WHERE id = ?", column, column)
	_, err := db.Exec(query, chunk, id)
	return err
}

func DeleteOldExecutions(db *sql.DB, olderThanDays int) (int64, error) {
	res, err := db.Exec(
		"DELETE FROM executions WHERE started_at < datetime('now', ?)",
		fmt.Sprintf("-%d days", olderThanDays),
	)
	if err != nil {
		return 0, fmt.Errorf("delete old executions: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
