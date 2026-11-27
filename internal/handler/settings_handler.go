package handler

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/model"
)

const (
	maxSettingValueLen = 2048
	maskedSettingValue = "********"
)

// allowedSettings defines the fixed registry of supported application settings.
// Secret-bearing keys (such as webhook URLs) are masked when returned by the API.
var allowedSettings = map[string]struct {
	secret bool
}{
	"teams_webhook_url": {secret: true},
}

func validateSetting(key, value string) error {
	if _, ok := allowedSettings[key]; !ok {
		return fmt.Errorf("unsupported setting key %q", key)
	}
	if len(value) > maxSettingValueLen {
		return fmt.Errorf("setting %q exceeds maximum length of %d", key, maxSettingValueLen)
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("setting %q contains invalid control characters", key)
	}
	if key == "teams_webhook_url" && value != "" {
		u, err := url.ParseRequestURI(value)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("teams_webhook_url must be a valid http or https URL")
		}
	}
	return nil
}

func sanitizeSettingsForResponse(settings []model.Setting) []model.Setting {
	out := make([]model.Setting, 0, len(settings))
	for _, s := range settings {
		spec, ok := allowedSettings[s.Key]
		if !ok {
			// Omit any legacy or unallowlisted rows (e.g. planted secret keys).
			continue
		}
		if spec.secret && s.Value != "" {
			s.Value = maskedSettingValue
		}
		out = append(out, s)
	}
	return out
}

func GetSettings(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := db.GetAllSettings(database)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to get settings")
			return
		}
		jsonResponse(w, http.StatusOK, sanitizeSettingsForResponse(settings))
	}
}

func UpdateSettings(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		role := auth.RoleFromContext(r.Context())

		var settings []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := decodeJSON(r, &settings); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		for _, s := range settings {
			if err := validateSetting(s.Key, s.Value); err != nil {
				slog.Warn("rejected settings update", "user_id", userID, "role", role, "key", s.Key, "error", err)
				jsonError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		updatedKeys := make([]string, 0, len(settings))
		for _, s := range settings {
			if err := db.SetSetting(database, s.Key, s.Value); err != nil {
				jsonError(w, http.StatusInternalServerError, "failed to update setting")
				return
			}
			updatedKeys = append(updatedKeys, s.Key)
		}
		slog.Info("settings updated", "user_id", userID, "role", role, "keys", updatedKeys)

		all, err := db.GetAllSettings(database)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to get settings")
			return
		}
		jsonResponse(w, http.StatusOK, sanitizeSettingsForResponse(all))
	}
}
