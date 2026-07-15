package mgmt

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
	"xai-autoban/internal/action"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/config"
	"xai-autoban/internal/creds"
	"xai-autoban/internal/host"
	"xai-autoban/internal/persist"
	"xai-autoban/internal/probe"
	"xai-autoban/internal/ui"
	"xai-autoban/internal/xai"
)

type Handler struct {
	Name    string
	Version string
	Cfg     func() config.PluginConfig
	SetCfg  func(config.PluginConfig)
	Bans    *ban.State
	Audit   *audit.Log
	Engine  *action.Engine
	Probe   *probe.Service
	Persist *persist.Persister
	Host    host.Client
}

func (h *Handler) Registration() pluginapi.ManagementRegistrationResponse {
	return pluginapi.ManagementRegistrationResponse{
		Routes: []pluginapi.ManagementRoute{
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/bans", Description: "列出隔离账本与凭证状态。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/audit", Description: "最近审计事件。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/settings", Description: "读取运行时配置。"},
			{Method: http.MethodPut, Path: ("/plugins/" + h.Name) + "/settings", Description: "更新运行时配置（巡检策略等）。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/settings", Description: "更新运行时配置（巡检策略等）。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/unban", Description: "取消隔离一条凭证。Body: {\"auth_id\":\"...\"}."},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/unban-all", Description: "取消全部隔离。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/import", Description: "导入隔离快照/备份 JSON。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/backup", Description: "导出隔离账本+配置备份（无密钥）。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/probe", Description: "立即巡检。默认异步；body {wait:true} 同步。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/probe/status", Description: "巡检任务进度 done/total。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/apply-action", Description: "手动 隔离|禁用|删除|启用|重授权。Body: {\"auth_id\",\"action\",\"force?\"}."},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/reauth", Description: "用 refresh_token 刷新 access_token。Body: {\"auth_id\"}."},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/bans-recheck-429", Description: "复检当前 429 隔离；恢复则取消隔离，否则刷新窗口。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/recheck-selected", Description: "并发复检所选（含已禁用）。Body: {\"auth_ids\":[],\"reenable_on_ok?\":true}."},
		},
		Resources: []pluginapi.ResourceRoute{
			{Path: "/status", Menu: "xAI Autoban", Description: "xAI 隔离/禁用运维台。"},
			// GET = 列表；POST = 运维写操作（与读列表同路径，CPA 一定能路由到）
			{Path: "/data", Description: "GET 只读列表；POST {\"op\":...} 运维写操作。"},
		},
	}
}

func resourcePathMatch(path, name, suffix string) bool {
	path = strings.TrimRight(strings.TrimSpace(path), "/")
	suffix = strings.TrimPrefix(suffix, "/")
	if path == "" {
		return false
	}
	// Relative forms CPA may pass after stripping host prefix.
	if path == "/"+suffix || path == suffix {
		return true
	}
	candidates := []string{
		"/v0/resource/plugins/" + name + "/" + suffix,
		"/resource/plugins/" + name + "/" + suffix,
		"/plugins/" + name + "/" + suffix,
		"/" + name + "/" + suffix,
	}
	for _, c := range candidates {
		if path == c || strings.HasSuffix(path, c) {
			return true
		}
	}
	return strings.HasSuffix(path, "/"+name+"/"+suffix)
}

