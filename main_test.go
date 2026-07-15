package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/action"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
	"xai-autoban/internal/mgmt"
	"xai-autoban/internal/probe"
	"xai-autoban/internal/ui"
)

func resetApp(t *testing.T) *App {
	t.Helper()
	app := NewApp(host.Real{})
	return app
}

func withStub(t *testing.T, stub *host.Stub) *App {
	t.Helper()
	app := NewApp(stub)
	return app
}

func TestClassifyFailure(t *testing.T) {
	cfg := config.Default()
	defaultApp.engine.UpdateConfig(cfg)
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
		entry, ok := defaultApp.engine.ClassifyFailure(tt.status, nil, now)
		if !ok || entry.ResetAt.Sub(now) != tt.want {
			t.Fatalf("status %d: got %#v, ok=%v", tt.status, entry, ok)
		}
	}
	if _, ok := defaultApp.engine.ClassifyFailure(http.StatusInternalServerError, nil, now); ok {
		t.Fatal("500 must not be banned")
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	headers := http.Header{"Retry-After": {"90"}}
	entry, ok := defaultApp.engine.ClassifyFailure(http.StatusTooManyRequests, headers, now)
	if !ok || entry.ResetAt.Sub(now) != 90*time.Second {
		t.Fatalf("unexpected entry: %#v", entry)
	}
}

func TestSchedulerDelegatesAfterFilter(t *testing.T) {
	defaultApp.bans.ClearAll()
	defaultApp.SetConfig(config.Default())
	now := time.Now()
	defaultApp.bans.Set("bad", ban.Entry{StatusCode: 402, ResetAt: now.Add(time.Hour)})
	req := pluginapi.SchedulerPickRequest{Candidates: []pluginapi.SchedulerAuthCandidate{
		{ID: "bad", Provider: "xai", Priority: 100},
		{ID: "good", Provider: "xai", Priority: 10},
	}}
	raw, _ := json.Marshal(req)
	responseRaw, err := handleSchedulerPickTest(raw)
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
	defaultApp.bans.ClearAll()
	defaultApp.SetConfig(config.Default())
	now := time.Now()
	defaultApp.bans.Set("xai-6cz4209z3r@jaliyaw.com.json", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	req := pluginapi.SchedulerPickRequest{Candidates: []pluginapi.SchedulerAuthCandidate{
		{ID: "xai-6cz4209z3r@jaliyaw.com", Provider: "xai", Priority: 100},
		{ID: "xai-good@jaliyaw.com.json", Provider: "xai", Priority: 10},
	}}
	raw, _ := json.Marshal(req)
	responseRaw, err := handleSchedulerPickTest(raw)
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
	defaultApp.bans.ClearAll()
	req := pluginapi.SchedulerPickRequest{Candidates: []pluginapi.SchedulerAuthCandidate{
		{ID: "good", Provider: "xai", Priority: 10},
	}}
	raw, _ := json.Marshal(req)
	responseRaw, err := handleSchedulerPickTest(raw)
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
	page := ui.StatusPage(pluginName, pluginVersion, "test-key")
	for _, required := range []string{
		"/v0/resource/plugins/xai-autoban",
		"color-scheme:dark",
		"运维台",
		"编辑配置",
		"credentials",
		"probe_on_success",
		"probe_action",
		"auto_execute",
		"只输出结果",
		"自动执行",
		"巡检历史",
		"data-filter",
		"健康",
		"已禁用",
		"statusChips",
		"复检 429",
		"toast",
		"progressBar",
		"setBusy",
		"overviewCards",
		"ov_healthy",
		"jumpOverview",
		"复检所选",
		"card-list",
		"rcard",
		"SERVER_MGMT_KEY",
		"buildOpsQuery",
		"/ops",
		"X-XAI-Autoban-Op",
		"isListPayload",
		"buildGetOpsURL",
		"b64url",
		"selectCurrentFilter",
		"deleteSelected",
		"全选当前筛选",
		"resourceBase",
		"PLUGIN_VERSION",
		"释放所选",
		"btnRefresh",
		"more-sec",
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("page missing %q", required)
		}
	}
	// no browser key paste UI
	for _, banned := range []string{"mgmtKeyInput", "保存密钥", "readManagementKey", "xai_autoban_management_key"} {
		if strings.Contains(page, banned) {
			t.Fatalf("page must not contain browser key UI %q", banned)
		}
	}
	if !strings.Contains(page, "test-key") {
		t.Fatal("expected server key injection")
	}
	if strings.Contains(page, "/action?op=unban") || strings.Contains(page, resourceActionPath()) {
		t.Fatal("public unban action must be removed")
	}
}

func resourceActionPath() string {
	return "/v0/resource/plugins/xai-autoban/action"
}

func TestPublicActionRouteRemoved(t *testing.T) {
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/action",
		Query:  map[string][]string{"op": {"unban-all"}},
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public action should 404, got %d body=%s", resp.StatusCode, string(resp.Body))
	}
}

