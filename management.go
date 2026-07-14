package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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
			{Method: http.MethodGet, Path: managementPrefix + "/settings", Description: "Get effective runtime settings."},
			{Method: http.MethodPut, Path: managementPrefix + "/settings", Description: "Update runtime settings (probe actions, intervals, etc.)."},
			{Method: http.MethodPost, Path: managementPrefix + "/settings", Description: "Update runtime settings (probe actions, intervals, etc.)."},
			{Method: http.MethodPost, Path: managementPrefix + "/unban", Description: "Release one xAI credential. Body: {\"auth_id\":\"...\"}."},
			{Method: http.MethodPost, Path: managementPrefix + "/unban-all", Description: "Release all credentials held by xai-autoban."},
			{Method: http.MethodPost, Path: managementPrefix + "/import", Description: "Restore a previously exported ban snapshot or full backup JSON."},
			{Method: http.MethodGet, Path: managementPrefix + "/backup", Description: "Export bans + settings backup JSON (safe, no secrets)."},
			{Method: http.MethodPost, Path: managementPrefix + "/probe", Description: "Run credential probe immediately."},
			{Method: http.MethodPost, Path: managementPrefix + "/apply-action", Description: "Manually apply ban|disable|delete|reenable. Body: {\"auth_id\",\"action\",\"force?\"}."},
			{Method: http.MethodPost, Path: managementPrefix + "/bans-recheck-429", Description: "Probe currently isolated 429 credentials; unban if recovered, else refresh ban window."},
			{Method: http.MethodPost, Path: managementPrefix + "/recheck-selected", Description: "Concurrently probe selected credentials (includes disabled). Body: {\"auth_ids\":[],\"reenable_on_ok?\":true}."},
		},
		Resources: []pluginapi.ResourceRoute{
			{Path: "/status", Menu: "xAI Autoban", Description: "View xAI autoban status; mutations require management key."},
			{Path: "/data", Description: "Public read-only status data. Query: filter,q,page,page_size."},
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
		return jsonResponse(http.StatusOK, currentStatusPaged(req.Query))
	case method == http.MethodGet && strings.HasSuffix(path, managementPrefix+"/audit"):
		return jsonResponse(http.StatusOK, map[string]any{"events": audit.list()})
	case method == http.MethodGet && strings.HasSuffix(path, managementPrefix+"/settings"):
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "settings": currentConfig().publicView()})
	case (method == http.MethodPut || method == http.MethodPost) && strings.HasSuffix(path, managementPrefix+"/settings"):
		return updateSettings(req.Body)
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
	case method == http.MethodGet && strings.HasSuffix(path, managementPrefix+"/backup"):
		return jsonResponse(http.StatusOK, buildBackup())
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
		action := strings.ToLower(strings.TrimSpace(body.Action))
		entry := banEntry{
			StatusCode: 403,
			Reason:     "manual",
			BannedAt:   now,
			ResetAt:    now.Add(currentConfig().durationForStatus(403)),
			Action:     action,
			Source:     "manual",
		}
		// reenable does not create a ban ledger entry; still uses applyAction path.
		if action == successReenable {
			entry.StatusCode = 0
			entry.Reason = "manual_reenable"
			entry.ResetAt = time.Time{}
		}
		if err := engine.applyAction(body.AuthID, action, "manual", entry, body.Force); err != nil {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "status": currentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, managementPrefix+"/bans-recheck-429"):
		var body struct {
			Force bool `json:"force"`
		}
		_ = json.Unmarshal(req.Body, &body)
		res, err := recheck429Bans(body.Force)
		if err != nil {
			return jsonResponse(http.StatusBadGateway, map[string]any{"error": err.Error(), "result": res})
		}
		audit.add("manual", "", "recheck429", "ok", fmt.Sprintf("checked=%d unbanned=%d relocked=%d", res.Checked, res.Unbanned, res.Relocked), 0)
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "result": res, "status": currentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, managementPrefix+"/recheck-selected"):
		var body struct {
			AuthIDs      []string `json:"auth_ids"`
			ReenableOnOK *bool    `json:"reenable_on_ok"`
		}
		_ = json.Unmarshal(req.Body, &body)
		reenable := true
		if body.ReenableOnOK != nil {
			reenable = *body.ReenableOnOK
		}
		res, err := recheckSelectedCredentials(body.AuthIDs, reenable)
		if err != nil {
			code := http.StatusBadGateway
			if strings.Contains(err.Error(), "missing_auth_ids") {
				code = http.StatusBadRequest
			}
			return jsonResponse(code, map[string]any{"error": err.Error(), "result": res})
		}
		audit.add("manual", "", "recheck_selected", "ok", fmt.Sprintf("checked=%d ok=%d failed=%d reenabled=%d", res.Checked, res.OK, res.Failed, res.Reenabled), 0)
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "result": res, "status": currentStatus()})
	case method == http.MethodGet && strings.HasSuffix(path, resourcePrefix+"/data"):
		return jsonResponse(http.StatusOK, currentStatusPaged(req.Query))
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