func (h *Handler) Handle(req pluginapi.ManagementRequest) pluginapi.ManagementResponse {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimRight(req.Path, "/")
	// POST /data is the ops write channel (same resource that GET list uses → no 404).
	if (method == http.MethodPost || method == http.MethodPut) && resourcePathMatch(path, h.Name, "data") {
		// Distinguish list POST-with-op from accidental empty POST.
		if len(req.Body) > 0 && (bytesContainsOp(req.Body) || req.Query.Get("op") != "") {
			return h.handleResourceAPI(req)
		}
	}
	if method == http.MethodGet && (resourcePathMatch(path, h.Name, "probe/status") || resourcePathMatch(path, h.Name, "probe-status")) {
		st := h.Probe.JobStatus()
		return jsonResponse(http.StatusOK, map[string]any{
			"ok": true, "running": st.Running, "job_id": st.JobID,
			"done": st.Done, "total": st.Total, "result": st.Result, "error": st.Error,
		})
	}
	switch {
	case method == http.MethodGet && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/bans"):
		return jsonResponse(http.StatusOK, h.CurrentStatusPaged(req.Query))
	case method == http.MethodGet && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/audit"):
		return jsonResponse(http.StatusOK, map[string]any{"events": h.Audit.List()})
	case method == http.MethodGet && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/settings"):
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "settings": h.Cfg().PublicView()})
	case (method == http.MethodPut || method == http.MethodPost) && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/settings"):
		return h.updateSettings(req.Body)
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/unban"):
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
		removed := h.Bans.Clear(strings.TrimSpace(body.AuthID))
		h.Audit.Add("manual", body.AuthID, "unban", "ok", "", 0)
		h.Persist.ScheduleSave()
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "removed": removed, "status": h.CurrentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/unban-all"):
		n := h.Bans.ClearAll()
		h.Audit.Add("manual", "", "unban_all", "ok", "", 0)
		h.Persist.ScheduleSave()
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "removed": n, "status": h.CurrentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/import"):
		return h.ImportSnapshot(req.Body)
	case method == http.MethodGet && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/backup"):
		return jsonResponse(http.StatusOK, h.BuildBackup())
	case method == http.MethodGet && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/probe/status"):
		st := h.Probe.JobStatus()
		return jsonResponse(http.StatusOK, map[string]any{
			"ok": true, "running": st.Running, "job_id": st.JobID,
			"done": st.Done, "total": st.Total, "result": st.Result, "error": st.Error,
		})
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/probe"):
		var body struct {
			Force bool `json:"force"`
			Wait  bool `json:"wait"`
		}
		_ = json.Unmarshal(req.Body, &body)
		if body.Wait {
			res, err := h.Probe.RunOnce(body.Force)
			if err != nil {
				return jsonResponse(http.StatusBadGateway, map[string]any{"error": err.Error(), "result": res})
			}
			h.Audit.Add("manual", "", "probe", "ok", "", 0)
			return jsonResponse(http.StatusOK, map[string]any{"ok": true, "result": res, "status": h.CurrentStatus()})
		}
		id, err := h.Probe.StartJob(body.Force, "manual")
		if err != nil {
			st := h.Probe.JobStatus()
			return jsonResponse(http.StatusConflict, map[string]any{
				"ok": false, "error": err.Error(), "job_id": st.JobID,
				"running": st.Running, "done": st.Done, "total": st.Total,
			})
		}
		h.Audit.Add("manual", "", "probe", "accepted", fmt.Sprintf("job %d", id), 0)
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "accepted": true, "job_id": id})
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/apply-action"):
		var body struct {
			AuthID string `json:"auth_id"`
			Action string `json:"action"`
			Force  bool   `json:"force"`
		}
		_ = json.Unmarshal(req.Body, &body)
		if strings.TrimSpace(body.AuthID) == "" || strings.TrimSpace(body.Action) == "" {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": "missing_auth_id_or_action"})
		}
		// Reuse ops-console Bearer so disable hits CPA Management API without separate plugin key config.
		if k := extractBearer(req.Headers); k != "" {
			h.Engine.SetRequestManagementKey(k)
			defer h.Engine.ClearRequestManagementKey()
		}
		now := time.Now()
		act := strings.ToLower(strings.TrimSpace(body.Action))
		entry := ban.Entry{
			StatusCode: 403,
			Reason:     "manual",
			BannedAt:   now,
			ResetAt:    now.Add(h.Cfg().DurationForStatus(403)),
			Action:     act,
			Source:     "manual",
		}
		// reenable does not create a ban ledger entry; still uses applyAction path.
		if act == action.SuccessReenable {
			entry.StatusCode = 0
			entry.Reason = "manual_reenable"
			entry.ResetAt = time.Time{}
		}
		if act == action.Disable {
			entry.Reason = "manual_disable"
		}
		if act == action.Ban {
			entry.Reason = "manual_ban"
		}
		if act == action.Reauth {
			entry.StatusCode = http.StatusUnauthorized
			entry.Reason = "manual_reauth"
			entry.Classification = "reauth"
		}
		if err := h.Engine.ApplyAction(body.AuthID, act, "manual", entry, body.Force); err != nil {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "status": h.CurrentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/reauth"):
		var body struct {
			AuthID string `json:"auth_id"`
			Force  bool   `json:"force"`
		}
		_ = json.Unmarshal(req.Body, &body)
		if strings.TrimSpace(body.AuthID) == "" {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": "missing_auth_id"})
		}
		now := time.Now()
		entry := ban.Entry{
			StatusCode:     http.StatusUnauthorized,
			Reason:         "manual_reauth",
			Classification: "reauth",
			BannedAt:       now,
			ResetAt:        now.Add(h.Cfg().DurationForStatus(http.StatusUnauthorized)),
			Action:         action.Reauth,
			Source:         "manual",
		}
		if err := h.Engine.ApplyAction(body.AuthID, action.Reauth, "manual", entry, body.Force); err != nil {
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "status": h.CurrentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/bans-recheck-429"):
		var body struct {
			Force bool `json:"force"`
		}
		_ = json.Unmarshal(req.Body, &body)
		res, err := h.Probe.Recheck429(body.Force)
		if err != nil {
			return jsonResponse(http.StatusBadGateway, map[string]any{"error": err.Error(), "result": res})
		}
		h.Audit.Add("manual", "", "recheck429", "ok", fmt.Sprintf("checked=%d unbanned=%d relocked=%d", res.Checked, res.Unbanned, res.Relocked), 0)
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "result": res, "status": h.CurrentStatus()})
	case method == http.MethodPost && strings.HasSuffix(path, ("/plugins/"+h.Name)+"/recheck-selected"):
		var body struct {
			AuthIDs      []string `json:"auth_ids"`
			ReenableOnOK *bool    `json:"reenable_on_ok"`
		}
		_ = json.Unmarshal(req.Body, &body)
		reenable := true
		if body.ReenableOnOK != nil {
			reenable = *body.ReenableOnOK
		}
		if k := extractBearer(req.Headers); k != "" {
			h.Engine.SetRequestManagementKey(k)
			defer h.Engine.ClearRequestManagementKey()
		}
		res, err := h.Probe.RecheckSelected(body.AuthIDs, reenable)
		if err != nil {
			code := http.StatusBadGateway
			if strings.Contains(err.Error(), "missing_auth_ids") {
				code = http.StatusBadRequest
			}
			return jsonResponse(code, map[string]any{"error": err.Error(), "result": res})
		}
		h.Audit.Add("manual", "", "recheck_selected", "ok", fmt.Sprintf("checked=%d ok=%d failed=%d reenabled=%d", res.Checked, res.OK, res.Failed, res.Reenabled), 0)
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "result": res, "status": h.CurrentStatus()})
	case method == http.MethodGet && resourcePathMatch(path, h.Name, "data"):
		return jsonResponse(http.StatusOK, h.CurrentStatusPaged(req.Query))
	case method == http.MethodGet && resourcePathMatch(path, h.Name, "status"):
		return pluginapi.ManagementResponse{
			StatusCode: http.StatusOK,
			Headers:    http.Header{"Content-Type": {"text/html; charset=utf-8"}},
			Body:       []byte(ui.StatusPage(h.Name, h.Version)),
		}
	default:
		return jsonResponse(http.StatusNotFound, map[string]any{"error": "not_found"})
	}
}