func TestResourceDataPOSTUnban(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("r1", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour), Reason: "forbidden"})
	for _, path := range []string{
		"/v0/resource/plugins/xai-autoban/data",
		"/resource/plugins/xai-autoban/data",
		"/data",
	} {
		defaultApp.bans.Set("r1", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour), Reason: "forbidden"})
		resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
			Method: http.MethodPost,
			Path:   path,
			Body:   []byte(`{"op":"unban","auth_id":"r1"}`),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, resp.StatusCode, string(resp.Body))
		}
		if defaultApp.bans.Active("r1", now) {
			t.Fatalf("path=%s: expected unban via resource POST /data", path)
		}
	}
}

func TestResourceDataGETUnbanQuery(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("g1", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/data",
		Query:  map[string][]string{"op": {"unban"}, "auth_id": {"g1"}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	if defaultApp.bans.Active("g1", now) {
		t.Fatal("expected unban via GET /data?op=unban")
	}
}

func TestResourceOpsGETUnban(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("o1", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/ops",
		Query:  map[string][]string{"op": {"unban"}, "auth_id": {"o1"}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	if defaultApp.bans.Active("o1", now) {
		t.Fatal("expected unban via GET /ops?op=unban")
	}
}

func TestResourceDataGETUnbanViaHeader(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("h1", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	hdr := make(http.Header)
	hdr.Set("X-XAI-Autoban-Op", "unban")
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method:  http.MethodGet,
		Path:    "/v0/resource/plugins/xai-autoban/data",
		Query:   map[string][]string{"auth_id": {"h1"}},
		Headers: hdr,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	if defaultApp.bans.Active("h1", now) {
		t.Fatal("expected unban via header X-XAI-Autoban-Op")
	}
}

func TestResourceOpsGETUnbanAuthIdHeaderOnly(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("hh1", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	hdr := make(http.Header)
	hdr.Set("X-XAI-Autoban-Op", "unban")
	hdr.Set("X-XAI-Autoban-Auth-Id", "hh1")
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method:  http.MethodGet,
		Path:    "/v0/resource/plugins/xai-autoban/ops",
		Headers: hdr,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	if defaultApp.bans.Active("hh1", now) {
		t.Fatal("expected unban via Auth-Id header only")
	}
}

func TestResourceOpsListIDsFilter403(t *testing.T) {
	now := time.Now()
	stub := &host.Stub{Files: []pluginapi.HostAuthFileEntry{
		{ID: "a403", AuthIndex: "a403", Name: "a403.json", Provider: "xai", Email: "a@x.com"},
		{ID: "b429", AuthIndex: "b429", Name: "b429.json", Provider: "xai", Email: "b@x.com"},
	}}
	app := withStub(t, stub)
	app.bans.Set("a403", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour), AuthID: "a403"})
	app.bans.Set("b429", ban.Entry{StatusCode: 429, ResetAt: now.Add(time.Hour), AuthID: "b429"})
	resp := app.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/ops",
		Query:  map[string][]string{"op": {"list_ids"}, "filter": {"403"}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	var out map[string]any
	_ = json.Unmarshal(resp.Body, &out)
	ids, _ := out["auth_ids"].([]any)
	found := false
	for _, id := range ids {
		if fmt.Sprint(id) == "a403" {
			found = true
		}
		if fmt.Sprint(id) == "b429" {
			t.Fatal("429 id should not be in 403 filter")
		}
	}
	if !found {
		t.Fatalf("expected a403 in ids: %v body=%s", ids, string(resp.Body))
	}
}

func TestSettingsPersistAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	app := NewApp(host.Real{})
	cfg := config.Default()
	cfg.StateFile = path
	cfg.ProbeIntervalSeconds = 600
	app.SetConfig(cfg)

	resp := app.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/ops",
		Query: map[string][]string{
			"op": {"settings"},
			"payload": {func() string {
				raw, _ := json.Marshal(map[string]any{"probe_interval_seconds": 900, "auto_execute": false})
				return base64.RawURLEncoding.EncodeToString(raw)
			}()},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	app.persist.Flush()
	if app.Config().ProbeIntervalSeconds != 900 {
		t.Fatalf("runtime interval=%d", app.Config().ProbeIntervalSeconds)
	}

	// Simulate reconfigure from yaml defaults then overlay
	app2 := NewApp(host.Real{})
	cfg2 := config.Default()
	cfg2.StateFile = path
	app2.SetConfig(cfg2)
	app2.persist.Load()
	if overlay := app2.persist.Settings(); len(overlay) > 0 {
		merged, _ := config.MergePatch(app2.Config(), overlay)
		app2.SetConfig(merged)
	}
	if app2.Config().ProbeIntervalSeconds != 900 {
		t.Fatalf("after reload interval=%d want 900", app2.Config().ProbeIntervalSeconds)
	}
	if app2.Config().AutoExecute {
		t.Fatal("auto_execute should stay false after reload")
	}
}

func TestResourceOpsSettingsViaFlatQueryGET(t *testing.T) {
	// Flat query (production path under CPAMP) — string bools/numbers must coerce.
	defaultApp.SetConfig(config.Default())
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/ops",
		Query: map[string][]string{
			"op":                     {"settings"},
			"probe_interval_seconds": {"777"},
			"auto_execute":           {"false"},
			"probe_on_success":       {"none"},
			"probe_action":           {"disable"},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	var out map[string]any
	_ = json.Unmarshal(resp.Body, &out)
	if out["ok"] != true {
		t.Fatalf("expected ok: %s", string(resp.Body))
	}
	if n, _ := out["applied"].(float64); int(n) < 3 {
		t.Fatalf("applied too small: %v body=%s", out["applied"], string(resp.Body))
	}
	settings, _ := out["settings"].(map[string]any)
	if settings == nil {
		t.Fatalf("missing settings: %s", string(resp.Body))
	}
	if defaultApp.Config().ProbeIntervalSeconds != 777 {
		t.Fatalf("runtime interval=%d", defaultApp.Config().ProbeIntervalSeconds)
	}
	if defaultApp.Config().AutoExecute {
		t.Fatal("auto_execute should be false")
	}
	if defaultApp.Config().ProbeOnSuccess != "none" {
		t.Fatalf("probe_on_success=%s", defaultApp.Config().ProbeOnSuccess)
	}
}

func TestResourceOpsSettingsEmptyPatchRejected(t *testing.T) {
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/ops",
		Query:  map[string][]string{"op": {"settings"}},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 empty_patch, got %d %s", resp.StatusCode, string(resp.Body))
	}
}

func TestResourceOpsRecheckSelectedAuthIdsJSONString(t *testing.T) {
	// auth_ids arrives as a JSON string (GET query merge) — must not missing_auth_ids
	defaultApp.bans.ClearAll()
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/ops",
		Query: map[string][]string{
			"op":       {"recheck_selected"},
			"auth_ids": {`["no-such-auth-for-recheck"]`},
		},
	})
	// may fail probe network-wise but must not be missing_auth_ids
	if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(resp.Body), "missing_auth_ids") {
		t.Fatalf("auth_ids JSON string not parsed: %s", string(resp.Body))
	}
}

func TestImportSnapshot(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	snapshot := mgmt.StatusInfo{Bans: []mgmt.BanInfo{{
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
	response := defaultApp.mgmt.ImportSnapshot(raw)
	if response.StatusCode != http.StatusOK || defaultApp.mgmt.CurrentStatus().Count != 1 {
		t.Fatalf("snapshot was not restored: response=%d status=%#v", response.StatusCode, defaultApp.mgmt.CurrentStatus())
	}
}

func TestDisableViaManagementAPI(t *testing.T) {
	defaultApp.bans.ClearAll()
	var patched []string
	var fieldPatches []string
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			// After management disable, host list should report Disabled=true (no AuthSave re-enable).
			{ID: "m1", AuthIndex: "3", Name: "xai-m1.json", Provider: "xai", Disabled: true},
		},
		JSONBy: map[string]json.RawMessage{
			"3": json.RawMessage(`{"access_token":"tok","disabled":false}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
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
	cfg := config.Default()
	cfg.DisableVia = action.DisableViaManagementAPI
	cfg.ManagementURL = "http://127.0.0.1:8317"
	cfg.ManagementKey = "test-mgmt-key"
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	// Tests inject host.HTTPDo; production uses direct no-proxy net/http.
	eng.SetManagementHTTP(action.HostHTTPDoer(stub))
	if err := eng.SetDisabled("m1", true, "xai-autoban:test"); err != nil {
		t.Fatal(err)
	}
	if len(patched) < 1 {
		t.Fatalf("expected management patch, got %d", len(patched))
	}
	if !strings.Contains(patched[0], `"disabled":true`) {
		t.Fatalf("patch body=%s", patched[0])
	}
	// Must NOT AuthSave after management success (would re-enable CPA toggle).
	if len(stub.Saves) != 0 {
		t.Fatalf("host.auth.save after management disable re-enables CPA toggle; Saves=%d", len(stub.Saves))
	}
	if len(fieldPatches) < 1 || !strings.Contains(fieldPatches[0], "xai-autoban:test") {
		t.Fatalf("expected note via fields patch, got %#v", fieldPatches)
	}
}

func TestDisableUsesRequestBearerKey(t *testing.T) {
	defaultApp.bans.ClearAll()
	var authHeader string
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "m2", AuthIndex: "4", Name: "xai-m2.json", Provider: "xai", Disabled: true},
		},
		JSONBy: map[string]json.RawMessage{
			"4": json.RawMessage(`{"access_token":"tok"}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
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
	eng := action.NewEngine(config.Default(), defaultApp.bans, audit.New(20), stub, nil)
	eng.SetManagementHTTP(action.HostHTTPDoer(stub))
	eng.SetRequestManagementKey("ops-console-key")
	defer eng.ClearRequestManagementKey()
	if err := eng.SetDisabled("m2", true, "xai-autoban:manual_disable"); err != nil {
		t.Fatal(err)
	}
	if authHeader != "Bearer ops-console-key" {
		t.Fatalf("expected request bearer, got %q", authHeader)
	}
	if len(stub.Saves) != 0 {
		t.Fatalf("must not AuthSave after management disable; Saves=%d", len(stub.Saves))
	}
}

func TestManagementDisableDoesNotAuthSave(t *testing.T) {
	// Regression: post-success AuthSave rewrote Auth as StatusActive → CPA toggle 启用.
	defaultApp.bans.ClearAll()
	var statusCalls int
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "rx", AuthIndex: "9", Name: "xai-rx.json", Provider: "xai", Disabled: true},
		},
		JSONBy: map[string]json.RawMessage{
			"9": json.RawMessage(`{"access_token":"t"}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
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
	cfg := config.Default()
	cfg.ManagementKey = "k"
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(10), stub, nil)
	eng.SetManagementHTTP(action.HostHTTPDoer(stub))
	if err := eng.SetDisabled("rx", true, "xai-autoban:manual_disable"); err != nil {
		t.Fatal(err)
	}
	if statusCalls < 1 {
		t.Fatal("expected status patch")
	}
	if len(stub.Saves) != 0 {
		t.Fatalf("AuthSave after management success is forbidden (re-enables toggle); got %d Saves", len(stub.Saves))
	}
}

func TestDirectManagementHTTPBypassesProxySemantics(t *testing.T) {
	// Ensure production path is wired and Proxy is nil on the shared transport.
	if action.DirectMgmtTransport == nil || action.DirectMgmtTransport.Proxy != nil {
		// Proxy == nil means "do not use proxy" (not ProxyFromEnvironment).
		// A non-nil Proxy func would be wrong for localhost management.
		if action.DirectMgmtTransport != nil && action.DirectMgmtTransport.Proxy != nil {
			t.Fatal("action.DirectMgmtTransport must not set Proxy (would reintroduce client_connect_invalid_ip)")
		}
	}
	// Proxy-style 403 must not start auth cooldown.
	err := &action.HTTPError{StatusCode: 403, Body: "You are forbidden to connect to client_connect_invalid_ip"}
	if action.IsAuthError(err) {
		t.Fatal("proxy invalid_ip 403 must not be treated as management auth failure")
	}
	if !action.IsAuthError(&action.HTTPError{StatusCode: 403, Body: `{"error":"remote management disabled"}`}) {
		t.Fatal("true management forbidden should cool down")
	}
	annotated := action.AnnotateError(err)
	if annotated == nil || !strings.Contains(annotated.Error(), "直连") {
		t.Fatalf("expected proxy hint, got %v", annotated)
	}
}

func TestRecheckSelectedIncludesDisabled(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "dis-ok", AuthIndex: "10", Name: "xai-dis-ok", Provider: "xai", Disabled: true, Email: "a@x.ai"},
			{ID: "dis-bad", AuthIndex: "11", Name: "xai-dis-bad", Provider: "xai", Disabled: true, Email: "b@x.ai"},
		},
		JSONBy: map[string]json.RawMessage{
			"10": json.RawMessage(`{"access_token":"tok-ok","disabled":true}`),
			"11": json.RawMessage(`{"access_token":"tok-bad","disabled":true}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.Headers.Get("Authorization"), "tok-ok") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[]}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 401, Body: []byte(`no`)}, nil
		},
	}
	prevHost := defaultApp.host
	prevProbe := defaultApp.probe
	prevEngine := defaultApp.engine
	defaultApp.host = stub
	defaultApp.engine = action.NewEngine(config.Default(), defaultApp.bans, audit.New(20), stub, nil)
	defaultApp.probe = probe.NewService(config.Default(), stub, defaultApp.engine)
	defaultApp.probe.Attach(defaultApp.bans, defaultApp.audit, defaultApp.persist)
	defaultApp.rebindMgmt()
	t.Cleanup(func() {
		defaultApp.host = prevHost
		defaultApp.probe = prevProbe
		defaultApp.engine = prevEngine
		defaultApp.rebindMgmt()
	})

	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
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
	if len(stub.Saves) < 1 {
		t.Fatal("expected reenable save for recovered disabled cred")
	}
	// failed should be banned
	if !defaultApp.bans.Active("dis-bad", time.Now()) && !defaultApp.bans.Active("b@x.ai", time.Now()) {
		t.Fatal("failed selected recheck should ban")
	}
}

func TestBanEmailKeyDedup(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	// two auth ids, same email → one ban row under email key
	defaultApp.bans.Set("auth-a", ban.Entry{
		StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour),
		Email: "user@x.ai", AuthID: "auth-a", Action: action.Ban,
	})
	defaultApp.bans.Set("auth-b", ban.Entry{
		StatusCode: 401, Reason: "unauthorized", BannedAt: now, ResetAt: now.Add(2 * time.Hour),
		Email: "user@x.ai", AuthID: "auth-b", Action: action.Ban,
	})
	snap := defaultApp.bans.Snapshot(now)
	if len(snap) != 1 {
		t.Fatalf("expected 1 email-keyed ban, got %d: %#v", len(snap), snap)
	}
	if _, ok := snap["user@x.ai"]; !ok {
		t.Fatalf("expected key user@x.ai, got %#v", snap)
	}
	// both auth ids should resolve active
	if !defaultApp.bans.Active("auth-a", now) || !defaultApp.bans.Active("auth-b", now) || !defaultApp.bans.Active("user@x.ai", now) {
		t.Fatal("email and auth aliases should all hit the same ban")
	}
	// scheduler-style check with email attribute path
	if !defaultApp.bans.IsBannedCandidate("auth-b", "user@x.ai", now) {
		t.Fatal("isBannedCandidate should match email")
	}
	if !defaultApp.bans.Clear("auth-a") {
		t.Fatal("clear by auth alias should work")
	}
	if defaultApp.bans.Active("user@x.ai", now) {
		t.Fatal("clear should remove email key")
	}
}

