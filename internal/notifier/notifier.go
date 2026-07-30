package notifier

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"kronize/internal/db"
	"kronize/internal/model"
)

func SendTeamsNotification(database *sql.DB, job *model.Job, execID int64, stderr string, durationMs int64) {
	webhookURL, err := db.GetSetting(database, "teams_webhook_url")
	if err != nil || webhookURL == "" {
		return
	}

	if len(stderr) > 500 {
		stderr = stderr[:500] + "..."
	}

	card := map[string]any{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": "FF0000",
		"summary":    fmt.Sprintf("Job Failed: %s", job.Name),
		"sections": []map[string]any{
			{
				"activityTitle": fmt.Sprintf("Job Failed: %s", job.Name),
				"facts": []map[string]string{
					{"name": "Job ID", "value": fmt.Sprintf("%d", job.ID)},
					{"name": "Execution ID", "value": fmt.Sprintf("%d", execID)},
					{"name": "Duration", "value": fmt.Sprintf("%dms", durationMs)},
					{"name": "Error", "value": stderr},
					{"name": "Time", "value": time.Now().Format(time.RFC3339)},
				},
				"markdown": true,
			},
		},
	}

	body, _ := json.Marshal(card)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Warn("failed to send Teams notification", "error", err)
		return
	}
	resp.Body.Close()
}