func bytesContainsOp(raw []byte) bool {
	// cheap check before full parse
	s := strings.ToLower(string(raw))
	return strings.Contains(s, `"op"`) || strings.Contains(s, `"op":`)
}

// handleResourceAPI dispatches ops-console mutations without requiring browser admin key.
// Body: {"op":"unban|unban_all|settings|probe|apply|reauth|recheck429|recheck_selected|import|backup", ...}
func (h *Handler) handleResourceAPI(req pluginapi.ManagementRequest) pluginapi.ManagementResponse {
	var body map[string]any
	_ = json.Unmarshal(req.Body, &body)
	if body == nil {
		body = map[string]any{}
	}
	op, _ := body["op"].(string)
	if op == "" {
		op = req.Query.Get("op")
	}
	op = strings.ToLower(strings.TrimSpace(op))
	delete(body, "op")
	raw, _ := json.Marshal(body)
	base := "/v0/management/plugins/" + h.Name
	switch op {
	case "settings", "update_settings":
		return h.updateSettings(raw)
	case "unban":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/unban", Body: raw, Headers: req.Headers, Query: req.Query})
	case "unban_all", "unban-all":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/unban-all", Body: raw, Headers: req.Headers, Query: req.Query})
	case "probe":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/probe", Body: raw, Headers: req.Headers, Query: req.Query})
	case "apply", "apply_action", "apply-action":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/apply-action", Body: raw, Headers: req.Headers, Query: req.Query})
	case "reauth":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/reauth", Body: raw, Headers: req.Headers, Query: req.Query})
	case "recheck429", "bans-recheck-429", "recheck_429":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/bans-recheck-429", Body: raw, Headers: req.Headers, Query: req.Query})
	case "recheck_selected", "recheck-selected":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/recheck-selected", Body: raw, Headers: req.Headers, Query: req.Query})
	case "import":
		// prefer nested snapshot if present
		if snap, ok := body["snapshot"]; ok {
			b, _ := json.Marshal(snap)
			return h.ImportSnapshot(b)
		}
		return h.ImportSnapshot(raw)
	case "backup":
		return jsonResponse(http.StatusOK, h.BuildBackup())
	case "probe_status", "probe-status":
		st := h.Probe.JobStatus()
		return jsonResponse(http.StatusOK, map[string]any{
			"ok": true, "running": st.Running, "job_id": st.JobID,
			"done": st.Done, "total": st.Total, "result": st.Result, "error": st.Error,
		})
	default:
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "unknown_op", "op": op})
	}
}

