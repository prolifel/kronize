package db

import (
	"database/sql"
	"fmt"
	"kronize/internal/model"
)

func GetSetting(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting: %w", err)
	}
	return value, nil
}

func SetSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value,
	)
	return err
}

func GetAllSettings(db *sql.DB) ([]model.Setting, error) {
	rows, err := db.Query("SELECT key, value FROM settings ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	defer rows.Close()

	var settings []model.Setting
	for rows.Next() {
		var s model.Setting
		if err := rows.Scan(&s.Key, &s.Value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings = append(settings, s)
	}
	return settings, nil
}
