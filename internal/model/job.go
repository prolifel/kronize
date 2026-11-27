package model

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"
)

type HostMapping struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	ReadOnly  bool   `json:"read_only"`
}

var forbiddenHostPrefixes = []string{
	"/bin",
	"/boot",
	"/data",
	"/dev",
	"/etc",
	"/lib",
	"/lib32",
	"/lib64",
	"/opt",
	"/proc",
	"/root",
	"/run",
	"/sbin",
	"/srv",
	"/sys",
	"/usr",
	"/var",
}

var forbiddenContainerPrefixes = []string{
	"/bin",
	"/dev",
	"/entrypoint.sh",
	"/etc",
	"/lib",
	"/lib32",
	"/lib64",
	"/proc",
	"/run",
	"/sbin",
	"/sys",
	"/usr",
}

func hasUnsafePathChars(p string) bool {
	if p == "" || strings.ContainsAny(p, "\x00\r\n") {
		return true
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

func matchesPrefix(cleanPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+"/") {
			return true
		}
	}
	return false
}

// ParseAndValidateHostMappings parses a JSON array of HostMapping and rejects
// relative paths, traversal segments, root binds, Docker socket binds, and sensitive paths.
func ParseAndValidateHostMappings(raw string) ([]HostMapping, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" {
		return nil, nil
	}
	var mappings []HostMapping
	if err := json.Unmarshal([]byte(trimmed), &mappings); err != nil {
		return nil, fmt.Errorf("invalid host_mappings JSON: %w", err)
	}
	for i, m := range mappings {
		if hasUnsafePathChars(m.Host) || !path.IsAbs(m.Host) {
			return nil, fmt.Errorf("host_mappings[%d]: host path must be a clean absolute path", i)
		}
		if hasUnsafePathChars(m.Container) || !path.IsAbs(m.Container) {
			return nil, fmt.Errorf("host_mappings[%d]: container path must be a clean absolute path", i)
		}
		hostClean := path.Clean(m.Host)
		containerClean := path.Clean(m.Container)
		if hostClean == "/" {
			return nil, fmt.Errorf("host_mappings[%d]: mounting host root '/' is forbidden", i)
		}
		if containerClean == "/" {
			return nil, fmt.Errorf("host_mappings[%d]: mounting to container root '/' is forbidden", i)
		}
		if strings.Contains(strings.ToLower(hostClean), "docker.sock") {
			return nil, fmt.Errorf("host_mappings[%d]: mounting docker.sock is forbidden", i)
		}
		if matchesPrefix(hostClean, forbiddenHostPrefixes) {
			return nil, fmt.Errorf("host_mappings[%d]: host path %q is forbidden", i, hostClean)
		}
		if matchesPrefix(containerClean, forbiddenContainerPrefixes) {
			return nil, fmt.Errorf("host_mappings[%d]: container path %q is forbidden", i, containerClean)
		}
		mappings[i].Host = hostClean
		mappings[i].Container = containerClean
	}
	return mappings, nil
}

// ParseAndValidateEnvVars parses a JSON object of environment variables and reports whether any are set.
func ParseAndValidateEnvVars(raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "{}" {
		return nil, nil
	}
	var env map[string]string
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return nil, fmt.Errorf("invalid env_vars JSON: %w", err)
	}
	for k, v := range env {
		if k == "" || strings.ContainsAny(k, "=\x00\r\n") {
			return nil, fmt.Errorf("invalid env_vars key %q", k)
		}
		if strings.ContainsAny(v, "\x00") {
			return nil, fmt.Errorf("invalid env_vars value for %q", k)
		}
	}
	return env, nil
}

type Job struct {
	ID                int64              `json:"id"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	CronExpression    string             `json:"cron_expression"`
	PythonCode        string             `json:"python_code,omitempty"`
	ImageID           int64              `json:"image_id"`
	Image             string             `json:"image"`
	EnvVars           string             `json:"env_vars,omitempty"`
	HostMappings      string             `json:"host_mappings"`
	LogLevel          string             `json:"log_level"`
	Enabled           bool               `json:"enabled"`
	CreatedBy         int64              `json:"created_by"`
	CreatedByUsername string             `json:"created_by_username"`
	Visibility        []VisibilityTarget `json:"visibility"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type VisibilityTarget struct {
	Type     string `json:"type"` // "user" | "role"
	UserID   int64  `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role,omitempty"` // "admin"
}

type CreateJobRequest struct {
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	CronExpression string             `json:"cron_expression"`
	PythonCode     string             `json:"python_code"`
	EnvVars        string             `json:"env_vars"`
	HostMappings   string             `json:"host_mappings"`
	LogLevel       string             `json:"log_level"`
	ImageID        int64              `json:"image_id"`
	Visibility     []VisibilityTarget `json:"visibility"`
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
