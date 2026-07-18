package action

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
)

// Stability contract tests (STABILITY.md §4 CI checklist).
// Named so CI failures map 1:1 to the 1.0 exit criteria.

func stabilityEngine(t *testing.T, cfg config.PluginConfig, stub *host.Stub) (*Engine, *ban.State) {
	t.Helper()
	if stub == nil {
		stub = &host.Stub{}
	}
	bans := &ban.State{}
	cfg.ActionCooldownSeconds = 0
	eng := NewEngine(cfg, bans, audit.New(50), stub, nil)
	return eng, bans
}

func TestStability_Soft403StreakBeforeIsolate(t *testing.T) {
	cfg := config.Default()
	cfg.FailStreak403 = 3
	cfg.ActionOn403 = Ban
	eng, bans := stabilityEngine(t, cfg, nil)
	now := time.Now()
	entry := ban.Entry{
		StatusCode:     http.StatusForbidden,
		Reason:         "permission denied (HTTP 403)",
		Classification: "permission_denied",
		Action:         Ban,
		BannedAt:       now,
		ResetAt:        now.Add(time.Hour),
	}
	for i := 0; i < 2; i++ {
		if err := eng.ApplyFailure("soft-a", "usage", entry, false); err != nil {
			t.Fatal(err)
		}
		if bans.Active("soft-a", time.Now()) {
			t.Fatalf("must not isolate soft 403 on failure %d/3", i+1)
		}
	}
	if err := eng.ApplyFailure("soft-a", "usage", entry, false); err != nil {
		t.Fatal(err)
	}
	if !bans.Active("soft-a", time.Now()) {
		t.Fatal("third soft 403 must isolate")
	}
}

func TestStability_UsageSuccessClearsIsolation(t *testing.T) {
	cfg := config.Default()
	cfg.ProbeOnSuccess = SuccessUnban
	eng, bans := stabilityEngine(t, cfg, nil)
	now := time.Now()
	bans.Set("live", ban.Entry{
		StatusCode: 403,
		Reason:     "permission denied",
		BannedAt:   now,
		ResetAt:    now.Add(24 * time.Hour),
		AuthID:     "live",
	})
	if err := eng.ApplyUsageSuccess("live"); err != nil {
		t.Fatal(err)
	}
	if bans.Active("live", time.Now()) {
		t.Fatal("real usage success must clear isolation")
	}
}

func TestStability_DefaultSoft403IsolatesOnce(t *testing.T) {
	cfg := config.Default()
	if cfg.FailStreak403 != 1 {
		t.Fatalf("default fail_streak_403=%d want 1", cfg.FailStreak403)
	}
	cfg.ActionOn403 = Ban
	eng, bans := stabilityEngine(t, cfg, nil)
	now := time.Now()
	entry := ban.Entry{
		StatusCode:     http.StatusForbidden,
		Reason:         "permission denied (HTTP 403)",
		Classification: "permission_denied",
		Action:         Ban,
		BannedAt:       now,
		ResetAt:        now.Add(time.Hour),
	}
	if err := eng.ApplyFailure("soft-once", "usage", entry, false); err != nil {
		t.Fatal(err)
	}
	if !bans.Active("soft-once", time.Now()) {
		t.Fatal("default fail_streak_403=1: one soft 403 must isolate")
	}
}

func TestStability_Probe402DoesIsolate(t *testing.T) {
	cfg := config.Default()
	cfg.ActionOn402 = Ban
	eng, bans := stabilityEngine(t, cfg, nil)
	now := time.Now()
	entry, ok := eng.ClassifyFailureWithBody(http.StatusPaymentRequired, nil, `{"error":{"code":"free-usage-exhausted","message":"used all free usage"}}`, now)
	if !ok {
		entry, ok = eng.ClassifyFailure(http.StatusPaymentRequired, nil, now)
	}
	if !ok {
		t.Fatal("402 should be classifiable")
	}
	entry.Action = Ban
	entry.ResetAt = now.Add(time.Hour)
	if err := eng.ApplyFailure("p402", "probe", entry, false); err != nil {
		t.Fatal(err)
	}
	if !bans.Active("p402", time.Now()) {
		t.Fatal("probe 402 must isolate once (same as usage)")
	}
	if err := eng.ApplyFailure("p402r", "recheck-selected", entry, false); err != nil {
		t.Fatal(err)
	}
	if !bans.Active("p402r", time.Now()) {
		t.Fatal("recheck 402 must isolate once")
	}
}

func TestStability_Usage402DoesIsolate(t *testing.T) {
	cfg := config.Default()
	cfg.ActionOn402 = Ban
	eng, bans := stabilityEngine(t, cfg, nil)
	now := time.Now()
	// Explicit empty-body 402 (status path) — free-usage body also isolates on usage.
	entry, ok := eng.ClassifyFailure(http.StatusPaymentRequired, nil, now)
	if !ok {
		t.Fatal("402 classifiable")
	}
	entry.Action = Ban
	entry.ResetAt = now.Add(time.Hour)
	if err := eng.ApplyFailure("u402", "usage", entry, false); err != nil {
		t.Fatal(err)
	}
	if !bans.Active("u402", time.Now()) {
		t.Fatalf("real usage 402 must isolate; entry=%+v snap=%v", entry, bans.Snapshot(time.Now()))
	}
}

func TestStability_UsingAPIWriteVerify(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1.json", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"access_token":"tok","using_api":false}`),
		},
	}
	eng, bans := stabilityEngine(t, config.Default(), stub)
	bans.Set("a1", ban.Entry{StatusCode: 403, ResetAt: time.Now().Add(time.Hour), AuthID: "a1"})
	if err := eng.SetUsingAPI("a1", true); err != nil {
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
	if obj["using_api"] != true {
		t.Fatalf("write not reflected: %#v", obj)
	}
}

func TestStability_DeleteFallbackPendingDelete(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "auth-2", AuthIndex: "2", Name: "xai-2", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{
			"2": json.RawMessage(`{"access_token":"t"}`),
		},
		// no management key → delete falls back
	}
	cfg := config.Default()
	cfg.DeleteFallback = Disable
	cfg.DisableVia = DisableViaHostAuth
	eng, bans := stabilityEngine(t, cfg, stub)
	entry := ban.Entry{StatusCode: 401, Reason: "delete_me", Action: Delete, ResetAt: time.Now().Add(time.Hour)}
	if err := eng.ApplyAction("auth-2", Delete, "probe", entry, true); err != nil {
		t.Fatal(err)
	}
	snap := bans.Snapshot(time.Now())
	// storage may be under auth-2
	found := false
	for _, e := range snap {
		if e.PendingDelete {
			found = true
			break
		}
	}
	if !found {
		// host disable path should set pending on ban entry
		if e, ok := snap["auth-2"]; !ok || !e.PendingDelete {
			t.Fatalf("expected pending_delete, snap=%#v saves=%d", snap, len(stub.Saves))
		}
	}
}