func updateSettings(raw []byte) pluginapi.ManagementResponse {
	var patch map[string]any
	if err := json.Unmarshal(raw, &patch); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
	}
	// allow nested {"settings":{...}}
	if nested, ok := patch["settings"].(map[string]any); ok {
		patch = nested
	}
	cfg, warnings := mergeConfigPatch(currentConfig(), patch)
	setConfig(cfg)
	audit.add("manual", "", "settings", "ok", "runtime settings updated", 0)
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":       true,
		"settings": cfg.publicView(),
		"warnings": warnings,
		"note":     "runtime only until CPA reloads plugins.configs.xai-autoban",
	})
}

type backupSnapshot struct {
	Format        string         `json:"format"`
	FormatVersion int            `json:"format_version"`
	Plugin        string         `json:"plugin"`
	PluginVersion string         `json:"plugin_version"`
	ExportedAt    string         `json:"exported_at"`
	Count         int            `json:"count"`
	Bans          []banInfo      `json:"bans"`
	Settings      map[string]any `json:"settings,omitempty"`
	Counts        statusCounts   `json:"counts,omitempty"`
	Probe         map[string]any `json:"probe,omitempty"`
	Audit         []auditEvent   `json:"audit,omitempty"`
	// legacy fields so old statusInfo JSON still unmarshals into backupSnapshot
	Version string `json:"version,omitempty"`
}

func buildBackup() backupSnapshot {
	st := currentStatus()
	// strip credentials/page from backup payload; keep bans+settings+meta
	probe := map[string]any{}
	if st.Probe != nil {
		// avoid huge history in backup; keep summary fields only
		for _, k := range []string{"enabled", "running", "last_run", "last_ok", "last_fail", "last_err", "mode", "interval", "auto_execute"} {
			if v, ok := st.Probe[k]; ok {
				probe[k] = v
			}
		}
	}
	// keep a short audit tail
	events := st.Audit
	if len(events) > 50 {
		events = events[:50]
	}
	return backupSnapshot{
		Format:        "xai-autoban-backup",
		FormatVersion: 1,
		Plugin:        pluginName,
		PluginVersion: pluginVersion,
		ExportedAt:    time.Now().Format(time.RFC3339),
		Count:         st.Count,
		Bans:          st.Bans,
		Settings:      st.Settings,
		Counts:        st.Counts,
		Probe:         probe,
		Audit:         events,
	}
}

