package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

func TestClassifyFailure(t *testing.T) {
	cfg := defaultConfig()
	engine.updateConfig(cfg)
	now := time.Unix(1_700_000_000, 0)
	tests := []struct {
		status int
		want   time.Duration
	}{
		{http.StatusUnauthorized, time.Duration(cfg.Ban401Seconds) * time.Second},
		{http.StatusPaymentRequired, time.Duration(cfg.Ban402Seconds) * time.Second},
		{http.StatusForbidden, time.Duration(cfg.Ban403Seconds) * time.Second},
		{http.StatusTooManyRequests, time.Duration(cfg.Ban429FallbackSeconds) * time.Second},
	}
	for _, tt := range tests {
		entry, ok := engine.classifyFailure(tt.status, nil, now)
		if !ok || entry.ResetAt.Sub(now) != tt.want {
			t.Fatalf("status %d: got %#v, ok=%v", tt.status, entry, ok)
		}
	}
	if _, ok := engine.classifyFailure(http.StatusInternalServerError, nil, now); ok {
		t.Fatal("500 must not be banned")
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	headers := http.Header{"Retry-After": {"90"}}
	entry, ok := engine.classifyFailure(http.StatusTooManyRequests, headers, now)
	if !ok || entry.ResetAt.Sub(now) != 90*time.Second {
		t.Fatalf("unexpected entry: %#v", entry)
	}
}

func TestSchedulerDelegatesAfterFilter(t *testing.T) {
	bans.clearAll()
	setConfig(defaultConfig())
	now := time.Now()
	bans.set("bad", banEntry{StatusCode: 402, ResetAt: now.Add(time.Hour)})
	req := pluginapi.SchedulerPickRequest{Candidates: []pluginapi.SchedulerAuthCandidate{
		{ID: "bad", Provider: "xai", Priority: 100},
		{ID: "good", Provider: "xai", Priority: 10},
	}}
	raw, _ := json.Marshal(req)
	responseRaw, err := handleSchedulerPick(raw)
	if err != nil {
		t.Fatal(err)
	}
	var response envelope
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatal(err)
	}
	var picked pluginapi.SchedulerPickResponse
	if err := json.Unmarshal(response.Result, &picked); err != nil {
		t.Fatal(err)
	}
	if !picked.Handled || picked.AuthID != "good" || picked.DelegateBuiltin != "" {
		t.Fatalf("unexpected pick: %#v", picked)
	}
}

func TestSchedulerSkipsBannedAliasIDs(t *testing.T) {
	bans.clearAll()
	setConfig(defaultConfig())
	now := time.Now()
	bans.set("xai-6cz4209z3r@jaliyaw.com.json", banEntry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	req := pluginapi.SchedulerPickRequest{Candidates: []pluginapi.SchedulerAuthCandidate{
		{ID: "xai-6cz4209z3r@jaliyaw.com", Provider: "xai", Priority: 100},
		{ID: "xai-good@jaliyaw.com.json", Provider: "xai", Priority: 10},
	}}
	raw, _ := json.Marshal(req)
	responseRaw, err := handleSchedulerPick(raw)
	if err != nil {
		t.Fatal(err)
	}
	var response envelope
	_ = json.Unmarshal(responseRaw, &response)
	var picked pluginapi.SchedulerPickResponse
	_ = json.Unmarshal(response.Result, &picked)
	if !picked.Handled || picked.AuthID != "xai-good@jaliyaw.com.json" {
		t.Fatalf("expected good auth after alias ban match, got %#v", picked)
	}
}

func TestSchedulerNoopWhenNothingFiltered(t *testing.T) {
	bans.clearAll()
	req := pluginapi.SchedulerPickRequest{Candidates: []pluginapi.SchedulerAuthCandidate{
		{ID: "good", Provider: "xai", Priority: 10},
	}}
	raw, _ := json.Marshal(req)
	responseRaw, err := handleSchedulerPick(raw)
	if err != nil {
		t.Fatal(err)
	}
	var response envelope
	_ = json.Unmarshal(responseRaw, &response)
	var picked pluginapi.SchedulerPickResponse
	_ = json.Unmarshal(response.Result, &picked)
	if picked.Handled {
		t.Fatalf("expected unhandled, got %#v", picked)
	}
}

func TestStatusPageUsesManagementKeyFlow(t *testing.T) {
	page := statusPage()
	for _, required := range []string{
		"/v0/management/plugins/xai-autoban",
		"Authorization",
		"Bearer",
		"/v0/resource/plugins/xai-autoban",
		"readManagementKey",
		"color-scheme:dark",
		"mgmtKeyInput",
		"保存密钥",
		"运维台",
		"编辑配置",
		"probe_on_success",
		"probe_action",
		"auto_execute",
		"只输出结果",
		"自动执行",
		"巡检历史",
		"data-filter",
		"credentials",
		"apply-action",
		"reenable",
		"健康",
		"已禁用",
		"statusChips",
		"bans-recheck-429",
		"复检 429",
		"toast",
		"progressBar",
		"setBusy",
		"/backup",
		"exportBackup",
		"importBackup",
		"overviewCards",
		"ov_healthy",
		"jumpOverview",
		"recheck-selected",
		"复检所选",
		"card-list",
		"rcard",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("page missing %q", required)
		}
	}
	if strings.Contains(page, "/action?op=unban") || strings.Contains(page, resourceActionPath()) {
		t.Fatal("public unban action must be removed")
	}
}

