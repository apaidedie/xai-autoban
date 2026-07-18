package mgmt

import (
	"encoding/base64"
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
	// Meta caches using_api flags across status/list calls (large fleets).
	Meta *creds.MetaCache
}

func (h *Handler) Registration() pluginapi.ManagementRegistrationResponse {
	return pluginapi.ManagementRegistrationResponse{
		Routes: []pluginapi.ManagementRoute{
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/bans", Description: "列出隔离账本与凭证状态。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/audit", Description: "最近审计事件。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/settings", Description: "读取运行时配置。"},
			{Method: http.MethodPut, Path: ("/plugins/" + h.Name) + "/settings", Description: "更新运行时配置（巡检策略等）。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/settings", Description: "更新运行时配置（巡检策略等）。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/unban", Description: "释放一条凭证（清除隔离账本）。Body: {\"auth_id\":\"...\"}."},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/unban-all", Description: "释放全部隔离。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/import", Description: "导入隔离账本快照/备份 JSON。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/backup", Description: "导出隔离账本+配置备份（无密钥）。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/probe", Description: "立即巡检。默认异步；body {wait:true} 同步。"},
			{Method: http.MethodGet, Path: ("/plugins/" + h.Name) + "/probe/status", Description: "巡检任务进度 done/total。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/apply-action", Description: "手动 隔离|释放|禁用|启用|删除|重授权。Body: {\"auth_id\",\"action\",\"force?\"}."},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/reauth", Description: "用 refresh_token 刷新 access_token。Body: {\"auth_id\"}."},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/bans-recheck-429", Description: "复检当前 429 隔离；恢复则释放，否则刷新隔离窗口。"},
			{Method: http.MethodPost, Path: ("/plugins/" + h.Name) + "/recheck-selected", Description: "并发复检所选（含已禁用）；失败按状态码动作。Body: {\"auth_ids\":[],\"reenable_on_ok?\":true}."},
		},
		Resources: []pluginapi.ResourceRoute{
			{Path: "/status", Menu: "xAI Autoban", Description: "xAI 隔离 · 禁用 · 巡检运维台。"},
			// GET = 列表；GET?op= / Header X-XAI-Autoban-Op / POST = 运维写操作
			{Path: "/data", Description: "GET 只读列表；写操作：GET?op= / Header X-XAI-Autoban-Op / POST {\"op\":...}。"},
			// 独立写通道，避免与列表 GET /data 混淆（CPAMP 对 resource GET 用已保存 CPA 密钥代理）
			{Path: "/ops", Description: "运维写操作专用。GET/POST ?op=unban&auth_id= 或 JSON {\"op\":...}。"},
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

func opHintFromRequest(req pluginapi.ManagementRequest) string {
	if op := strings.TrimSpace(req.Query.Get("op")); op != "" {
		return op
	}
	if req.Headers != nil {
		for _, k := range []string{"X-XAI-Autoban-Op", "X-Plugin-Op", "X-Op"} {
			if op := strings.TrimSpace(req.Headers.Get(k)); op != "" {
				return op
			}
		}
	}
	if len(req.Body) > 0 && bytesContainsOp(req.Body) {
		var body map[string]any
		if json.Unmarshal(req.Body, &body) == nil {
			if op, _ := body["op"].(string); strings.TrimSpace(op) != "" {
				return strings.TrimSpace(op)
			}
		}
	}
	return ""
}

func (h *Handler) Handle(req pluginapi.ManagementRequest) pluginapi.ManagementResponse {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimRight(req.Path, "/")
	// Dedicated /ops resource: always mutation channel (CPAMP GET uses saved CPA key).
	if resourcePathMatch(path, h.Name, "ops") {
		return h.handleResourceAPI(req)
	}
	// Mutations on /data: POST body {op:...}, GET ?op=..., or Header X-XAI-Autoban-Op.
	if resourcePathMatch(path, h.Name, "data") {
		hint := opHintFromRequest(req)
		if method == http.MethodPost || method == http.MethodPut {
			if hint != "" || (len(req.Body) > 0 && bytesContainsOp(req.Body)) {
				return h.handleResourceAPI(req)
			}
		}
		if method == http.MethodGet && hint != "" {
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
		view := h.Cfg().PublicView()
		if h.Persist != nil && h.Persist.Path() != "" {
			view["state_file"] = h.Persist.Path()
			view["state_file_resolved"] = h.Persist.Path()
		}
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "settings": view})
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
		if body.AuthID == "" && req.Headers != nil {
			body.AuthID = req.Headers.Get("X-XAI-Autoban-Auth-Id")
			if body.AuthID == "" {
				body.AuthID = req.Headers.Get("X-Plugin-Auth-Id")
			}
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
		// force query/body: force=true also clears a stuck "already running" lock
		if !body.Force {
			if v := strings.ToLower(strings.TrimSpace(req.Query.Get("force"))); v == "1" || v == "true" {
				body.Force = true
			}
		}
		id, err := h.Probe.StartJob(body.Force, "manual")
		if err != nil {
			st := h.Probe.JobStatus()
			// Attach to in-flight job instead of hard-fail (UI will poll progress).
			if strings.Contains(err.Error(), "already running") {
				return jsonResponse(http.StatusOK, map[string]any{
					"ok": true, "accepted": true, "already_running": true,
					"job_id": st.JobID, "running": st.Running, "done": st.Done, "total": st.Total,
				})
			}
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
		if body.AuthID == "" {
			body.AuthID = req.Query.Get("auth_id")
		}
		if body.Action == "" {
			body.Action = req.Query.Get("action")
		}
		if body.AuthID == "" && req.Headers != nil {
			body.AuthID = firstHeader(req.Headers, "X-XAI-Autoban-Auth-Id", "X-Plugin-Auth-Id")
		}
		if body.Action == "" && req.Headers != nil {
			body.Action = firstHeader(req.Headers, "X-XAI-Autoban-Action", "X-Plugin-Action")
		}
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
		if body.AuthID == "" {
			body.AuthID = req.Query.Get("auth_id")
		}
		if body.AuthID == "" {
			body.AuthID = firstHeader(req.Headers, "X-XAI-Autoban-Auth-Id", "X-Plugin-Auth-Id")
		}
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
		authIDs, reenable := parseRecheckSelected(req)
		if k := extractBearer(req.Headers); k != "" {
			h.Engine.SetRequestManagementKey(k)
			defer h.Engine.ClearRequestManagementKey()
		}
		res, err := h.Probe.RecheckSelected(authIDs, reenable)
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
			Body:       []byte(ui.StatusPage(h.Name, h.Version, resolveOpsKey(h.Cfg()))),
		}
	default:
		return jsonResponse(http.StatusNotFound, map[string]any{"error": "not_found"})
	}
}

func resolveOpsKey(cfg config.PluginConfig) string {
	return cfg.ResolveManagementKey()
}

func firstHeader(h http.Header, keys ...string) string {
	if h == nil {
		return ""
	}
	for _, k := range keys {
		if v := strings.TrimSpace(h.Get(k)); v != "" {
			return v
		}
	}
	return ""
}

// parseRecheckSelected accepts auth_ids as JSON array, JSON string, comma list, or single auth_id.
func parseRecheckSelected(req pluginapi.ManagementRequest) (ids []string, reenable bool) {
	reenable = true
	var body struct {
		AuthIDs      []string `json:"auth_ids"`
		AuthID       string   `json:"auth_id"`
		ReenableOnOK *bool    `json:"reenable_on_ok"`
	}
	_ = json.Unmarshal(req.Body, &body)
	ids = append(ids, body.AuthIDs...)
	if body.AuthID != "" {
		ids = append(ids, body.AuthID)
	}
	// Body may have auth_ids as a JSON-encoded string (from GET query merge).
	var raw map[string]any
	if json.Unmarshal(req.Body, &raw) == nil {
		if s, ok := raw["auth_ids"].(string); ok && strings.TrimSpace(s) != "" {
			ids = append(ids, splitAuthIDs(s)...)
		}
		if s, ok := raw["auth_id"].(string); ok && strings.TrimSpace(s) != "" {
			ids = append(ids, strings.TrimSpace(s))
		}
		if b, ok := raw["reenable_on_ok"].(bool); ok {
			reenable = b
		}
		if s, ok := raw["reenable_on_ok"].(string); ok {
			lv := strings.ToLower(strings.TrimSpace(s))
			reenable = lv == "1" || lv == "true" || lv == "yes"
		}
	}
	if req.Query != nil {
		if s := strings.TrimSpace(req.Query.Get("auth_ids")); s != "" {
			ids = append(ids, splitAuthIDs(s)...)
		}
		if s := strings.TrimSpace(req.Query.Get("auth_id")); s != "" {
			ids = append(ids, s)
		}
		if s := strings.TrimSpace(req.Query.Get("reenable_on_ok")); s != "" {
			lv := strings.ToLower(s)
			reenable = lv == "1" || lv == "true" || lv == "yes"
		}
	}
	if req.Headers != nil {
		if s := firstHeader(req.Headers, "X-XAI-Autoban-Auth-Ids", "X-Plugin-Auth-Ids"); s != "" {
			ids = append(ids, splitAuthIDs(s)...)
		}
		if s := firstHeader(req.Headers, "X-XAI-Autoban-Auth-Id", "X-Plugin-Auth-Id"); s != "" {
			ids = append(ids, s)
		}
	}
	if body.ReenableOnOK != nil {
		reenable = *body.ReenableOnOK
	}
	// dedupe
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, reenable
}

func splitAuthIDs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "[") {
		var arr []string
		if json.Unmarshal([]byte(s), &arr) == nil {
			return arr
		}
		var anyArr []any
		if json.Unmarshal([]byte(s), &anyArr) == nil {
			out := make([]string, 0, len(anyArr))
			for _, v := range anyArr {
				out = append(out, strings.TrimSpace(fmt.Sprint(v)))
			}
			return out
		}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func bytesContainsOp(raw []byte) bool {
	// cheap check before full parse
	s := strings.ToLower(string(raw))
	return strings.Contains(s, `"op"`) || strings.Contains(s, `"op":`)
}

// decodeOpsPayload decodes base64url / std base64 JSON blob used by GET ops under CPAMP.
func decodeOpsPayload(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty payload")
	}
	// raw URL encoding (no padding)
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// URL encoding with padding
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// std base64
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// try adding padding
	if m := len(s) % 4; m != 0 {
		s2 := s + strings.Repeat("=", 4-m)
		if b, err := base64.URLEncoding.DecodeString(s2); err == nil {
			return b, nil
		}
		if b, err := base64.StdEncoding.DecodeString(s2); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("invalid payload encoding")
}

// mergeOpsParams folds query + custom headers into body and normalizes types for GET query ops.
func mergeOpsParams(body map[string]any, req pluginapi.ManagementRequest) map[string]any {
	if body == nil {
		body = map[string]any{}
	}
	if req.Query != nil {
		for k, vs := range req.Query {
			if len(vs) == 0 {
				continue
			}
			if _, exists := body[k]; !exists {
				body[k] = vs[0]
			}
		}
	}
	// Compact GET form: ?op=settings&payload=<base64url(json)>
	if s, ok := body["payload"].(string); ok && strings.TrimSpace(s) != "" {
		if raw, err := decodeOpsPayload(s); err == nil {
			var nested map[string]any
			if json.Unmarshal(raw, &nested) == nil {
				for k, v := range nested {
					body[k] = v
				}
			}
		}
		delete(body, "payload")
	}
	if req.Headers != nil {
		headerMap := map[string]string{
			"X-XAI-Autoban-Auth-Id":  "auth_id",
			"X-XAI-Autoban-Auth-Ids": "auth_ids",
			"X-XAI-Autoban-Action":   "action",
			"X-Plugin-Auth-Id":       "auth_id",
			"X-Plugin-Auth-Ids":      "auth_ids",
			"X-Plugin-Action":        "action",
		}
		for hk, bk := range headerMap {
			if v := strings.TrimSpace(req.Headers.Get(hk)); v != "" {
				if _, exists := body[bk]; !exists || body[bk] == "" || body[bk] == nil {
					body[bk] = v
				}
			}
		}
	}
	// Query/header often stringify JSON arrays and bools.
	if s, ok := body["auth_ids"].(string); ok {
		s = strings.TrimSpace(s)
		if s != "" {
			var arr []any
			if json.Unmarshal([]byte(s), &arr) == nil {
				body["auth_ids"] = arr
			} else {
				// comma-separated fallback
				parts := strings.Split(s, ",")
				out := make([]any, 0, len(parts))
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						out = append(out, p)
					}
				}
				if len(out) > 0 {
					body["auth_ids"] = out
				}
			}
		}
	}
	// Single auth_id → auth_ids when bulk ops only look at the array.
	if _, has := body["auth_ids"]; !has {
		if id, ok := body["auth_id"].(string); ok && strings.TrimSpace(id) != "" {
			// keep auth_id; handlers that need array can still use auth_id
		}
	}
	for _, k := range []string{"force", "wait", "reenable_on_ok"} {
		switch v := body[k].(type) {
		case string:
			lv := strings.ToLower(strings.TrimSpace(v))
			body[k] = lv == "1" || lv == "true" || lv == "yes"
		}
	}
	return body
}

// handleResourceAPI dispatches ops-console mutations without requiring browser admin key.
// Body: {"op":"unban|unban_all|settings|probe|apply|reauth|recheck429|recheck_selected|import|backup", ...}
func (h *Handler) handleResourceAPI(req pluginapi.ManagementRequest) pluginapi.ManagementResponse {
	var body map[string]any
	_ = json.Unmarshal(req.Body, &body)
	body = mergeOpsParams(body, req)
	op, _ := body["op"].(string)
	if strings.TrimSpace(op) == "" {
		op = opHintFromRequest(req)
	}
	op = strings.ToLower(strings.TrimSpace(op))
	delete(body, "op")
	if op == "" {
		return jsonResponse(http.StatusBadRequest, map[string]any{
			"error":   "missing_op",
			"message": "需要 op 参数（query/header/body）",
		})
	}
	raw, _ := json.Marshal(body)
	base := "/v0/management/plugins/" + h.Name
	// Rebuild query with normalized values so recursive handlers can also read auth_id.
	q := url.Values{}
	if req.Query != nil {
		for k, vs := range req.Query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
	}
	if id, ok := body["auth_id"].(string); ok && strings.TrimSpace(id) != "" {
		q.Set("auth_id", strings.TrimSpace(id))
	}
	if act, ok := body["action"].(string); ok && strings.TrimSpace(act) != "" {
		q.Set("action", strings.TrimSpace(act))
	}
	switch op {
	case "settings", "update_settings":
		return h.updateSettings(raw)
	case "unban":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/unban", Body: raw, Headers: req.Headers, Query: q})
	case "unban_all", "unban-all":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/unban-all", Body: raw, Headers: req.Headers, Query: q})
	case "probe":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/probe", Body: raw, Headers: req.Headers, Query: q})
	case "apply", "apply_action", "apply-action":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/apply-action", Body: raw, Headers: req.Headers, Query: q})
	case "reauth":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/reauth", Body: raw, Headers: req.Headers, Query: q})
	case "recheck429", "bans-recheck-429", "recheck_429":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/bans-recheck-429", Body: raw, Headers: req.Headers, Query: q})
	case "recheck_selected", "recheck-selected":
		return h.Handle(pluginapi.ManagementRequest{Method: http.MethodPost, Path: base + "/recheck-selected", Body: raw, Headers: req.Headers, Query: q})
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
	case "list_ids", "select_ids":
		return h.listAuthIDs(body, q)
	case "bulk", "bulk_action", "bulk-action":
		return h.startBulkAction(body, req)
	case "bulk_status", "bulk-status":
		return jsonResponse(http.StatusOK, map[string]any{"ok": true, "bulk": h.Probe.BulkStatus()})
	case "export":
		return h.exportInspectList(body, q)
	default:
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "unknown_op", "op": op})
	}
}