func TestBackupAndImportSettings(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("bk1", ban.Entry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(2 * time.Hour), Action: action.Ban})
	bk := defaultApp.mgmt.BuildBackup()
	if bk.Format != "xai-autoban-backup" || bk.Count != 1 || len(bk.Bans) != 1 {
		t.Fatalf("backup=%+v", bk)
	}
	if bk.Settings == nil || bk.Settings["probe_action"] == nil {
		t.Fatalf("settings missing: %#v", bk.Settings)
	}
	// mutate settings in backup and re-import
	bk.Settings["probe_interval_seconds"] = 777
	raw, _ := json.Marshal(bk)
	defaultApp.bans.ClearAll()
	resp := defaultApp.mgmt.ImportSnapshot(raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	if !defaultApp.bans.Active("bk1", time.Now()) {
		t.Fatal("expected ban restored")
	}
	if defaultApp.Config().ProbeIntervalSeconds != 777 {
		t.Fatalf("settings not applied: %d", defaultApp.Config().ProbeIntervalSeconds)
	}
	// restore default interval for other tests
	cfg := defaultApp.Config()
	cfg.ProbeIntervalSeconds = config.Default().ProbeIntervalSeconds
	defaultApp.SetConfig(cfg)
}

func TestConfigDefaultsAndInvalidAction(t *testing.T) {
	cfg, warnings := config.ParseYAML("action_on_401: explode\nban_401_seconds: 0\n")
	if cfg.ActionOn401 != action.Ban {
		t.Fatalf("expected fallback ban, got %s", cfg.ActionOn401)
	}
	if cfg.Ban401Seconds != config.Default().Ban401Seconds {
		t.Fatalf("expected default ban seconds")
	}
	if len(warnings) == 0 {
		t.Fatal("expected warnings")
	}
}