func resourceActionPath() string {
	return "/v0/resource/plugins/xai-autoban/action"
}

func TestPublicActionRouteRemoved(t *testing.T) {
	resp := dispatchManagement(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/action",
		Query:  map[string][]string{"op": {"unban-all"}},
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public action should 404, got %d body=%s", resp.StatusCode, string(resp.Body))
	}
}

func TestImportSnapshot(t *testing.T) {
	bans.clearAll()
	now := time.Now()
	snapshot := statusInfo{Bans: []banInfo{{
		AuthID:     "restored",
		StatusCode: 429,
		Reason:     "rate_limited",
		BannedAt:   now.Format(time.RFC3339),
		ResetAt:    now.Add(time.Hour).Format(time.RFC3339),
	}}}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	response := importSnapshot(raw)
	if response.StatusCode != http.StatusOK || currentStatus().Count != 1 {
		t.Fatalf("snapshot was not restored: response=%d status=%#v", response.StatusCode, currentStatus())
	}
}

func TestDisableViaManagementAPI(t *testing.T) {
	bans.clearAll()
	var patched []string
	var fieldPatches []string
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{
			// After management disable, host list should report Disabled=true (no AuthSave re-enable).
			{ID: "m1", AuthIndex: "3", Name: "xai-m1.json", Provider: "xai", Disabled: true},
		},
		jsonBy: map[string]json.RawMessage{
			"3": json.RawMessage(`{"access_token":"tok","disabled":false}`),
		},
		httpFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if req.Method == http.MethodPatch && strings.Contains(req.URL, "/auth-files/status") {
				patched = append(patched, string(req.Body))
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
			}
			if req.Method == http.MethodPatch && strings.Contains(req.URL, "/auth-files/fields") {
				fieldPatches = append(fieldPatches, string(req.Body))
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
			}
			if req.Method == http.MethodGet && strings.Contains(req.URL, "/auth-files") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"files":[{"id":"m1","auth_index":"3","name":"xai-m1.json","disabled":true}]}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	}
	cfg := defaultConfig()
	cfg.DisableVia = disableViaManagementAPI
	cfg.ManagementURL = "http://127.0.0.1:8317"
	cfg.ManagementKey = "test-mgmt-key"
	eng := newActionEngine(cfg, &bans, newAuditLog(20), stub, nil)
	// Tests inject host.HTTPDo; production uses direct no-proxy net/http.
	eng.mgmt.httpDo = hostHTTPDoer(stub)
	if err := eng.setDisabled("m1", true, "xai-autoban:test"); err != nil {
		t.Fatal(err)
	}
	if len(patched) < 1 {
		t.Fatalf("expected management patch, got %d", len(patched))
	}
	if !strings.Contains(patched[0], `"disabled":true`) {
		t.Fatalf("patch body=%s", patched[0])
	}
	// Must NOT AuthSave after management success (would re-enable CPA toggle).
	if len(stub.saves) != 0 {
		t.Fatalf("host.auth.save after management disable re-enables CPA toggle; saves=%d", len(stub.saves))
	}
	if len(fieldPatches) < 1 || !strings.Contains(fieldPatches[0], "xai-autoban:test") {
		t.Fatalf("expected note via fields patch, got %#v", fieldPatches)
	}
}