func importSnapshot(raw []byte) pluginapi.ManagementResponse {
	var snapshot backupSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "invalid_snapshot", "message": err.Error()})
	}
	// also accept nested {"status":{bans:...}} and plain statusInfo shape
	if len(snapshot.Bans) == 0 {
		var legacy statusInfo
		if err := json.Unmarshal(raw, &legacy); err == nil && len(legacy.Bans) > 0 {
			snapshot.Bans = legacy.Bans
			if snapshot.Settings == nil {
				snapshot.Settings = legacy.Settings
			}
		}
	}
	if nested := map[string]json.RawMessage{}; json.Unmarshal(raw, &nested) == nil {
		if body, ok := nested["status"]; ok {
			var st statusInfo
			if json.Unmarshal(body, &st) == nil && len(st.Bans) > 0 {
				snapshot.Bans = st.Bans
			}
		}
		if body, ok := nested["backup"]; ok {
			var b backupSnapshot
			if json.Unmarshal(body, &b) == nil && len(b.Bans) > 0 {
				snapshot = b
			}
		}
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
		bans.forceSet(item.AuthID, banEntry{
			StatusCode:    item.StatusCode,
			Reason:        item.Reason,
			BannedAt:      bannedAt,
			ResetAt:       resetAt,
			PendingDelete: item.PendingDelete,
			Source:        "import",
			Action:        item.Action,
			Email:         item.Email,
			AuthID:        item.AuthID,
		})
		imported++
	}

	settingsApplied := false
	var warnings []string
	if len(snapshot.Settings) > 0 {
		cfg, w := mergeConfigPatch(currentConfig(), snapshot.Settings)
		setConfig(cfg)
		warnings = w
		settingsApplied = true
	}

	persister.scheduleSave()
	audit.add("manual", "", "import", "ok", fmt.Sprintf("imported=%d settings=%v", imported, settingsApplied), 0)
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":               true,
		"imported":         imported,
		"settings_applied": settingsApplied,
		"warnings":         warnings,
		"status":           currentStatus(),
	})
}

type statusInfo struct {
	Plugin      string           `json:"plugin"`
	Version     string           `json:"version"`
	Count       int              `json:"count"`
	Bans        []banInfo        `json:"bans"`
	Credentials []credentialInfo `json:"credentials,omitempty"`
	Counts      statusCounts     `json:"counts"`
	Page        pageMeta         `json:"page"`
	Probe       map[string]any   `json:"probe,omitempty"`
	Settings    map[string]any   `json:"settings,omitempty"`
	Audit       []auditEvent     `json:"audit,omitempty"`
}

type banInfo struct {
	AuthID           string `json:"auth_id"`
	Email            string `json:"email,omitempty"`
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
	return currentStatusPaged(nil)
}

func currentStatusPaged(query url.Values) statusInfo {
	now := time.Now()
	snapshot := bans.snapshot(now)
	items := make([]banInfo, 0, len(snapshot))
	for id, entry := range snapshot {
		authID := id
		if entry.AuthID != "" {
			authID = entry.AuthID
		}
		items = append(items, banInfo{
			AuthID:           authID,
			Email:            entry.Email,
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

	files, _ := collectXAIAuthFiles(hostImpl)
	creds, counts := buildCredentials(files, snapshot, probeSvc.lastResults(), now)

	pq := pageQueryFromValues(query)
	pageCreds, page := pageCredentials(creds, pq)

	st := statusInfo{
		Plugin:      pluginName,
		Version:     pluginVersion,
		Count:       len(items),
		Bans:        items,
		Credentials: pageCreds,
		Counts:      counts,
		Page:        page,
		Probe:       probeSvc.status(),
		Settings:    currentConfig().publicView(),
		Audit:       audit.list(),
	}
	if engine != nil && engine.mgmt != nil {
		if st.Probe == nil {
			st.Probe = map[string]any{}
		}
		st.Probe["management"] = engine.mgmt.status()
	}
	return st
}

func pageQueryFromValues(q url.Values) pageQuery {
	if q == nil {
		return parsePageQuery(1, defaultCredentialPageSize, "all", "")
	}
	page, _ := strconv.Atoi(strings.TrimSpace(q.Get("page")))
	pageSize, _ := strconv.Atoi(strings.TrimSpace(firstNonEmpty(q.Get("page_size"), q.Get("limit"))))
	filter := firstNonEmpty(q.Get("filter"), q.Get("status"))
	search := firstNonEmpty(q.Get("q"), q.Get("search"))
	return parsePageQuery(page, pageSize, filter, search)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func jsonResponse(status int, value any) pluginapi.ManagementResponse {
	raw, _ := json.MarshalIndent(value, "", "  ")
	return pluginapi.ManagementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Body:       raw,
	}
}
