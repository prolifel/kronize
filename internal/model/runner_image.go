package model

import "time"

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