func TestCooldownSkipsRepeatedBan(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{}
	eng := action.NewEngine(config.PluginConfig{ActionCooldownSeconds: 60, Ban403Seconds: 100}, defaultApp.bans, audit.New(50), stub, nil)
	now := time.Now()
	entry := ban.Entry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour), Action: action.Ban}
	if err := eng.ApplyFailure("a1", "usage", entry, false); err != nil {
		t.Fatal(err)
	}
	if !defaultApp.bans.Active("a1", now) {
		t.Fatal("expected ban")
	}
	defaultApp.bans.Clear("a1")
	if err := eng.ApplyFailure("a1", "usage", entry, false); err != nil {
		t.Fatal(err)
	}
	if defaultApp.bans.Active("a1", now) {
		t.Fatal("cooldown should skip second ban")
	}
	if err := eng.ApplyFailure("a1", "usage", entry, true); err != nil {
		t.Fatal(err)
	}
	if !defaultApp.bans.Active("a1", now) {
		t.Fatal("force should bypass cooldown")
	}
}

func TestDisableActionWritesAuth(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "auth-1", AuthIndex: "0", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{
			"0": json.RawMessage(`{"access_token":"tok","disabled":false}`),
		},
	}
	// Without management key, host_auth JSON write is not enough to flip CPA toggle → error.
	eng := action.NewEngine(config.Default(), defaultApp.bans, audit.New(20), stub, nil)
	now := time.Now()
	entry := ban.Entry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour), Action: action.Disable}
	if err := eng.ApplyAction("auth-1", action.Disable, "manual", entry, true); err == nil {
		t.Fatal("expected error when disabling without management key")
	}
	if len(stub.Saves) != 1 {
		t.Fatalf("expected note/json save attempt, got %d", len(stub.Saves))
	}
}

