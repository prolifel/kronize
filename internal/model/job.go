package model

import "time"

type Job struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CronExpression string    `json:"cron_expression"`
	PythonCode     string    `json:"python_code,omitempty"`
	ImageID        int64     `json:"image_id"`
	Image          string    `json:"image"`
	EnvVars        string    `json:"env_vars,omitempty"`
	LogLevel       string    `json:"log_level"`
	Enabled        bool      `json:"enabled"`
	CreatedBy      int64     `json:"created_by"`
	CreatedByUsername string  `json:"created_by_username"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateJobRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	CronExpression string `json:"cron_expression"`
	PythonCode     string `json:"python_code"`
	EnvVars        string `json:"env_vars"`
	LogLevel       string `json:"log_level"`
	ImageID        int64  `json:"image_id"`
}

type UpdateJobRequest struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	CronExpression *string `json:"cron_expression"`
	PythonCode     *string `json:"python_code"`
	EnvVars        *string `json:"env_vars"`
	LogLevel       *string `json:"log_level"`
	Enabled        *bool   `json:"enabled"`
	ImageID        *int64  `json:"image_id"`
}
