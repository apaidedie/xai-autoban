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
		"OPS CONSOLE",
		"编辑配置",
		"probe_on_success",
		"probe_action",
		"当前巡检配置",
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
	eng := newActionEngine(defaultConfig(), &bans, newAuditLog(20), stub, nil)
	now := time.Now()
	entry := banEntry{StatusCode: 403, Reason: "forbidden", BannedAt: now, ResetAt: now.Add(time.Hour), Action: actionDisable}
	if err := eng.applyAction("auth-1", actionDisable, "manual", entry, true); err != nil {
		t.Fatal(err)
	}
	if len(stub.saves) != 1 {
		t.Fatalf("expected save, got %d", len(stub.saves))
	}
	var obj map[string]any
	_ = json.Unmarshal(stub.saves[0].JSON, &obj)
	if obj["disabled"] != true {
		t.Fatalf("expected disabled true: %#v", obj)
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
