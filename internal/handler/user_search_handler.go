package handler

import (
	"database/sql"
	"net/http"

	"kronize/internal/db"
	"kronize/internal/model"
)

func SearchUsers(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := db.SearchUsers(database, r.URL.Query().Get("q"))
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to search users")
			return
		}
		if users == nil {
			users = []model.UserSearchResult{}
		}
		jsonResponse(w, http.StatusOK, users)
	}
}
