package usage

import (
	"encoding/json"
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/action"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
)

func TestHandle_UsageSuccessClearsBan(t *testing.T) {
	bans := &ban.State{}
	now := time.Now()
	bans.Set("u-ok", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour), AuthID: "u-ok"})
	cfg := config.Default()
	cfg.ActionCooldownSeconds = 0
	cfg.ProbeOnSuccess = action.SuccessUnban
	eng := action.NewEngine(cfg, bans, audit.New(20), &host.Stub{}, nil)

	raw, _ := json.Marshal(pluginapi.UsageRecord{
		Provider: "xai",
		AuthID:   "u-ok",
		Failed:   false,
		Model:    "grok-4.5",
	})
	Handle(raw, eng)
	if bans.Active("u-ok", time.Now()) {
		t.Fatal("usage success must clear isolation via Handle")
	}
}

func TestHandle_IgnoresNonXAI(t *testing.T) {
	bans := &ban.State{}
	now := time.Now()
	bans.Set("other", ban.Entry{StatusCode: 403, ResetAt: now.Add(time.Hour)})
	eng := action.NewEngine(config.Default(), bans, audit.New(10), &host.Stub{}, nil)
	raw, _ := json.Marshal(pluginapi.UsageRecord{Provider: "openai", AuthID: "other", Failed: false})
	Handle(raw, eng)
	if !bans.Active("other", time.Now()) {
		t.Fatal("non-xai usage must not clear ban")
	}
}
