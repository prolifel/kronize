package handler

import (
	"database/sql"
	"net/http"
	"time"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"
)

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
		token, err := auth.GenerateToken(jwtSecret, user.ID, user.Username, user.Role)
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

func RequirePasswordChanged(database *sql.DB) func(http.Handler) http.Handler {
	allowedPaths := map[string]bool{
		"/api/auth/login":          true,
		"/api/auth/me":              true,
		"/api/auth/change-password": true,
		"/api/auth/logout":          true,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allowedPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			userID := auth.UserIDFromContext(r.Context())
			user, err := db.GetUserByID(database, userID)
			if err != nil {
				jsonError(w, http.StatusNotFound, "user not found")
				return
			}
			if user.MustChangePassword {
				jsonError(w, http.StatusForbidden, "password change required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ChangePassword(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.ChangePasswordRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if len(req.NewPassword) < 6 {
			jsonError(w, http.StatusBadRequest, "new password must be at least 6 characters")
			return
		}
		userID := auth.UserIDFromContext(r.Context())
		user, err := db.GetUserByID(database, userID)
		if err != nil {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		if !auth.CheckPassword(req.CurrentPassword, user.PasswordHash) {
			jsonError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		hash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		if _, err := db.UpdateUserPassword(database, userID, hash); err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to update password")
			return
		}
		if err := db.ClearMustChangePassword(database, userID); err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to update user")
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"message": "password changed"})
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
