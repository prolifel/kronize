package db

import (
	"database/sql"
	"fmt"

	"kronize/internal/model"
)

func ListRunnerImages(database *sql.DB) ([]*model.RunnerImage, error) {
	rows, err := database.Query("SELECT id, name, image, description, created_at FROM runner_images ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list runner images: %w", err)
	}
	defer rows.Close()

	var images []*model.RunnerImage
	for rows.Next() {
		var img model.RunnerImage
		if err := rows.Scan(&img.ID, &img.Name, &img.Image, &img.Description, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan runner image: %w", err)
		}
		images = append(images, &img)
	}
	return images, nil
}

func GetRunnerImageByID(database *sql.DB, id int64) (*model.RunnerImage, error) {
	row := database.QueryRow("SELECT id, name, image, description, created_at FROM runner_images WHERE id = ?", id)
	var img model.RunnerImage
	if err := row.Scan(&img.ID, &img.Name, &img.Image, &img.Description, &img.CreatedAt); err != nil {
		return nil, fmt.Errorf("get runner image by id: %w", err)
	}
	return &img, nil
}

func CreateRunnerImage(database *sql.DB, req model.CreateRunnerImageRequest) (*model.RunnerImage, error) {
	res, err := database.Exec(
		"INSERT INTO runner_images (name, image, description) VALUES (?, ?, ?)",
		req.Name, req.Image, req.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("create runner image: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetRunnerImageByID(database, id)
}

func UpdateRunnerImage(database *sql.DB, id int64, req model.UpdateRunnerImageRequest) (*model.RunnerImage, error) {
	query := "UPDATE runner_images SET"
	args := []interface{}{}
	parts := []string{}

	if req.Name != nil {
		parts = append(parts, " name = ?")
		args = append(args, *req.Name)
	}
	if req.Image != nil {
		parts = append(parts, " image = ?")
		args = append(args, *req.Image)
	}
	if req.Description != nil {
		parts = append(parts, " description = ?")
		args = append(args, *req.Description)
	}

	if len(parts) == 0 {
		return GetRunnerImageByID(database, id)
	}

	for i, p := range parts {
		if i == 0 {
			query += p
		} else {
			query += "," + p
		}
	}
	query += " WHERE id = ?"
	args = append(args, id)

	if _, err := database.Exec(query, args...); err != nil {
		return nil, fmt.Errorf("update runner image: %w", err)
	}
	return GetRunnerImageByID(database, id)
}

func DeleteRunnerImage(database *sql.DB, id int64) error {
	_, err := database.Exec("DELETE FROM runner_images WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete runner image: %w", err)
	}
	return nil
}
