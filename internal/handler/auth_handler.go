package handler

import (
	"database/sql"
	"net/http"
	"time"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"
)

func Register(database *sql.DB, jwtSecret string) http.HandlerFunc {
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
		token, err := auth.GenerateToken(jwtSecret, user.ID, user.Username)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}
		setTokenCookie(w, token)
		jsonResponse(w, http.StatusCreated, model.LoginResponse{Token: token, User: *user})
	}
}

func Login(database *sql.DB, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.LoginRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		user, err := db.GetUserByUsername(database, req.Username)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		if !auth.CheckPassword(req.Password, user.PasswordHash) {
			jsonError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		token, err := auth.GenerateToken(jwtSecret, user.ID, user.Username)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}
		setTokenCookie(w, token)
		jsonResponse(w, http.StatusOK, model.LoginResponse{Token: token, User: *user})
	}
}

func GetCurrentUser(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		user, err := db.GetUserByID(database, userID)
		if err != nil {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		jsonResponse(w, http.StatusOK, user)
	}
}

func Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
		})
		jsonResponse(w, http.StatusOK, map[string]string{"message": "logged out"})
	}
}

func setTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   72 * 3600,
	})
}