func TestDisableUsesRequestBearerKey(t *testing.T) {
	bans.clearAll()
	var authHeader string
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{
			{ID: "m2", AuthIndex: "4", Name: "xai-m2.json", Provider: "xai", Disabled: true},
		},
		jsonBy: map[string]json.RawMessage{
			"4": json.RawMessage(`{"access_token":"tok"}`),
		},
		httpFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if req.Method == http.MethodPatch && strings.Contains(req.URL, "/auth-files/status") {
				authHeader = req.Headers.Get("Authorization")
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
			}
			if req.Method == http.MethodPatch && strings.Contains(req.URL, "/auth-files/fields") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"files":[]}`)}, nil
		},
	}
	// no management_key in config — only request bearer
	eng := newActionEngine(defaultConfig(), &bans, newAuditLog(20), stub, nil)
	eng.mgmt.httpDo = hostHTTPDoer(stub)
	eng.setRequestManagementKey("ops-console-key")
	defer eng.clearRequestManagementKey()
	if err := eng.setDisabled("m2", true, "xai-autoban:manual_disable"); err != nil {
		t.Fatal(err)
	}
	if authHeader != "Bearer ops-console-key" {
		t.Fatalf("expected request bearer, got %q", authHeader)
	}
	if len(stub.saves) != 0 {
		t.Fatalf("must not AuthSave after management disable; saves=%d", len(stub.saves))
	}
}

func TestManagementDisableDoesNotAuthSave(t *testing.T) {
	// Regression: post-success AuthSave rewrote Auth as StatusActive → CPA toggle 启用.
	bans.clearAll()
	var statusCalls int
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{
			{ID: "rx", AuthIndex: "9", Name: "xai-rx.json", Provider: "xai", Disabled: true},
		},
		jsonBy: map[string]json.RawMessage{
			"9": json.RawMessage(`{"access_token":"t"}`),
		},
		httpFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if req.Method == http.MethodPatch && strings.Contains(req.URL, "/auth-files/status") {
				statusCalls++
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"status":"ok","disabled":true}`)}, nil
			}
			if req.Method == http.MethodPatch && strings.Contains(req.URL, "/auth-files/fields") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"status":"ok"}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	}
	cfg := defaultConfig()
	cfg.ManagementKey = "k"
	eng := newActionEngine(cfg, &bans, newAuditLog(10), stub, nil)
	eng.mgmt.httpDo = hostHTTPDoer(stub)
	if err := eng.setDisabled("rx", true, "xai-autoban:manual_disable"); err != nil {
		t.Fatal(err)
	}
	if statusCalls < 1 {
		t.Fatal("expected status patch")
	}
	if len(stub.saves) != 0 {
		t.Fatalf("AuthSave after management success is forbidden (re-enables toggle); got %d saves", len(stub.saves))
	}
}

func TestDirectManagementHTTPBypassesProxySemantics(t *testing.T) {
	// Ensure production path is wired and Proxy is nil on the shared transport.
	if directMgmtTransport == nil || directMgmtTransport.Proxy != nil {
		// Proxy == nil means "do not use proxy" (not ProxyFromEnvironment).
		// A non-nil Proxy func would be wrong for localhost management.
		if directMgmtTransport != nil && directMgmtTransport.Proxy != nil {
			t.Fatal("directMgmtTransport must not set Proxy (would reintroduce client_connect_invalid_ip)")
		}
	}
	// Proxy-style 403 must not start auth cooldown.
	err := &managementHTTPError{StatusCode: 403, Body: "You are forbidden to connect to client_connect_invalid_ip"}
	if isManagementAuthError(err) {
		t.Fatal("proxy invalid_ip 403 must not be treated as management auth failure")
	}
	if !isManagementAuthError(&managementHTTPError{StatusCode: 403, Body: `{"error":"remote management disabled"}`}) {
		t.Fatal("true management forbidden should cool down")
	}
	annotated := annotateManagementError(err)
	if annotated == nil || !strings.Contains(annotated.Error(), "直连") {
		t.Fatalf("expected proxy hint, got %v", annotated)
	}
}