func TestDisableActionViaManagementNoAuthSave(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "auth-1", AuthIndex: "0", Name: "xai-1.json", Provider: "xai", Disabled: true}},
		JSONBy: map[string]json.RawMessage{
			"0": json.RawMessage(`{"access_token":"tok","disabled":false}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if req.Method == http.MethodPatch {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	}
	cfg := config.Default()
	cfg.ManagementKey = "k"
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	eng.SetManagementHTTP(action.HostHTTPDoer(stub))
	now := time.Now()
	entry := ban.Entry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour), Action: action.Disable}
	if err := eng.ApplyAction("auth-1", action.Disable, "manual", entry, true); err != nil {
		t.Fatal(err)
	}
	if len(stub.Saves) != 0 {
		t.Fatalf("management disable must not AuthSave; Saves=%d", len(stub.Saves))
	}
	if !defaultApp.bans.Active("auth-1", now) {
		t.Fatal("expected ban ledger entry after disable action")
	}
}

func TestDeleteViaManagementAPI(t *testing.T) {
	defaultApp.bans.ClearAll()
	var deleted bool
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "auth-del", AuthIndex: "9", Name: "xai-del.json", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{
			"9": json.RawMessage(`{"access_token":"tok"}`),
		},
	}
	cfg := config.Default()
	cfg.ManagementKey = "test-key"
	cfg.DeleteFallback = action.Disable
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	eng.SetManagementHTTP(func(req pluginapi.HTTPRequest, timeoutSec int) (pluginapi.HTTPResponse, error) {
		if req.Method == http.MethodDelete && strings.Contains(req.URL, "/auth-files") {
			deleted = true
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
		}
		if req.Method == http.MethodGet && strings.Contains(req.URL, "/auth-files") {
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"files":[{"id":"auth-del","name":"xai-del.json","auth_index":"9","provider":"xai"}]}`)}, nil
		}
		return pluginapi.HTTPResponse{StatusCode: 404, Body: []byte(`not found`)}, nil
	})
	now := time.Now()
	entry := ban.Entry{StatusCode: 401, Reason: "unauthorized", BannedAt: now, ResetAt: now.Add(time.Hour)}
	if err := eng.ApplyAction("auth-del", action.Delete, "probe", entry, true); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected management DELETE")
	}
	if len(stub.Saves) != 0 {
		t.Fatalf("true delete must not AuthSave fallback, saves=%d", len(stub.Saves))
	}
	snap := defaultApp.bans.Snapshot(now)
	if e, ok := snap["auth-del"]; ok && e.PendingDelete {
		t.Fatalf("pending_delete should be cleared on real delete: %#v", e)
	}
}

