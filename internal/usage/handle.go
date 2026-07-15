package usage

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/action"
	"xai-autoban/internal/xai"
)

func Handle(raw []byte, engine *action.Engine) {
	var record pluginapi.UsageRecord
	if len(raw) == 0 || json.Unmarshal(raw, &record) != nil {
		return
	}
	if !strings.EqualFold(record.Provider, xai.Provider) || !record.Failed {
		return
	}
	if record.AuthID == "" {
		return
	}
	now := time.Now()
	entry, ok := engine.ClassifyFailureWithBody(
		record.Failure.StatusCode,
		record.ResponseHeaders,
		record.Failure.Body,
		now,
	)
	if !ok {
		return
	}
	if err := engine.ApplyFailure(record.AuthID, "usage", entry, false); err != nil {
		slog.Warn("xai-autoban: apply failure action failed", "auth_id", record.AuthID, "error", err)
	} else {
		slog.Warn("xai-autoban: excluded credential",
			"auth_id", record.AuthID,
			"status", entry.StatusCode,
			"classification", entry.Classification,
			"reason", entry.Reason,
			"reset_at", entry.ResetAt.Format(time.RFC3339),
			"action", entry.Action,
		)
	}
}