func TestRecheckSelectedIncludesDisabled(t *testing.T) {
	bans.clearAll()
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{
			{ID: "dis-ok", AuthIndex: "10", Name: "xai-dis-ok", Provider: "xai", Disabled: true, Email: "a@x.ai"},
			{ID: "dis-bad", AuthIndex: "11", Name: "xai-dis-bad", Provider: "xai", Disabled: true, Email: "b@x.ai"},
		},
		jsonBy: map[string]json.RawMessage{
			"10": json.RawMessage(`{"access_token":"tok-ok","disabled":true}`),
			"11": json.RawMessage(`{"access_token":"tok-bad","disabled":true}`),
		},
		httpFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.Headers.Get("Authorization"), "tok-ok") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[]}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 401, Body: []byte(`no`)}, nil
		},
	}
	prevHost := hostImpl
	prevProbe := probeSvc
	prevEngine := engine
	hostImpl = stub
	engine = newActionEngine(defaultConfig(), &bans, audit, stub, nil)
	probeSvc = newProbeService(defaultConfig(), stub, engine)
	t.Cleanup(func() {
		hostImpl = prevHost
		probeSvc = prevProbe
		engine = prevEngine
	})

	resp := dispatchManagement(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/recheck-selected",
		Body:   []byte(`{"auth_ids":["dis-ok","dis-bad"],"reenable_on_ok":true}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	var payload map[string]any
	_ = json.Unmarshal(resp.Body, &payload)
	result, _ := payload["result"].(map[string]any)
	if int(result["checked"].(float64)) != 2 {
		t.Fatalf("result=%#v", result)
	}
	if int(result["ok"].(float64)) != 1 || int(result["failed"].(float64)) != 1 {
		t.Fatalf("ok/failed: %#v", result)
	}
	// recovered disabled should reenable (save with disabled=false)
	if len(stub.saves) < 1 {
		t.Fatal("expected reenable save for recovered disabled cred")
	}
	// failed should be banned
	if !bans.active("dis-bad", time.Now()) && !bans.active("b@x.ai", time.Now()) {
		t.Fatal("failed selected recheck should ban")
	}
}

func TestBanEmailKeyDedup(t *testing.T) {
	bans.clearAll()
	now := time.Now()
	// two auth ids, same email → one ban row under email key
	bans.set("auth-a", banEntry{
		StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour),
		Email: "user@x.ai", AuthID: "auth-a", Action: actionBan,
	})
	bans.set("auth-b", banEntry{
		StatusCode: 401, Reason: "unauthorized", BannedAt: now, ResetAt: now.Add(2 * time.Hour),
		Email: "user@x.ai", AuthID: "auth-b", Action: actionBan,
	})
	snap := bans.snapshot(now)
	if len(snap) != 1 {
		t.Fatalf("expected 1 email-keyed ban, got %d: %#v", len(snap), snap)
	}
	if _, ok := snap["user@x.ai"]; !ok {
		t.Fatalf("expected key user@x.ai, got %#v", snap)
	}
	// both auth ids should resolve active
	if !bans.active("auth-a", now) || !bans.active("auth-b", now) || !bans.active("user@x.ai", now) {
		t.Fatal("email and auth aliases should all hit the same ban")
	}
	// scheduler-style check with email attribute path
	if !bans.isBannedCandidate("auth-b", "user@x.ai", now) {
		t.Fatal("isBannedCandidate should match email")
	}
	if !bans.clear("auth-a") {
		t.Fatal("clear by auth alias should work")
	}
	if bans.active("user@x.ai", now) {
		t.Fatal("clear should remove email key")
	}
}

func TestBackupAndImportSettings(t *testing.T) {
	bans.clearAll()
	now := time.Now()
	bans.set("bk1", banEntry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(2 * time.Hour), Action: actionBan})
	bk := buildBackup()
	if bk.Format != "xai-autoban-backup" || bk.Count != 1 || len(bk.Bans) != 1 {
		t.Fatalf("backup=%+v", bk)
	}
	if bk.Settings == nil || bk.Settings["probe_action"] == nil {
		t.Fatalf("settings missing: %#v", bk.Settings)
	}
	// mutate settings in backup and re-import
	bk.Settings["probe_interval_seconds"] = 777
	raw, _ := json.Marshal(bk)
	bans.clearAll()
	resp := importSnapshot(raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	if !bans.active("bk1", time.Now()) {
		t.Fatal("expected ban restored")
	}
	if currentConfig().ProbeIntervalSeconds != 777 {
		t.Fatalf("settings not applied: %d", currentConfig().ProbeIntervalSeconds)
	}
	// restore default interval for other tests
	cfg := currentConfig()
	cfg.ProbeIntervalSeconds = defaultConfig().ProbeIntervalSeconds
	setConfig(cfg)
}

func TestConfigDefaultsAndInvalidAction(t *testing.T) {
	cfg, warnings := parseConfigYAML("action_on_401: explode\nban_401_seconds: 0\n")
	if cfg.ActionOn401 != actionBan {
		t.Fatalf("expected fallback ban, got %s", cfg.ActionOn401)
	}
	if cfg.Ban401Seconds != defaultConfig().Ban401Seconds {
		t.Fatalf("expected default ban seconds")
	}
	if len(warnings) == 0 {
		t.Fatal("expected warnings")
	}
}

func TestCooldownSkipsRepeatedBan(t *testing.T) {
	bans.clearAll()
	stub := &stubHost{}
	eng := newActionEngine(PluginConfig{ActionCooldownSeconds: 60, Ban403Seconds: 100}, &bans, newAuditLog(50), stub, nil)
	now := time.Now()
	entry := banEntry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour), Action: actionBan}
	if err := eng.applyFailure("a1", "usage", entry, false); err != nil {
		t.Fatal(err)
	}
	if !bans.active("a1", now) {
		t.Fatal("expected ban")
	}
	bans.clear("a1")
	if err := eng.applyFailure("a1", "usage", entry, false); err != nil {
		t.Fatal(err)
	}
	if bans.active("a1", now) {
		t.Fatal("cooldown should skip second ban")
	}
	if err := eng.applyFailure("a1", "usage", entry, true); err != nil {
		t.Fatal(err)
	}
	if !bans.active("a1", now) {
		t.Fatal("force should bypass cooldown")
	}
}

func TestDisableActionWritesAuth(t *testing.T) {
	bans.clearAll()
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{{ID: "auth-1", AuthIndex: "0", Name: "xai-1", Provider: "xai"}},
		jsonBy: map[string]json.RawMessage{
			"0": json.RawMessage(`{"access_token":"tok","disabled":false}`),
		},
	}
	// Without management key, host_auth JSON write is not enough to flip CPA toggle → error.
	eng := newActionEngine(defaultConfig(), &bans, newAuditLog(20), stub, nil)
	now := time.Now()
	entry := banEntry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour), Action: actionDisable}
	if err := eng.applyAction("auth-1", actionDisable, "manual", entry, true); err == nil {
		t.Fatal("expected error when disabling without management key")
	}
	if len(stub.saves) != 1 {
		t.Fatalf("expected note/json save attempt, got %d", len(stub.saves))
	}
}

func TestDisableActionViaManagementNoAuthSave(t *testing.T) {
	bans.clearAll()
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{{ID: "auth-1", AuthIndex: "0", Name: "xai-1.json", Provider: "xai", Disabled: true}},
		jsonBy: map[string]json.RawMessage{
			"0": json.RawMessage(`{"access_token":"tok","disabled":false}`),
		},
		httpFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if req.Method == http.MethodPatch {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	}
	cfg := defaultConfig()
	cfg.ManagementKey = "k"
	eng := newActionEngine(cfg, &bans, newAuditLog(20), stub, nil)
	eng.mgmt.httpDo = hostHTTPDoer(stub)
	now := time.Now()
	entry := banEntry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour), Action: actionDisable}
	if err := eng.applyAction("auth-1", actionDisable, "manual", entry, true); err != nil {
		t.Fatal(err)
	}
	if len(stub.saves) != 0 {
		t.Fatalf("management disable must not AuthSave; saves=%d", len(stub.saves))
	}
	if !bans.active("auth-1", now) {
		t.Fatal("expected ban ledger entry after disable action")
	}
}

func TestDeleteFallsBackToDisable(t *testing.T) {
	bans.clearAll()
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{{ID: "auth-2", AuthIndex: "2", Name: "xai-2", Provider: "xai"}},
		jsonBy: map[string]json.RawMessage{
			"2": json.RawMessage(`{"access_token":"tok"}`),
		},
	}
	cfg := defaultConfig()
	cfg.DeleteFallback = actionDisable
	eng := newActionEngine(cfg, &bans, newAuditLog(20), stub, nil)
	now := time.Now()
	entry := banEntry{StatusCode: 401, Reason: "unauthorized", BannedAt: now, ResetAt: now.Add(time.Hour)}
	if err := eng.applyAction("auth-2", actionDelete, "probe", entry, true); err != nil {
		t.Fatal(err)
	}
	snap := bans.snapshot(now)
	if !snap["auth-2"].PendingDelete {
		t.Fatalf("expected pending_delete: %#v", snap["auth-2"])
	}
	if len(stub.saves) != 1 {
		t.Fatal("expected disable save fallback")
	}
}

func TestExtractAccessToken(t *testing.T) {
	tok, err := extractAccessToken(json.RawMessage(`{"access_token":"abc"}`))
	if err != nil || tok != "abc" {
		t.Fatalf("got %q err=%v", tok, err)
	}
}

func TestManagementUnban(t *testing.T) {
	bans.clearAll()
	now := time.Now()
	bans.set("x", banEntry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	resp := dispatchManagement(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/unban",
		Body:   []byte(`{"auth_id":"x"}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if currentStatus().Count != 0 {
		t.Fatal("expected unban")
	}
}

func TestApplyActionReenable(t *testing.T) {
	bans.clearAll()
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{{ID: "auth-r", AuthIndex: "9", Name: "xai-r", Provider: "xai", Disabled: true}},
		jsonBy: map[string]json.RawMessage{
			"9": json.RawMessage(`{"access_token":"tok","disabled":true,"note":"xai-autoban:forbidden"}`),
		},
	}
	prev := hostImpl
	hostImpl = stub
	engine = newActionEngine(defaultConfig(), &bans, audit, stub, nil)
	t.Cleanup(func() {
		hostImpl = prev
		engine = newActionEngine(defaultConfig(), &bans, audit, hostImpl, func() { persister.scheduleSave() })
	})

	resp := dispatchManagement(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/apply-action",
		Body:   []byte(`{"auth_id":"auth-r","action":"reenable","force":true}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	if len(stub.saves) != 1 {
		t.Fatalf("expected reenable save, got %d", len(stub.saves))
	}
	var obj map[string]any
	_ = json.Unmarshal(stub.saves[0].JSON, &obj)
	if obj["disabled"] != false {
		t.Fatalf("expected disabled false: %#v", obj)
	}
	// reenable must not create a ban
	if bans.active("auth-r", time.Now()) {
		t.Fatal("reenable should not ban")
	}
}

func TestRecheck429UnbanAndRelock(t *testing.T) {
	bans.clearAll()
	now := time.Now()
	bans.set("r-ok", banEntry{StatusCode: 429, Reason: "rate_limited", BannedAt: now, ResetAt: now.Add(time.Hour), Action: actionBan, Source: "usage"})
	bans.set("r-still", banEntry{StatusCode: 429, Reason: "rate_limited", BannedAt: now, ResetAt: now.Add(time.Hour), Action: actionBan, Source: "usage"})
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{
			{ID: "r-ok", AuthIndex: "1", Name: "xai-ok", Provider: "xai"},
			{ID: "r-still", AuthIndex: "2", Name: "xai-still", Provider: "xai"},
		},
		jsonBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"tok-ok"}`),
			"2": json.RawMessage(`{"access_token":"tok-still"}`),
		},
		httpFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.Headers.Get("Authorization"), "tok-ok") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[]}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 429, Body: []byte(`rate`)}, nil
		},
	}
	prevHost := hostImpl
	prevProbe := probeSvc
	hostImpl = stub
	probeSvc = newProbeService(defaultConfig(), stub, engine)
	t.Cleanup(func() {
		hostImpl = prevHost
		probeSvc = prevProbe
	})

	resp := dispatchManagement(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/bans-recheck-429",
		Body:   []byte(`{"force":true}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	if bans.active("r-ok", time.Now()) {
		t.Fatal("recovered 429 should be unbanned")
	}
	if !bans.active("r-still", time.Now()) {
		t.Fatal("still-429 should remain banned")
	}
}

func TestCurrentStatusIncludesCredentials(t *testing.T) {
	bans.clearAll()
	stub := &stubHost{
		files: []pluginapi.HostAuthFileEntry{
			{ID: "c1", Name: "xai-c1", Provider: "xai"},
			{ID: "c2", Name: "xai-c2", Provider: "xai"},
		},
	}
	prev := hostImpl
	hostImpl = stub
	t.Cleanup(func() { hostImpl = prev })
	now := time.Now()
	bans.set("c2", banEntry{StatusCode: 402, Reason: "payment_required", BannedAt: now, ResetAt: now.Add(time.Hour)})
	st := currentStatus()
	if len(st.Credentials) != 2 {
		t.Fatalf("credentials=%d", len(st.Credentials))
	}
	if st.Counts.All != 2 || st.Counts.Banned != 1 || st.Counts.Code402 != 1 || st.Counts.Healthy != 1 {
		t.Fatalf("counts=%+v", st.Counts)
	}
}