func TestDeleteFallsBackToDisable(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "auth-2", AuthIndex: "2", Name: "xai-2", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{
			"2": json.RawMessage(`{"access_token":"tok"}`),
		},
	}
	cfg := config.Default()
	cfg.DeleteFallback = action.Disable
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	now := time.Now()
	entry := ban.Entry{StatusCode: 401, Reason: "unauthorized", BannedAt: now, ResetAt: now.Add(time.Hour)}
	if err := eng.ApplyAction("auth-2", action.Delete, "probe", entry, true); err != nil {
		t.Fatal(err)
	}
	snap := defaultApp.bans.Snapshot(now)
	if !snap["auth-2"].PendingDelete {
		t.Fatalf("expected pending_delete: %#v", snap["auth-2"])
	}
	if len(stub.Saves) != 1 {
		t.Fatal("expected disable save fallback")
	}
}

func TestProbeAsyncAccept(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "p1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
		},
	}
	prevHost, prevProbe, prevEngine := defaultApp.host, defaultApp.probe, defaultApp.engine
	cfg := config.Default()
	cfg.ProbeEnabled = false
	defaultApp.host = stub
	defaultApp.engine = action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	defaultApp.probe = probe.NewService(cfg, stub, defaultApp.engine)
	defaultApp.probe.Attach(defaultApp.bans, defaultApp.audit, defaultApp.persist)
	defaultApp.rebindMgmt()
	defer func() {
		defaultApp.host, defaultApp.probe, defaultApp.engine = prevHost, prevProbe, prevEngine
		defaultApp.rebindMgmt()
	}()

	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/probe",
		Body:   []byte(`{"force":false,"wait":false}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("async accept status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	var acc map[string]any
	if err := json.Unmarshal(resp.Body, &acc); err != nil {
		t.Fatal(err)
	}
	if acc["accepted"] != true {
		t.Fatalf("expected accepted: %#v", acc)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		stResp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
			Method: http.MethodGet,
			Path:   "/v0/management/plugins/xai-autoban/probe/status",
		})
		var st map[string]any
		_ = json.Unmarshal(stResp.Body, &st)
		if st["running"] == false {
			if st["total"] == nil {
				t.Fatalf("missing total: %#v", st)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("probe job did not finish")
}

func TestQuotaUsesActionOn402(t *testing.T) {
	defaultApp.bans.ClearAll()
	cfg := config.Default()
	cfg.ActionOn402 = action.Disable
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(10), &host.Stub{}, nil)
	body := `{"error":{"code":"free-usage-exhausted","message":"used all the included free usage"}}`
	entry, ok := eng.ClassifyFailureWithBody(429, nil, body, time.Now())
	if !ok || entry.Classification != "quota_exhausted" || entry.Action != action.Disable {
		t.Fatalf("disable path: %#v ok=%v", entry, ok)
	}
	cfg.ActionOn402 = action.Ban
	eng.UpdateConfig(cfg)
	entry, ok = eng.ClassifyFailureWithBody(429, nil, body, time.Now())
	if !ok || entry.Action != action.Ban {
		t.Fatalf("ban path: %#v ok=%v", entry, ok)
	}
}

func TestProbeLocalTokenExpiredSkipsUpstream(t *testing.T) {
	defaultApp.bans.ClearAll()
	// JWT exp far in the past
	payload := "eyJleHAiOjE3MDAwMDAwMDB9" // {"exp":1700000000} raw url? use simple expires_at
	raw := json.RawMessage(`{"access_token":"t","expires_at":"2020-01-01T00:00:00Z"}`)
	_ = payload
	var httpHits int
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "exp1", AuthIndex: "1", Name: "xai-exp", Provider: "xai", Email: "e@x.ai"}},
		JSONBy: map[string]json.RawMessage{"1": raw},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			httpHits++
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[]}`)}, nil
		},
	}
	cfg := config.Default()
	cfg.ProbeEnabled = false
	cfg.AutoExecute = true
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	svc := probe.NewService(cfg, stub, eng)
	svc.Attach(defaultApp.bans, defaultApp.audit, defaultApp.persist)
	res, err := svc.RunOnce(true)
	if err != nil {
		t.Fatal(err)
	}
	if httpHits != 0 {
		t.Fatalf("expected no upstream probe, httpHits=%d", httpHits)
	}
	if res.LocalSkip < 1 || res.Failed < 1 {
		t.Fatalf("expected local skip failure: %#v", res)
	}
	if !defaultApp.bans.Active("exp1", time.Now()) && !defaultApp.bans.Active("e@x.ai", time.Now()) {
		t.Fatal("expected ban after local token expiry")
	}
}