func (h *Handler) startBulkAction(body map[string]any, req pluginapi.ManagementRequest) pluginapi.ManagementResponse {
	act := strings.ToLower(strings.TrimSpace(fmt.Sprint(body["action"])))
	if act == "" || act == "<nil>" {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "missing_action"})
	}
	ids := []string{}
	switch v := body["auth_ids"].(type) {
	case []any:
		for _, x := range v {
			s := strings.TrimSpace(fmt.Sprint(x))
			if s != "" && s != "<nil>" {
				ids = append(ids, s)
			}
		}
	case string:
		ids = splitAuthIDs(v)
	}
	if len(ids) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": "missing_auth_ids"})
	}
	if len(ids) > 800 {
		ids = ids[:800]
	}
	if k := extractBearer(req.Headers); k != "" {
		h.Engine.SetRequestManagementKey(k)
	}
	force := true
	err := h.Probe.StartBulk(act, ids, func(id string) error {
		if act == "unban" || act == action.SuccessUnban {
			if h.Bans != nil {
				_ = h.Bans.Clear(id)
			}
			return nil
		}
		now := time.Now()
		entry := ban.Entry{StatusCode: 403, Reason: "bulk", BannedAt: now, ResetAt: now.Add(h.Cfg().DurationForStatus(403)), Action: act, Source: "bulk"}
		if act == action.SuccessReenable {
			entry.StatusCode = 0
			entry.Reason = "bulk_reenable"
			entry.ResetAt = time.Time{}
		}
		if act == action.Reauth {
			entry.StatusCode = http.StatusUnauthorized
			entry.Classification = "reauth"
		}
		return h.Engine.ApplyAction(id, act, "bulk", entry, force)
	})
	if err != nil {
		h.Engine.ClearRequestManagementKey()
		return jsonResponse(http.StatusConflict, map[string]any{"error": err.Error()})
	}
	// clear request key after bulk finishes — clear in goroutine end is better; clear after short delay risk
	go func() {
		for {
			st := h.Probe.BulkStatus()
			if running, _ := st["running"].(bool); !running {
				h.Engine.ClearRequestManagementKey()
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
	return jsonResponse(http.StatusOK, map[string]any{"ok": true, "accepted": true, "total": len(ids), "action": act})
}

// exportInspectList exports reauth / pending_delete lists for cpa-auth-inspect workflows.
func (h *Handler) exportInspectList(body map[string]any, q url.Values) pluginapi.ManagementResponse {
	kind := strings.ToLower(strings.TrimSpace(firstNonEmpty(fmt.Sprint(body["kind"]), q.Get("kind"), "reauth")))
	if kind == "<nil>" {
		kind = "reauth"
	}
	now := time.Now()
	files, _ := collectXAIAuthFiles(h.Host)
	snap := h.Bans.Snapshot(now)
	type row struct {
		AuthID  string `json:"auth_id"`
		Email   string `json:"email,omitempty"`
		Name    string `json:"name,omitempty"`
		Reason  string `json:"reason,omitempty"`
		Code    int    `json:"status_code,omitempty"`
		Pending bool   `json:"pending_delete,omitempty"`
	}
	out := make([]row, 0)
	for _, f := range files {
		if !xai.IsAuth(f) {
			continue
		}
		key := xai.AuthKey(f)
		entry, ok := creds.LookupBan(snap, key, f)
		switch kind {
		case "pending_delete", "pending-delete", "delete":
			if ok && entry.PendingDelete {
				out = append(out, row{AuthID: key, Email: f.Email, Name: f.Name, Reason: entry.Reason, Code: entry.StatusCode, Pending: true})
			}
		default: // reauth
			if ok && (entry.StatusCode == 401 || entry.Classification == "reauth" ||
				strings.Contains(strings.ToLower(entry.Reason), "token") ||
				strings.Contains(strings.ToLower(entry.Reason), "unauthorized")) {
				out = append(out, row{AuthID: key, Email: f.Email, Name: f.Name, Reason: entry.Reason, Code: entry.StatusCode})
			}
		}
	}
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":      true,
		"kind":    kind,
		"count":   len(out),
		"items":   out,
		"note":    "for cpa-auth-inspect / manual reauth workflows",
		"plugin":  h.Name,
		"version": h.Version,
	})
}

