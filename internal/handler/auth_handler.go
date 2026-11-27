package handler

import (
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"
)

const (
	loginMaxFailures   = 5
	loginBaseLockout   = 30 * time.Second
	loginMaxLockout    = 15 * time.Minute
	loginFailureWindow = 5 * time.Minute
)

type loginAttemptState struct {
	failures    int
	lastFailure time.Time
	lockedUntil time.Time
}

type loginLimiter struct {
	mu      sync.Mutex
	entries map[string]*loginAttemptState
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{entries: make(map[string]*loginAttemptState)}
}

func (l *loginLimiter) keys(ip, username string) []string {
	keys := []string{"ip:" + ip}
	if username != "" {
		keys = append(keys, "user:"+strings.ToLower(username))
	}
	return keys
}

func (l *loginLimiter) allow(ip, username string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var maxWait time.Duration
	for _, k := range l.keys(ip, username) {
		st, ok := l.entries[k]
		if !ok {
			continue
		}
		if now.Before(st.lockedUntil) {
			if wait := st.lockedUntil.Sub(now); wait > maxWait {
				maxWait = wait
			}
		} else if now.Sub(st.lastFailure) > loginFailureWindow {
			delete(l.entries, k)
		}
	}
	if maxWait > 0 {
		return false, maxWait
	}
	return true, 0
}

func (l *loginLimiter) recordFailure(ip, username string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range l.keys(ip, username) {
		st, ok := l.entries[k]
		if !ok || now.Sub(st.lastFailure) > loginFailureWindow {
			st = &loginAttemptState{}
			l.entries[k] = st
		}
		st.failures++
		st.lastFailure = now
		if st.failures >= loginMaxFailures {
			shift := st.failures - loginMaxFailures
			if shift > 5 {
				shift = 5
			}
			backoff := loginBaseLockout << shift
			if backoff > loginMaxLockout {
				backoff = loginMaxLockout
			}
			st.lockedUntil = now.Add(backoff)
		}
	}
}

func (l *loginLimiter) reset(ip, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range l.keys(ip, username) {
		delete(l.entries, k)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

var commonWeakPasswords = map[string]bool{
	"admin":       true,
	"admin123":    true,
	"password":    true,
	"password123": true,
	"changeme":    true,
	"12345678":    true,
}

func Login(database *sql.DB, jwtSecret string) http.HandlerFunc {
	limiter := newLoginLimiter()
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.LoginRequest
		if err := decodeJSON(r, &req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		ip := clientIP(r)
		now := time.Now()
		if ok, wait := limiter.allow(ip, req.Username, now); !ok {
			slog.Warn("login rate limited", "username", req.Username, "ip", ip, "retry_after_sec", int(wait.Seconds())+1)
			w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			jsonError(w, http.StatusTooManyRequests, "too many login attempts, try again later")
			return
		}
		user, err := db.GetUserByUsername(database, req.Username)
		if err != nil || !auth.CheckPassword(req.Password, user.PasswordHash) {
			limiter.recordFailure(ip, req.Username, now)
			slog.Warn("failed login attempt", "username", req.Username, "ip", ip)
			jsonError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		limiter.reset(ip, req.Username)
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
		if len(req.NewPassword) < 8 {
			jsonError(w, http.StatusBadRequest, "new password must be at least 8 characters")
			return
		}
		if commonWeakPasswords[strings.ToLower(req.NewPassword)] {
			jsonError(w, http.StatusBadRequest, "new password is too weak")
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