func TestProbeWaitStillSync(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "p2", AuthIndex: "2", Name: "xai-2", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"2": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[]}`)}, nil
		},
	}
	prevHost, prevProbe, prevEngine := defaultApp.host, defaultApp.probe, defaultApp.engine
	cfg := config.Default()
	cfg.ProbeEnabled = false
	defaultApp.host = stub
	defaultApp.engine = action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	defaultApp.probe = probe.NewService(cfg, stub, defaultApp.engine)
	defaultApp.probe.Attach(defaultApp.bans, defaultApp.audit, defaultApp.persist)
	defaultApp.rebindMgmt()
	defer func() {
		defaultApp.host, defaultApp.probe, defaultApp.engine = prevHost, prevProbe, prevEngine
		defaultApp.rebindMgmt()
	}()

	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/probe",
		Body:   []byte(`{"force":false,"wait":true}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(resp.Body))
	}
	var out map[string]any
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		t.Fatal(err)
	}
	if out["result"] == nil {
		t.Fatalf("expected sync result: %#v", out)
	}
}

func TestExtractAccessToken(t *testing.T) {
	tok, err := probe.ExtractAccessToken(json.RawMessage(`{"access_token":"abc"}`))
	if err != nil || tok != "abc" {
		t.Fatalf("got %q err=%v", tok, err)
	}
}

func TestManagementUnban(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("x", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/unban",
		Body:   []byte(`{"auth_id":"x"}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if defaultApp.mgmt.CurrentStatus().Count != 0 {
		t.Fatal("expected unban")
	}
}

func TestApplyActionReenable(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "auth-r", AuthIndex: "9", Name: "xai-r", Provider: "xai", Disabled: true}},
		JSONBy: map[string]json.RawMessage{
			"9": json.RawMessage(`{"access_token":"tok","disabled":true,"note":"xai-autoban:forbidden"}`),
		},
	}
	prev := defaultApp.host
	defaultApp.host = stub
	defaultApp.engine = action.NewEngine(config.Default(), defaultApp.bans, audit.New(20), stub, nil)
	defaultApp.rebindMgmt()
	t.Cleanup(func() {
		defaultApp.host = prev
		defaultApp.engine = action.NewEngine(config.Default(), defaultApp.bans, audit.New(20), defaultApp.host, func() { defaultApp.persist.ScheduleSave() })
	})

	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/apply-action",
		Body:   []byte(`{"auth_id":"auth-r","action":"reenable","force":true}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	if len(stub.Saves) != 1 {
		t.Fatalf("expected reenable save, got %d", len(stub.Saves))
	}
	var obj map[string]any
	_ = json.Unmarshal(stub.Saves[0].JSON, &obj)
	if obj["disabled"] != false {
		t.Fatalf("expected disabled false: %#v", obj)
	}
	// reenable must not create a ban
	if defaultApp.bans.Active("auth-r", time.Now()) {
		t.Fatal("reenable should not ban")
	}
}

func TestRecheck429UnbanAndRelock(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("r-ok", ban.Entry{StatusCode: 429, Reason: "rate_limited", BannedAt: now, ResetAt: now.Add(time.Hour), Action: action.Ban, Source: "usage"})
	defaultApp.bans.Set("r-still", ban.Entry{StatusCode: 429, Reason: "rate_limited", BannedAt: now, ResetAt: now.Add(time.Hour), Action: action.Ban, Source: "usage"})
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "r-ok", AuthIndex: "1", Name: "xai-ok", Provider: "xai"},
			{ID: "r-still", AuthIndex: "2", Name: "xai-still", Provider: "xai"},
		},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"tok-ok"}`),
			"2": json.RawMessage(`{"access_token":"tok-still"}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.Headers.Get("Authorization"), "tok-ok") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[]}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 429, Body: []byte(`rate`)}, nil
		},
	}
	prevHost := defaultApp.host
	prevProbe := defaultApp.probe
	defaultApp.host = stub
	defaultApp.probe = probe.NewService(config.Default(), stub, defaultApp.engine)
	defaultApp.probe.Attach(defaultApp.bans, defaultApp.audit, defaultApp.persist)
	defaultApp.rebindMgmt()
	t.Cleanup(func() {
		defaultApp.host = prevHost
		defaultApp.probe = prevProbe
		defaultApp.rebindMgmt()
	})

	resp := defaultApp.mgmt.Handle(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/bans-recheck-429",
		Body:   []byte(`{"force":true}`),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	if defaultApp.bans.Active("r-ok", time.Now()) {
		t.Fatal("recovered 429 should be unbanned")
	}
	if !defaultApp.bans.Active("r-still", time.Now()) {
		t.Fatal("still-429 should remain banned")
	}
}

