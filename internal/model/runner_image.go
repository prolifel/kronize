package model

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type RunnerImage struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateRunnerImageRequest struct {
	Name        string `json:"name"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type UpdateRunnerImageRequest struct {
	Name        *string `json:"name,omitempty"`
	Image       *string `json:"image,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ValidateRunnerImageRef validates a Docker image reference and blocks SSRF-prone
// registry targets (IP literals, loopback/private/internal hosts, and non-443 ports).
// If ALLOWED_RUNNER_REGISTRIES is set (comma-separated), the registry host must match it.
func ValidateRunnerImageRef(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("image reference is required")
	}
	if strings.ContainsAny(ref, " \t\r\n\x00?#") || strings.Contains(ref, "://") {
		return fmt.Errorf("invalid image reference format")
	}

	// Determine if the first path segment specifies a registry host.
	registry := ""
	if idx := strings.IndexByte(ref, '/'); idx != -1 {
		first := ref[:idx]
		if strings.ContainsAny(first, ".:") || strings.EqualFold(first, "localhost") {
			registry = first
		}
	}

	if registry != "" {
		host := registry
		port := ""
		if strings.HasPrefix(registry, "[") {
			// IPv6 literal like [::1] or [::1]:8080
			return fmt.Errorf("IP literal registry hosts are forbidden")
		}
		if h, p, err := net.SplitHostPort(registry); err == nil {
			host = h
			port = p
		} else if strings.Contains(registry, ":") {
			return fmt.Errorf("invalid registry host:port")
		}

		if port != "" && port != "443" {
			return fmt.Errorf("non-standard registry port %q is forbidden", port)
		}

		lowerHost := strings.ToLower(strings.TrimSuffix(host, "."))
		if lowerHost == "" {
			return fmt.Errorf("empty registry host")
		}
		if net.ParseIP(lowerHost) != nil {
			return fmt.Errorf("IP literal registry host %q is forbidden", host)
		}
		if lowerHost == "localhost" ||
			strings.HasSuffix(lowerHost, ".localhost") ||
			strings.HasSuffix(lowerHost, ".local") ||
			strings.HasSuffix(lowerHost, ".internal") {
			return fmt.Errorf("internal/local registry host %q is forbidden", host)
		}

		if allowed := strings.TrimSpace(os.Getenv("ALLOWED_RUNNER_REGISTRIES")); allowed != "" {
			matched := false
			for _, entry := range strings.Split(allowed, ",") {
				if strings.EqualFold(strings.TrimSpace(entry), lowerHost) {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("registry %q is not in ALLOWED_RUNNER_REGISTRIES", host)
			}
		}
	}

	return nil
}