// listAuthIDs returns auth_id list for current filter/search (for 全选当前筛选).
const maxListAuthIDs = 800

func (h *Handler) listAuthIDs(body map[string]any, q url.Values) pluginapi.ManagementResponse {
	filter := strings.TrimSpace(fmt.Sprint(body["filter"]))
	if filter == "" || filter == "<nil>" {
		filter = q.Get("filter")
	}
	search := strings.TrimSpace(fmt.Sprint(body["q"]))
	if search == "" || search == "<nil>" {
		search = firstNonEmpty(q.Get("q"), q.Get("search"))
	}
	limit := maxListAuthIDs
	if v, ok := body["limit"].(string); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			limit = n
		}
	}
	if v, ok := body["limit"].(float64); ok && int(v) > 0 {
		limit = int(v)
	}
	if limit > maxListAuthIDs {
		limit = maxListAuthIDs
	}
	if limit < 1 {
		limit = maxListAuthIDs
	}

	now := time.Now()
	snapshot := h.Bans.Snapshot(now)
	files, _ := collectXAIAuthFiles(h.Host)
	probeLast := map[string]creds.ProbeResult{}
	for k, v := range h.Probe.LastResults() {
		probeLast[k] = creds.ProbeResult{At: v.At, OK: v.OK, Status: v.Status, Error: v.Error}
	}
	// using_api filter: cache first, then fetch missing only.
	allCreds, _ := creds.BuildWithJSON(files, snapshot, probeLast, nil, now)
	if h.Meta != nil {
		h.Meta.Apply(allCreds)
	}
	if strings.EqualFold(strings.TrimSpace(filter), "using_api") || strings.EqualFold(strings.TrimSpace(filter), "api") {
		jsonByID := creds.SampleMissingAuthJSON(h.Host, files, 2000, h.Meta)
		if len(jsonByID) > 0 {
			allCreds, _ = creds.BuildWithJSON(files, snapshot, probeLast, jsonByID, now)
			if h.Meta != nil {
				h.Meta.Apply(allCreds)
			}
		}
	}
	matched := creds.Filter(allCreds, filter, search)
	total := len(matched)
	truncated := false
	if total > limit {
		matched = matched[:limit]
		truncated = true
	}
	ids := make([]string, 0, len(matched))
	for _, c := range matched {
		id := strings.TrimSpace(c.AuthID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":        true,
		"auth_ids":  ids,
		"count":     len(ids),
		"total":     total,
		"truncated": truncated,
		"filter":    filter,
		"q":         search,
		"limit":     limit,
	})
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
	// Drop non-ops keys that may ride along from list payloads / PublicView.
	for _, drop := range []string{"management_key", "management_key_configured", "management_key_env", "management_url", "disable_via", "state_file", "op", "payload", "filter", "q", "page", "page_size", "limit", "auth_id", "auth_ids", "action"} {
		delete(patch, drop)
	}
	patch = config.CoerceOpsPatch(patch)
	// Keep only known ops keys so stray query junk cannot count as "applied"
	clean := map[string]any{}
	allowed := map[string]struct{}{}
	for _, k := range config.OpsSettingsKeys {
		allowed[k] = struct{}{}
	}
	for k, v := range patch {
		if _, ok := allowed[k]; ok {
			clean[k] = v
		}
	}
	if len(clean) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]any{
			"error":   "empty_patch",
			"message": "未收到任何可应用的配置字段（query/payload 可能被代理丢弃）。",
		})
	}
	before := h.Cfg()
	cfg, warnings := config.MergePatch(before, clean)
	h.SetCfg(cfg)
	// Read back via Cfg() to ensure SetCfg stuck
	got := h.Cfg()
	if h.Persist != nil {
		h.Persist.SetSettings(got.OpsSettingsView())
		h.Persist.ScheduleSave()
		_ = h.Persist.SaveNow()
	}
	h.Audit.Add("manual", "", "settings", "ok", fmt.Sprintf("ops settings applied=%d", len(clean)), 0)
	statePath := ""
	if h.Persist != nil {
		statePath = h.Persist.Path()
	}
	note := "已保存到 state 文件"
	if statePath != "" {
		note = "已保存到 " + statePath
	} else {
		note = "已写入运行时；未配置 state_file（重启会丢失运维台配置）"
	}
	view := got.PublicView()
	if statePath != "" {
		view["state_file"] = statePath
		view["state_file_resolved"] = statePath
	}
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":       true,
		"settings": view,
		"applied":  len(clean),
		"keys":     mapKeys(clean),
		"warnings": warnings,
		"note":     note,
		"state_file": statePath,
	})
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
		if h.Persist != nil {
			h.Persist.SetSettings(cfg.OpsSettingsView())
		}
		warnings = w
		settingsApplied = true
	}

	if h.Persist != nil {
		h.Persist.ScheduleSave()
	}
	h.Audit.Add("manual", "", "import", "ok", fmt.Sprintf("imported=%d settings=%v", imported, settingsApplied), 0)
	return jsonResponse(http.StatusOK, map[string]any{
		"ok":               true,
		"imported":         imported,
		"settings_applied": settingsApplied,
		"warnings":         warnings,
		"status":           h.CurrentStatus(),
	})
}
