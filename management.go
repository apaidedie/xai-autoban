package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

const managementPrefix = "/plugins/" + pluginName

func managementRegistration() pluginapi.ManagementRegistrationResponse {
	return pluginapi.ManagementRegistrationResponse{
		Routes: []pluginapi.ManagementRoute{
			{Method: http.MethodGet, Path: managementPrefix + "/bans", Description: "List xAI credentials excluded by xai-autoban."},
			{Method: http.MethodGet, Path: managementPrefix + "/audit", Description: "List recent autoban audit events."},
			{Method: http.MethodPost, Path: managementPrefix + "/unban", Description: "Release one xAI credential. Body: {\"auth_id\":\"...\"}."},
			{Method: http.MethodPost, Path: managementPrefix + "/unban-all", Description: "Release all credentials held by xai-autoban."},
			{Method: http.MethodPost, Path: managementPrefix + "/import", Description: "Restore a previously exported ban snapshot."},
			{Method: http.MethodPost, Path: managementPrefix + "/probe", Description: "Run credential probe immediately."},
			{Method: http.MethodPost, Path: managementPrefix + "/apply-action", Description: "Manually apply ban|disable|delete. Body: {\"auth_id\",\"action\",\"force?\"}."},
		},
		Resources: []pluginapi.ResourceRoute{
			{Path: "/status", Menu: "xAI Autoban", Description: "View xAI autoban status; mutations require management key."},
			{Path: "/data", Description: "Public read-only xAI autoban status data."},
		},
	}
}

func handleManagement(raw []byte) ([]byte, error) {
	var req pluginapi.ManagementRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	return okEnvelope(dispatchManagement(req))
}

func dispatchManagement(req pluginapi.ManagementRequest) pluginapi.ManagementResponse {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	resourcePrefix := "/v0/resource/plugins/" + pluginName
	path := strings.TrimRight(req.Path, "/")
	switch {
	case method == http.MethodGet && strings.HasSuffix(path, managementPrefix+"/bans"):
		return jsonResponse(http.StatusOK, currentStatus())
	case method == http.MethodGet && strings.HasSuffix(path, managementPrefix+"/audit"):
		return jsonResponse(http.StatusOK, map[string]any{"events": audit.list()})
	case method == http.MethodPost && strings.HasSuffix(path, managementPrefix+"/unban"):
		var body struct {
			AuthID string `json:"auth_id"`
			Force  bool   `json:"force"`
		}
		_ = json.Unmarshal(req.Body, &body)
		if body.AuthID == "" {
			body.AuthID = req.Query.Get("auth_id")
		}
		if strings.TrimSpace(body.AuthID) == "" {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": "missing_auth_id"})
		}
		removed := bans.clear(strings.TrimSpace(body.AuthID))
		audit.add("manual", body.AuthID, "unban", "ok", "", 0)
		persister.scheduleSave()
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "removed": removed, "status": currentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, managementPrefix+"/unban-all"):
		n := bans.clearAll()
		audit.add("manual", "", "unban_all", "ok", "", 0)
		persister.scheduleSave()
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "removed": n, "status": currentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, managementPrefix+"/import"):
		return importSnapshot(req.Body)
	case method == http.MethodPost && strings.HasSuffix(path, managementPrefix+"/probe"):
		var body struct {
			Force bool `json:"force"`
		}
		_ = json.Unmarshal(req.Body, &body)
		res, err := probeSvc.runOnce(body.Force)
		if err != nil {
			return jsonResponse(http.StatusBadGateway, map[string]any{"error": err.Error(), "result": res})
		}
		audit.add("manual", "", "probe", "ok", "", 0)
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "result": res, "status": currentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, managementPrefix+"/apply-action"):
		var body struct {
			AuthID string `json:"auth_id"`
			Action string `json:"action"`
			Force  bool   `json:"force"`
		}
		_ = json.Unmarshal(req.Body, &body)
		if strings.TrimSpace(body.AuthID) == "" || strings.TrimSpace(body.Action) == "" {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": "missing_auth_id_or_action"})
		}
		now := time.Now()
		entry := banEntry{
			StatusCode: 403,
			Reason:     "manual",
			BannedAt:   now,
			ResetAt:    now.Add(currentConfig().durationForStatus(403)),
			Action:     body.Action,
			Source:     "manual",
		}
		if err := engine.applyAction(body.AuthID, body.Action, "manual", entry, body.Force); err != nil {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "status": currentStatus()})
	case method == http.MethodGet && strings.HasSuffix(path, resourcePrefix+"/data"):
		return jsonResponse(http.StatusOK, currentStatus())
	case method == http.MethodGet && (strings.HasSuffix(path, resourcePrefix+"/status") || strings.HasSuffix(path, managementPrefix+"/status")):
		return pluginapi.ManagementResponse{
			StatusCode: http.StatusOK,
			Headers:    http.Header{"Content-Type": {"text/html; charset=utf-8"}},
			Body:       []byte(statusPage()),
		}
	default:
		return jsonResponse(http.StatusNotFound, map[string]any{"error": "not_found"})
	}
}

