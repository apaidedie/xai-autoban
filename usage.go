package main

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

func handleUsage(raw []byte) ([]byte, error) {
	var record pluginapi.UsageRecord
	if len(raw) == 0 || json.Unmarshal(raw, &record) != nil {
		return okEnvelope(map[string]any{})
	}
	if !strings.EqualFold(record.Provider, providerXAI) || !record.Failed {
		return okEnvelope(map[string]any{})
	}
	if record.AuthID == "" {
		return okEnvelope(map[string]any{})
	}
	now := time.Now()
	entry, ok := engine.classifyFailure(record.Failure.StatusCode, record.ResponseHeaders, now)
	if !ok {
		return okEnvelope(map[string]any{})
	}
	if err := engine.applyFailure(record.AuthID, "usage", entry, false); err != nil {
		slog.Warn("xai-autoban: apply failure action failed", "auth_id", record.AuthID, "error", err)
	} else {
		slog.Warn("xai-autoban: excluded credential", "auth_id", record.AuthID, "status", entry.StatusCode, "reason", entry.Reason, "reset_at", entry.ResetAt.Format(time.RFC3339), "action", entry.Action)
	}
	return okEnvelope(map[string]any{})
}
