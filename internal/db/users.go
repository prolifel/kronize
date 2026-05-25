package db

import (
	"database/sql"
	"fmt"

	"kronize/internal/model"
)

func CreateUser(db *sql.DB, req model.CreateUserRequest, passwordHash string) (*model.User, error) {
	res, err := db.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		req.Username, passwordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetUserByID(db, id)
}

func GetUserByID(db *sql.DB, id int64) (*model.User, error) {
	row := db.QueryRow("SELECT id, username, password_hash, created_at FROM users WHERE id = ?", id)
	u := &model.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func GetUserByUsername(db *sql.DB, username string) (*model.User, error) {
	row := db.QueryRow("SELECT id, username, password_hash, created_at FROM users WHERE username = ?", username)
	u := &model.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}
