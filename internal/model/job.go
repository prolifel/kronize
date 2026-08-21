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
	HostMappings   string    `json:"host_mappings"`
	LogLevel       string    `json:"log_level"`
	Enabled        bool      `json:"enabled"`
	CreatedBy      int64     `json:"created_by"`
	CreatedByUsername string  `json:"created_by_username"`
	Visibility     []VisibilityTarget `json:"visibility,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type VisibilityTarget struct {
	Type     string `json:"type"` // "user" | "role"
	UserID   int64  `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role,omitempty"` // "admin"
}

type CreateJobRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	CronExpression string `json:"cron_expression"`
	PythonCode     string `json:"python_code"`
	EnvVars        string `json:"env_vars"`
	HostMappings   string `json:"host_mappings"`
	LogLevel       string `json:"log_level"`
	ImageID        int64  `json:"image_id"`
	Visibility     []VisibilityTarget `json:"visibility,omitempty"`
}

type UpdateJobRequest struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	CronExpression *string `json:"cron_expression"`
	PythonCode     *string `json:"python_code"`
	EnvVars        *string `json:"env_vars"`
	HostMappings   *string `json:"host_mappings"`
	LogLevel       *string `json:"log_level"`
	Enabled        *bool   `json:"enabled"`
	ImageID        *int64  `json:"image_id"`
}
