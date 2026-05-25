package handler

import (
	"database/sql"
	"net/http"

	"kronize/internal/db"
)

func GetSettings(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := db.GetAllSettings(database)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to get settings")
			return
		}
		jsonResponse(w, http.StatusOK, settings)
	}
}

func UpdateSettings(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var settings []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := decodeJSON(r, &settings); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		for _, s := range settings {
			if err := db.SetSetting(database, s.Key, s.Value); err != nil {
				jsonError(w, http.StatusInternalServerError, "failed to update setting")
				return
			}
		}
		all, err := db.GetAllSettings(database)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to get settings")
			return
		}
		jsonResponse(w, http.StatusOK, all)
	}
}
