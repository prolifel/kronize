package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"

	"github.com/go-chi/chi/v5"
)

func ListUsers(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := db.GetUsers(database)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to list users")
			return
		}
		jsonResponse(w, http.StatusOK, users)
	}
}

func CreateUser(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.CreateUserRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Username == "" || req.Password == "" {
			jsonError(w, http.StatusBadRequest, "username and password required")
			return
		}
		if req.Role != "admin" && req.Role != "user" {
			req.Role = "user"
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		user, err := db.CreateUser(database, req, hash)
		if err != nil {
			jsonError(w, http.StatusConflict, "username already taken")
			return
		}
		jsonResponse(w, http.StatusCreated, user)
	}
}

func UpdateUser(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid user id")
			return
		}
		var req model.UpdateUserRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Password != nil {
			hash, err := auth.HashPassword(*req.Password)
			if err != nil {
				jsonError(w, http.StatusInternalServerError, "failed to hash password")
				return
			}
			req.Password = &hash
		}
		user, err := db.UpdateUser(database, id, req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to update user")
			return
		}
		jsonResponse(w, http.StatusOK, user)
	}
}

func DeleteUser(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid user id")
			return
		}
		currentUserID := auth.UserIDFromContext(r.Context())
		if id == currentUserID {
			jsonError(w, http.StatusBadRequest, "cannot delete yourself")
			return
		}
		if err := db.DeleteUser(database, id); err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to delete user")
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"message": "user deleted"})
	}
}