func importSnapshot(raw []byte) pluginapi.ManagementResponse {
	var snapshot statusInfo
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "invalid_snapshot", "message": err.Error()})
	}
	now := time.Now()
	imported := 0
	for _, item := range snapshot.Bans {
		resetAt, errReset := time.Parse(time.RFC3339, item.ResetAt)
		if errReset != nil || !resetAt.After(now) || strings.TrimSpace(item.AuthID) == "" {
			continue
		}
		bannedAt, errBanned := time.Parse(time.RFC3339, item.BannedAt)
		if errBanned != nil {
			bannedAt = now
		}
		bans.set(item.AuthID, banEntry{
			StatusCode:    item.StatusCode,
			Reason:        item.Reason,
			BannedAt:      bannedAt,
			ResetAt:       resetAt,
			PendingDelete: item.PendingDelete,
			Source:        "import",
			Action:        item.Action,
		})
		imported++
	}
	persister.scheduleSave()
	return jsonResponse(http.StatusOK, map[string]any{"ok": true, "imported": imported, "status": currentStatus()})
}

type statusInfo struct {
	Plugin  string         `json:"plugin"`
	Version string         `json:"version"`
	Count   int            `json:"count"`
	Bans    []banInfo      `json:"bans"`
	Probe   map[string]any `json:"probe,omitempty"`
	Audit   []auditEvent   `json:"audit,omitempty"`
}

type banInfo struct {
	AuthID           string `json:"auth_id"`
	StatusCode       int    `json:"status_code"`
	Reason           string `json:"reason"`
	BannedAt         string `json:"banned_at"`
	ResetAt          string `json:"reset_at"`
	RemainingSeconds int64  `json:"remaining_seconds"`
	PendingDelete    bool   `json:"pending_delete,omitempty"`
	Action           string `json:"action,omitempty"`
	Source           string `json:"source,omitempty"`
}

func currentStatus() statusInfo {
	now := time.Now()
	snapshot := bans.snapshot(now)
	items := make([]banInfo, 0, len(snapshot))
	for id, entry := range snapshot {
		items = append(items, banInfo{
			AuthID:           id,
			StatusCode:       entry.StatusCode,
			Reason:           entry.Reason,
			BannedAt:         entry.BannedAt.Format(time.RFC3339),
			ResetAt:          entry.ResetAt.Format(time.RFC3339),
			RemainingSeconds: int64(entry.ResetAt.Sub(now).Seconds()),
			PendingDelete:    entry.PendingDelete,
			Action:           entry.Action,
			Source:           entry.Source,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ResetAt < items[j].ResetAt })
	st := statusInfo{
		Plugin:  pluginName,
		Version: pluginVersion,
		Count:   len(items),
		Bans:    items,
		Probe:   probeSvc.status(),
		Audit:   audit.list(),
	}
	return st
}

func jsonResponse(status int, value any) pluginapi.ManagementResponse {
	raw, _ := json.MarshalIndent(value, "", "  ")
	return pluginapi.ManagementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Body:       raw,
	}
}
