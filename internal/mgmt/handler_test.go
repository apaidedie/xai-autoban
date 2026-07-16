package mgmt

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
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
)

func testHandler(t *testing.T) *Handler {
	t.Helper()
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "a1", AuthIndex: "1", Name: "xai-1.json", Provider: "xai", Email: "a@b.com"},
			{ID: "a2", AuthIndex: "2", Name: "xai-2.json", Provider: "xai", Disabled: true},
		},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"t","using_api":true}`),
			"2": json.RawMessage(`{"access_token":"t2","using_api":false}`),
		},
	}
	cfg := config.Default()
	cfg.ProbeEnabled = false
	bans := &ban.State{}
	aud := audit.New(20)
	eng := action.NewEngine(cfg, bans, aud, stub, nil)
	pr := probe.NewService(cfg, stub, eng)
	pr.Attach(bans, aud, nil)
	pers := persist.New("", bans)
	var cur config.PluginConfig = cfg
	h := &Handler{
		Name:    "xai-autoban",
		Version: "test",
		Cfg:     func() config.PluginConfig { return cur },
		SetCfg:  func(c config.PluginConfig) { cur = c },
		Bans:    bans,
		Audit:   aud,
		Engine:  eng,
		Probe:   pr,
		Persist: pers,
		Host:    stub,
		Meta:    creds.NewMetaCache(0),
	}
	return h
}

func TestSettingsUpdateAndRead(t *testing.T) {
	h := testHandler(t)
	body, _ := json.Marshal(map[string]any{
		"probe_interval_seconds": 900,
		"auto_using_api":         "off",
	})
	resp := h.Handle(pluginapi.ManagementRequest{
		Method: http.MethodPut,
		Path:   "/v0/management/plugins/xai-autoban/settings",
		Body:   body,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	var out map[string]any
	_ = json.Unmarshal(resp.Body, &out)
	if out["ok"] != true {
		t.Fatalf("%s", string(resp.Body))
	}
	if h.Cfg().ProbeIntervalSeconds != 900 {
		t.Fatalf("interval=%d", h.Cfg().ProbeIntervalSeconds)
	}
	get := h.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/management/plugins/xai-autoban/settings",
	})
	if get.StatusCode != 200 {
		t.Fatal(get.StatusCode)
	}
}

func TestResourceDataList(t *testing.T) {
	h := testHandler(t)
	resp := h.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/data",
		Query:  url.Values{"page": {"1"}, "page_size": {"50"}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d %s", resp.StatusCode, string(resp.Body))
	}
	var st StatusInfo
	if err := json.Unmarshal(resp.Body, &st); err != nil {
		t.Fatal(err)
	}
	if st.Counts.All < 1 {
		t.Fatalf("counts=%+v", st.Counts)
	}
}

func TestListIDsFilterDisabled(t *testing.T) {
	h := testHandler(t)
	body, _ := json.Marshal(map[string]any{"filter": "disabled", "limit": 100})
	resp := h.Handle(pluginapi.ManagementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/xai-autoban/list-ids",
		Body:   body,
	})
	// list-ids may be resource op only
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == 404 {
		resp = h.Handle(pluginapi.ManagementRequest{
			Method: http.MethodGet,
			Path:   "/v0/resource/plugins/xai-autoban/ops",
			Query:  url.Values{"op": {"list_ids"}, "filter": {"disabled"}, "limit": {"100"}},
		})
	}
	if resp.StatusCode != http.StatusOK {
		// try resource data op
		resp = h.Handle(pluginapi.ManagementRequest{
			Method: http.MethodGet,
			Path:   "/v0/resource/plugins/xai-autoban/data",
			Query:  url.Values{"op": {"list_ids"}, "filter": {"disabled"}},
		})
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body=%s", resp.StatusCode, string(resp.Body))
	}
	var out map[string]any
	_ = json.Unmarshal(resp.Body, &out)
	ids, _ := out["auth_ids"].([]any)
	if len(ids) < 1 {
		t.Fatalf("expected disabled id, out=%s", string(resp.Body))
	}
}

func TestUnbanViaOps(t *testing.T) {
	h := testHandler(t)
	now := time.Now()
	h.Bans.Set("a1", ban.Entry{StatusCode: 403, Reason: "x", BannedAt: now, ResetAt: now.Add(time.Hour)})
	if !h.Bans.Active("a1", time.Now()) {
		t.Fatal("setup ban")
	}
	resp := h.Handle(pluginapi.ManagementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/xai-autoban/ops",
		Query:  url.Values{"op": {"unban"}, "auth_id": {"a1"}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, string(resp.Body))
	}
	if h.Bans.Active("a1", time.Now()) {
		t.Fatal("expected unban")
	}
}

func TestResourcePathMatch(t *testing.T) {
	if !resourcePathMatch("/v0/resource/plugins/xai-autoban/data", "xai-autoban", "data") {
		t.Fatal("data path")
	}
	if !resourcePathMatch("/v0/resource/plugins/xai-autoban/ops", "xai-autoban", "ops") {
		t.Fatal("ops path")
	}
	if resourcePathMatch("/v0/resource/plugins/xai-autoban/status", "xai-autoban", "data") {
		t.Fatal("status should not match data")
	}
}