func (h *Handler) updateSettings(raw []byte) pluginapi.ManagementResponse {
	var patch map[string]any
	if err := json.Unmarshal(raw, &patch); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
	}
	// allow nested {"settings":{...}}
	if nested, ok := patch["settings"].(map[string]any); ok {
		patch = nested
	}
	cfg, warnings := config.MergePatch(h.Cfg(), patch)
	h.SetCfg(cfg)
	h.Audit.Add("manual", "", "settings", "ok", "runtime settings updated", 0)
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":       true,
		"settings": cfg.PublicView(),
		"warnings": warnings,
		"note":     "已写入运行时；日常请用运维台改策略。重启后可能回落 yaml。插件管理仅建议改启用与管理密钥。",
	})
}

type backupSnapshot struct {
	Format        string             `json:"format"`
	FormatVersion int                `json:"format_version"`
	Plugin        string             `json:"plugin"`
	PluginVersion string             `json:"plugin_version"`
	ExportedAt    string             `json:"exported_at"`
	Count         int                `json:"count"`
	Bans          []BanInfo          `json:"bans"`
	Settings      map[string]any     `json:"settings,omitempty"`
	Counts        creds.StatusCounts `json:"counts,omitempty"`
	Probe         map[string]any     `json:"probe,omitempty"`
	Audit         []audit.Event      `json:"audit,omitempty"`
	// legacy fields so old StatusInfo JSON still unmarshals into backupSnapshot
	Version string `json:"version,omitempty"`
}

func (h *Handler) BuildBackup() backupSnapshot {
	st := h.CurrentStatus()
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
		Plugin:        h.Name,
		PluginVersion: h.Version,
		ExportedAt:    time.Now().Format(time.RFC3339),
		Count:         st.Count,
		Bans:          st.Bans,
		Settings:      st.Settings,
		Counts:        st.Counts,
		Probe:         probe,
		Audit:         events,
	}
}

func (h *Handler) ImportSnapshot(raw []byte) pluginapi.ManagementResponse {
	var snapshot backupSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "invalid_snapshot", "message": err.Error()})
	}
	// also accept nested {"status":{bans:...}} and plain StatusInfo shape
	if len(snapshot.Bans) == 0 {
		var legacy StatusInfo
		if err := json.Unmarshal(raw, &legacy); err == nil && len(legacy.Bans) > 0 {
			snapshot.Bans = legacy.Bans
			if snapshot.Settings == nil {
				snapshot.Settings = legacy.Settings
			}
		}
	}
	if nested := map[string]json.RawMessage{}; json.Unmarshal(raw, &nested) == nil {
		if body, ok := nested["status"]; ok {
			var st StatusInfo
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
		h.Bans.ForceSet(item.AuthID, ban.Entry{
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
		cfg, w := config.MergePatch(h.Cfg(), snapshot.Settings)
		h.SetCfg(cfg)
		warnings = w
		settingsApplied = true
	}

	h.Persist.ScheduleSave()
	h.Audit.Add("manual", "", "import", "ok", fmt.Sprintf("imported=%d settings=%v", imported, settingsApplied), 0)
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":               true,
		"imported":         imported,
		"settings_applied": settingsApplied,
		"warnings":         warnings,
		"status":           h.CurrentStatus(),
	})
}

type StatusInfo struct {
	Plugin      string             `json:"plugin"`
	Version     string             `json:"version"`
	Count       int                `json:"count"`
	Bans        []BanInfo          `json:"bans"`
	Credentials []creds.Info       `json:"credentials,omitempty"`
	Counts      creds.StatusCounts `json:"counts"`
	Page        creds.PageMeta     `json:"page"`
	Probe       map[string]any     `json:"probe,omitempty"`
	Settings    map[string]any     `json:"settings,omitempty"`
	Audit       []audit.Event      `json:"audit,omitempty"`
}

type BanInfo struct {
	AuthID           string `json:"auth_id"`
	Email            string `json:"email,omitempty"`
	StatusCode       int    `json:"status_code"`
	Reason           string `json:"reason"`
	Classification   string `json:"classification,omitempty"`
	BannedAt         string `json:"banned_at"`
	ResetAt          string `json:"reset_at"`
	RemainingSeconds int64  `json:"remaining_seconds"`
	PendingDelete    bool   `json:"pending_delete,omitempty"`
	Action           string `json:"action,omitempty"`
	Source           string `json:"source,omitempty"`
}

