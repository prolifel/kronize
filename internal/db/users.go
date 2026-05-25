package db

import (
	"database/sql"
	"fmt"
	"log"

	"kronize/internal/model"
)

func CreateUser(db *sql.DB, req model.CreateUserRequest, passwordHash string) (*model.User, error) {
	role := req.Role
	if role == "" {
		role = "user"
	}
	res, err := db.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		req.Username, passwordHash, role,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetUserByID(db, id)
}

func GetUserByID(db *sql.DB, id int64) (*model.User, error) {
	row := db.QueryRow("SELECT id, username, password_hash, role, must_change_password, created_at FROM users WHERE id = ?", id)
	u := &model.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MustChangePassword, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func GetUserByUsername(db *sql.DB, username string) (*model.User, error) {
	row := db.QueryRow("SELECT id, username, password_hash, role, must_change_password, created_at FROM users WHERE username = ?", username)
	u := &model.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MustChangePassword, &u.CreatedAt); err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

func GetUsers(db *sql.DB) ([]model.User, error) {
	rows, err := db.Query("SELECT id, username, role, must_change_password, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.MustChangePassword, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func CountUsers(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func UpdateUser(db *sql.DB, id int64, req model.UpdateUserRequest) (*model.User, error) {
	if req.Username != nil || req.Role != nil {
		query := "UPDATE users SET"
		args := []interface{}{}
		if req.Username != nil {
			query += " username = ?"
			args = append(args, *req.Username)
		}
		if req.Role != nil {
			if len(args) > 0 {
				query += ","
			}
			query += " role = ?"
			args = append(args, *req.Role)
		}
		query += " WHERE id = ?"
		args = append(args, id)
		if _, err := db.Exec(query, args...); err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	}
	if req.Password != nil {
		if _, err := UpdateUserPassword(db, id, *req.Password); err != nil {
			return nil, err
		}
	}
	return GetUserByID(db, id)
}

func UpdateUserPassword(db *sql.DB, id int64, passwordHash string) (int64, error) {
	res, err := db.Exec(
		"UPDATE users SET password_hash = ?, must_change_password = 1 WHERE id = ?",
		passwordHash, id,
	)
	if err != nil {
		return 0, fmt.Errorf("update password: %w", err)
	}
	return res.RowsAffected()
}

func ClearMustChangePassword(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE users SET must_change_password = 0 WHERE id = ?", id)
	return err
}

func DeleteUser(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func SeedAdmin(db *sql.DB, passwordHash string) error {
	count, err := CountUsers(db)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.Exec(
		"INSERT INTO users (username, password_hash, role, must_change_password) VALUES (?, ?, ?, 1)",
		"admin", passwordHash, "admin",
	)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	log.Println("seeded default admin user (password: admin)")
	return nil
}
