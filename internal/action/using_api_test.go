package action

import (
	"encoding/json"
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
)

func TestSetUsingAPIHostSaveAndVerify(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1.json", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"tok","using_api":false}`),
		},
	}
	eng := NewEngine(config.Default(), &ban.State{}, audit.New(10), stub, nil)
	if err := eng.SetUsingAPI("a1", true); err != nil {
		t.Fatal(err)
	}
	if len(stub.Saves) != 1 {
		t.Fatalf("saves=%d", len(stub.Saves))
	}
	got, err := stub.AuthGet("1")
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(got.JSON, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["using_api"] != true {
		t.Fatalf("AuthGet using_api=%#v", obj["using_api"])
	}
}

func TestApplyUsingAPIOff(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1.json", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"tok","using_api":true}`),
		},
	}
	eng := NewEngine(config.Default(), &ban.State{}, audit.New(10), stub, nil)
	if err := eng.ApplyAction("a1", UsingAPIOff, "manual", ban.Entry{}, true); err != nil {
		t.Fatal(err)
	}
	got, err := stub.AuthGet("1")
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(got.JSON, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["using_api"] != false {
		t.Fatalf("want using_api=false got %#v", obj["using_api"])
	}
}

func TestDisableDoesNotWriteBanLedger(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "d1", AuthIndex: "1", Name: "xai-d1.json", Provider: "xai", Disabled: true},
		},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"t"}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
		},
	}
	cfg := config.Default()
	cfg.DisableVia = DisableViaManagementAPI
	cfg.ManagementURL = "http://127.0.0.1:8317"
	cfg.ManagementKey = "k"
	cfg.ActionCooldownSeconds = 0
	bans := &ban.State{}
	now := time.Now()
	bans.Set("d1", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour), AuthID: "d1"})
	eng := NewEngine(cfg, bans, audit.New(10), stub, nil)
	eng.SetManagementHTTP(HostHTTPDoer(stub))
	if err := eng.ApplyAction("d1", Disable, "test", ban.Entry{StatusCode: 403, Reason: "forbidden", ResetAt: now.Add(time.Hour)}, true); err != nil {
		t.Fatal(err)
	}
	if bans.Active("d1", time.Now()) {
		t.Fatal("disable must not keep isolation ledger entry")
	}
	// ban-only still isolates
	if err := eng.ApplyAction("d1", Ban, "test", ban.Entry{StatusCode: 429, ResetAt: now.Add(time.Hour)}, true); err != nil {
		t.Fatal(err)
	}
	if !bans.Active("d1", time.Now()) {
		t.Fatal("ban should isolate")
	}
}

func TestSoft403StreakSnapshot(t *testing.T) {
	cfg := config.Default()
	cfg.FailStreak403 = 3
	eng := NewEngine(cfg, &ban.State{}, audit.New(10), &host.Stub{}, nil)
	now := time.Now()
	for i := 0; i < 2; i++ {
		ent, ok := eng.ClassifyFailureWithBody(403, nil, "You do not have permission to access this resource", now)
		if !ok {
			t.Fatalf("expected soft 403 classifiable, i=%d", i)
		}
		if err := eng.ApplyFailure("auth-s", "test", ent, false); err != nil {
			t.Fatal(err)
		}
	}
	snap := eng.Soft403StreakSnapshot()
	if snap["auth-s"] != 2 {
		t.Fatalf("snap=%v want auth-s=2", snap)
	}
	if eng.Soft403Need() != 3 {
		t.Fatalf("need=%d", eng.Soft403Need())
	}
	if eng.bans.Active("auth-s", now) {
		t.Fatal("should not isolate before streak threshold")
	}
}