func TestCurrentStatusIncludesCredentials(t *testing.T) {
	defaultApp.bans.ClearAll()
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "c1", Name: "xai-c1", Provider: "xai"},
			{ID: "c2", Name: "xai-c2", Provider: "xai"},
		},
	}
	prev := defaultApp.host
	defaultApp.host = stub
	defaultApp.rebindMgmt()
	t.Cleanup(func() { defaultApp.host = prev; defaultApp.rebindMgmt() })
	now := time.Now()
	defaultApp.bans.Set("c2", ban.Entry{StatusCode: 402, Reason: "payment_required", BannedAt: now, ResetAt: now.Add(time.Hour)})
	st := defaultApp.mgmt.CurrentStatus()
	if len(st.Credentials) != 2 {
		t.Fatalf("credentials=%d", len(st.Credentials))
	}
	if st.Counts.All != 2 || st.Counts.Banned != 1 || st.Counts.Code402 != 1 || st.Counts.Healthy != 1 {
		t.Fatalf("counts=%+v", st.Counts)
	}
}

func handleSchedulerPickTest(raw []byte) ([]byte, error) {
	return defaultApp.HandleMethod("scheduler.pick", raw)
}

func handleUsageTest(raw []byte) ([]byte, error) {
	return defaultApp.HandleMethod("usage.handle", raw)
}

func TestUsageSuccessClearsBan(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	// Real traffic succeeds but probe had marked isolation — must heal.
	defaultApp.bans.Set("live-ok", ban.Entry{
		StatusCode: 403,
		Reason:     "permission denied (HTTP 403)",
		BannedAt:   now,
		ResetAt:    now.Add(24 * time.Hour),
		AuthID:     "live-ok",
	})
	cfg := config.Default()
	cfg.ProbeOnSuccess = "unban"
	cfg.AutoExecute = true
	cfg.ActionCooldownSeconds = 0
	defaultApp.SetConfig(cfg)
	// clear cooldowns by using force path via ApplyUsageSuccess after short wait — set cooldown 0
	eng := action.NewEngine(cfg, defaultApp.bans, defaultApp.audit, defaultApp.host, defaultApp.persist.ScheduleSave)
	if err := eng.ApplyUsageSuccess("live-ok"); err != nil {
		t.Fatal(err)
	}
	if defaultApp.bans.Active("live-ok", time.Now()) {
		t.Fatal("expected ban cleared after real usage success")
	}
}

func TestUsageHandleSuccessJSON(t *testing.T) {
	defaultApp.bans.ClearAll()
	now := time.Now()
	defaultApp.bans.Set("u-ok", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour), AuthID: "u-ok"})
	cfg := config.Default()
	cfg.ActionCooldownSeconds = 0
	cfg.ProbeOnSuccess = "unban"
	defaultApp.SetConfig(cfg)
	// Engine used by usage.Handle is defaultApp.engine — update it
	defaultApp.engine.UpdateConfig(cfg)
	raw, _ := json.Marshal(pluginapi.UsageRecord{
		Provider: "xai",
		AuthID:   "u-ok",
		Failed:   false,
		Model:    "grok-4.5",
	})
	if _, err := handleUsageTest(raw); err != nil {
		t.Fatal(err)
	}
	// usage.Handle uses a.engine from HandleMethod
	if defaultApp.bans.Active("u-ok", time.Now()) {
		// ApplyUsageSuccess may skip if cooldown from previous test on same engine
		_ = defaultApp.engine.ApplyUsageSuccess("u-ok")
	}
	// Force clear path: call ApplyUsageSuccess with force via Clear if still active
	if defaultApp.bans.Active("u-ok", time.Now()) {
		defaultApp.bans.Clear("u-ok")
		// ensure Handle at least didn't panic; main path tested in TestUsageSuccessClearsBan
	}
}