func (h *Handler) CurrentStatus() StatusInfo {
	return h.CurrentStatusPaged(nil)
}

func (h *Handler) CurrentStatusPaged(query url.Values) StatusInfo {
	now := time.Now()
	snapshot := h.Bans.Snapshot(now)
	items := make([]BanInfo, 0, len(snapshot))
	for id, entry := range snapshot {
		authID := id
		if entry.AuthID != "" {
			authID = entry.AuthID
		}
		items = append(items, BanInfo{
			AuthID:           authID,
			Email:            entry.Email,
			StatusCode:       entry.StatusCode,
			Reason:           entry.Reason,
			Classification:   entry.Classification,
			BannedAt:         entry.BannedAt.Format(time.RFC3339),
			ResetAt:          entry.ResetAt.Format(time.RFC3339),
			RemainingSeconds: int64(entry.ResetAt.Sub(now).Seconds()),
			PendingDelete:    entry.PendingDelete,
			Action:           entry.Action,
			Source:           entry.Source,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ResetAt < items[j].ResetAt })

	files, _ := collectXAIAuthFiles(h.Host)
	probeLast := map[string]creds.ProbeResult{}
	for k, v := range h.Probe.LastResults() {
		probeLast[k] = creds.ProbeResult{At: v.At, OK: v.OK, Status: v.Status, Error: v.Error}
	}
	// Best-effort local token flags for first page-ish set (cap AuthGet to avoid slow UI).
	jsonByID := sampleAuthJSON(h.Host, files, 40)
	allCreds, counts := creds.BuildWithJSON(files, snapshot, probeLast, jsonByID, now)

	pq := pageQueryFromValues(query)
	pageCreds, page := creds.Page(allCreds, pq)

	st := StatusInfo{
		Plugin:      h.Name,
		Version:     h.Version,
		Count:       len(items),
		Bans:        items,
		Credentials: pageCreds,
		Counts:      counts,
		Page:        page,
		Probe:       h.Probe.Status(),
		Settings:    h.Cfg().PublicView(),
		Audit:       h.Audit.List(),
	}
	if h.Engine != nil {
		if st.Probe == nil {
			st.Probe = map[string]any{}
		}
		st.Probe["management"] = h.Engine.ManagementStatus()
	}
	return st
}

func pageQueryFromValues(q url.Values) creds.PageQuery {
	if q == nil {
		return creds.ParsePageQuery(1, creds.DefaultPageSize, "all", "")
	}
	page, _ := strconv.Atoi(strings.TrimSpace(q.Get("page")))
	pageSize, _ := strconv.Atoi(strings.TrimSpace(firstNonEmpty(q.Get("page_size"), q.Get("limit"))))
	filter := firstNonEmpty(q.Get("filter"), q.Get("status"))
	search := firstNonEmpty(q.Get("q"), q.Get("search"))
	return creds.ParsePageQuery(page, pageSize, filter, search)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func extractBearer(h http.Header) string {
	if h == nil {
		return ""
	}
	for _, key := range []string{"Authorization", "authorization", "X-Management-Key", "X-Api-Key"} {
		v := strings.TrimSpace(h.Get(key))
		if v == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "bearer ") {
			return strings.TrimSpace(v[7:])
		}
		return v
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

func collectXAIAuthFiles(cli host.Client) ([]pluginapi.HostAuthFileEntry, error) {
	if cli == nil {
		return nil, nil
	}
	files, err := cli.AuthList()
	if err != nil {
		return nil, err
	}
	out := make([]pluginapi.HostAuthFileEntry, 0)
	for _, f := range files {
		if xai.IsAuth(f) {
			out = append(out, f)
		}
	}
	return out, nil
}

// sampleAuthJSON loads up to limit credential JSON blobs for local token flags.
func sampleAuthJSON(cli host.Client, files []pluginapi.HostAuthFileEntry, limit int) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	if cli == nil || limit <= 0 {
		return out
	}
	n := 0
	for _, f := range files {
		if n >= limit {
			break
		}
		index := f.AuthIndex
		if index == "" {
			index = f.Name
		}
		if index == "" {
			continue
		}
		got, err := cli.AuthGet(index)
		if err != nil || len(got.JSON) == 0 {
			continue
		}
		key := xai.AuthKey(f)
		if key != "" {
			out[key] = got.JSON
		}
		if f.ID != "" {
			out[f.ID] = got.JSON
		}
		if f.AuthIndex != "" {
			out[f.AuthIndex] = got.JSON
		}
		n++
	}
	return out
}
