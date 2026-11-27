package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"
)

const (
	searchMinQueryLen = 2
	searchMaxQueryLen = 64
	searchRateLimit   = 30
	searchRateWindow  = time.Minute
)

type searchRateLimiter struct {
	mu      sync.Mutex
	buckets map[int64][]time.Time
}

func newSearchRateLimiter() *searchRateLimiter {
	return &searchRateLimiter{buckets: make(map[int64][]time.Time)}
}

func (l *searchRateLimiter) allow(userID int64, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-searchRateWindow)
	times := l.buckets[userID]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= searchRateLimit {
		l.buckets[userID] = kept
		wait := kept[0].Add(searchRateWindow).Sub(now)
		if wait < time.Second {
			wait = time.Second
		}
		return false, wait
	}
	l.buckets[userID] = append(kept, now)
	return true, 0
}

func nonWildcardLen(q string) int {
	n := 0
	for _, r := range q {
		if r != '%' && r != '_' && r != '\\' {
			n++
		}
	}
	return n
}

func SearchUsers(database *sql.DB) http.HandlerFunc {
	limiter := newSearchRateLimiter()
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if ok, wait := limiter.allow(userID, time.Now()); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			jsonError(w, http.StatusTooManyRequests, "too many search requests, try again later")
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if utf8.RuneCountInString(q) > searchMaxQueryLen {
			jsonError(w, http.StatusBadRequest, "query too long")
			return
		}
		if nonWildcardLen(q) < searchMinQueryLen {
			jsonError(w, http.StatusBadRequest, "query must contain at least 2 non-wildcard characters")
			return
		}

		users, err := db.SearchUsers(database, q)
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
