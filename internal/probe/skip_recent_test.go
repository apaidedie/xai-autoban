package probe

import (
	"encoding/json"
	"testing"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/action"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
)

func TestRunOnceSkipsRecentUsageOK(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "ok", AuthIndex: "1", Name: "ok.json", Provider: "xai"},
			{ID: "cold", AuthIndex: "2", Name: "cold.json", Provider: "xai"},
		},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"t1"}`),
			"2": json.RawMessage(`{"access_token":"t2"}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"id":"x"}`)}, nil
		},
	}
	cfg := config.Default()
	cfg.ProbeMode = "models"
	cfg.ProbeEnabled = false
	cfg.ProbeIncludeDisabled = true
	eng := action.NewEngine(cfg, &ban.State{}, audit.New(10), stub, nil)
	_ = eng.ApplyUsageSuccess("ok")

	p := NewService(cfg, stub, eng)
	p.Attach(&ban.State{}, audit.New(10), nil)
	res, err := p.RunOnceTrigger(false, "test")
	if err != nil {
		t.Fatal(err)
	}
	// only "cold" should be probed; ok skipped
	if res.Skipped < 1 {
		t.Fatalf("expected skip recent usage, got skipped=%d checked=%d", res.Skipped, res.Checked)
	}
	if res.Checked != 1 {
		t.Fatalf("checked=%d want 1 (only cold)", res.Checked)
	}
}

func TestRunOnceForceDoesNotSkip(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "ok", AuthIndex: "1", Name: "ok.json", Provider: "xai"},
		},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"t1"}`),
		},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"id":"x"}`)}, nil
		},
	}
	cfg := config.Default()
	cfg.ProbeMode = "models"
	eng := action.NewEngine(cfg, &ban.State{}, audit.New(10), stub, nil)
	_ = eng.ApplyUsageSuccess("ok")
	p := NewService(cfg, stub, eng)
	res, err := p.RunOnceTrigger(true, "force")
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 0 || res.Checked != 1 {
		t.Fatalf("force should probe all: skipped=%d checked=%d", res.Skipped, res.Checked)
	}
}
