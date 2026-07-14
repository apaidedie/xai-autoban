package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

const (
	actionBan     = "ban"
	actionDisable = "disable"
	actionDelete  = "delete"

	successNone             = "none"
	successUnban            = "unban"
	successReenable         = "reenable"
	successUnbanAndReenable = "unban_and_reenable"
)

type cooldownKey struct {
	AuthID string
	Action string
}

type actionEngine struct {
	mu        sync.Mutex
	cooldown  map[cooldownKey]time.Time
	cfg       PluginConfig
	bans      *banState
	audit     *auditLog
	host      HostClient
	mgmt      *managementDisabler
	onChanged func()
	// requestMgmtKey is set per Management API request (Bearer from ops console).
	requestMgmtKey string
}

func (e *actionEngine) setRequestManagementKey(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requestMgmtKey = strings.TrimSpace(key)
}

func (e *actionEngine) clearRequestManagementKey() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requestMgmtKey = ""
}

func (e *actionEngine) requestManagementKey() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.requestMgmtKey
}

func newActionEngine(cfg PluginConfig, bans *banState, audit *auditLog, host HostClient, onChanged func()) *actionEngine {
	return &actionEngine{
		cooldown:  make(map[cooldownKey]time.Time),
		cfg:       cfg,
		bans:      bans,
		audit:     audit,
		host:      host,
		mgmt:      newManagementDisabler(cfg, host),
		onChanged: onChanged,
	}
}

func (e *actionEngine) updateConfig(cfg PluginConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = cfg
	if e.mgmt != nil {
		e.mgmt.updateConfig(cfg)
	}
}

func (e *actionEngine) classifyFailure(status int, headers http.Header, now time.Time) (banEntry, bool) {
	e.mu.Lock()
	cfg := e.cfg
	e.mu.Unlock()

	entry := banEntry{StatusCode: status, BannedAt: now, Action: cfg.actionForStatus(status)}
	switch status {
	case http.StatusUnauthorized:
		entry.Reason = "unauthorized"
		entry.ResetAt = now.Add(cfg.durationForStatus(status))
	case http.StatusPaymentRequired:
		entry.Reason = "payment_required"
		entry.ResetAt = now.Add(cfg.durationForStatus(status))
	case http.StatusForbidden:
		entry.Reason = "forbidden"
		entry.ResetAt = now.Add(cfg.durationForStatus(status))
	case http.StatusTooManyRequests:
		entry.Reason = "rate_limited"
		entry.ResetAt = rateLimitReset(headers, now)
		if entry.ResetAt.IsZero() {
			entry.ResetAt = now.Add(cfg.durationForStatus(status))
			entry.Reason = "rate_limited_fallback"
		}
	default:
		return banEntry{}, false
	}
	return entry, true
}

func rateLimitReset(headers http.Header, now time.Time) time.Time {
	if headers == nil {
		return time.Time{}
	}
	if raw := strings.TrimSpace(headers.Get("Retry-After")); raw != "" {
		if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil && seconds > 0 {
			return now.Add(time.Duration(seconds) * time.Second)
		}
		if parsed, err := http.ParseTime(raw); err == nil && parsed.After(now) {
			return parsed
		}
	}
	for _, key := range []string{"x-ratelimit-reset", "x-ratelimit-reset-requests"} {
		raw := strings.TrimSpace(headers.Get(key))
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			continue
		}
		if value > 1000000000000 {
			value /= 1000
		}
		reset := time.Unix(value, 0)
		if reset.After(now) {
			return reset
		}
	}
	return time.Time{}
}

func (e *actionEngine) applyFailure(authID, source string, entry banEntry, force bool) error {
	e.mu.Lock()
	cfg := e.cfg
	e.mu.Unlock()
	action := entry.Action
	if action == "" {
		action = cfg.actionForStatus(entry.StatusCode)
	}
	return e.applyAction(authID, action, source, entry, force)
}

func (e *actionEngine) applyAction(authID, action, source string, entry banEntry, force bool) error {
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return fmt.Errorf("missing auth_id")
	}
	action = strings.ToLower(strings.TrimSpace(action))
	// Prefer email as isolation key when available (one mailbox → one ban row).
	if entry.Email == "" {
		entry.Email = e.lookupEmail(authID)
	}
	cooldownKeyID := banStorageKey(entry.Email, authID)
	if !force && e.inCooldown(cooldownKeyID, action) {
		e.audit.add(source, authID, action, "skipped_cooldown", "action skipped due to cooldown", entry.StatusCode)
		return nil
	}

	switch action {
	case actionBan:
		entry.Action = actionBan
		entry.Source = source
		entry.AuthID = authID
		e.bans.set(authID, entry)
		e.markCooldown(cooldownKeyID, action)
		e.audit.add(source, authID, actionBan, "ok", entry.Reason, entry.StatusCode)
		e.notifyChanged()
		return nil
	case actionDisable:
		if err := e.setDisabled(authID, true, fmt.Sprintf("xai-autoban:%s", entry.Reason)); err != nil {
			e.audit.add(source, authID, actionDisable, "error", err.Error(), entry.StatusCode)
			return err
		}
		entry.Action = actionDisable
		entry.Source = source
		entry.AuthID = authID
		e.bans.set(authID, entry)
		e.markCooldown(cooldownKeyID, action)
		e.audit.add(source, authID, actionDisable, "ok", entry.Reason, entry.StatusCode)
		e.notifyChanged()
		return nil
	case actionDelete:
		return e.applyDelete(authID, source, entry, force)
	case successReenable:
		if err := e.setDisabled(authID, false, ""); err != nil {
			e.audit.add(source, authID, successReenable, "error", err.Error(), entry.StatusCode)
			return err
		}
		e.markCooldown(authID, successReenable)
		e.audit.add(source, authID, successReenable, "ok", "manual reenable", entry.StatusCode)
		e.notifyChanged()
		return nil
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}

func (e *actionEngine) applyDelete(authID, source string, entry banEntry, force bool) error {
	e.mu.Lock()
	cfg := e.cfg
	e.mu.Unlock()
	// Host has no formal delete callback. Best-effort fallback.
	fallback := cfg.DeleteFallback
	if fallback == "" {
		fallback = actionDisable
	}
	entry.PendingDelete = true
	entry.Action = actionDelete
	entry.Source = source
	if fallback == actionBan {
		e.bans.set(authID, entry)
		e.markCooldown(authID, actionDelete)
		e.audit.add(source, authID, actionDelete, "fallback", "delete unavailable; ban only", entry.StatusCode)
		e.notifyChanged()
		return nil
	}
	if err := e.setDisabled(authID, true, "xai-autoban:pending_delete"); err != nil {
		e.audit.add(source, authID, actionDelete, "error", err.Error(), entry.StatusCode)
		return err
	}
	e.bans.set(authID, entry)
	e.markCooldown(authID, actionDelete)
	e.audit.add(source, authID, actionDelete, "fallback", "delete unavailable; disabled and pending_delete", entry.StatusCode)
	e.notifyChanged()
	_ = force
	return nil
}

func (e *actionEngine) applySuccess(authID, source string, force bool) error {
	e.mu.Lock()
	cfg := e.cfg
	mode := cfg.ProbeOnSuccess
	e.mu.Unlock()

	switch mode {
	case successNone:
		e.audit.add(source, authID, successNone, "ok", "probe success no-op", 0)
		return nil
	case successUnban:
		if !force && e.inCooldown(authID, successUnban) {
			e.audit.add(source, authID, successUnban, "skipped_cooldown", "", 0)
			return nil
		}
		removed := e.bans.clear(authID)
		e.markCooldown(authID, successUnban)
		e.audit.add(source, authID, successUnban, "ok", fmt.Sprintf("removed=%v", removed), 0)
		e.notifyChanged()
		return nil
	case successReenable:
		if !force && e.inCooldown(authID, successReenable) {
			e.audit.add(source, authID, successReenable, "skipped_cooldown", "", 0)
			return nil
		}
		if err := e.setDisabled(authID, false, ""); err != nil {
			e.audit.add(source, authID, successReenable, "error", err.Error(), 0)
			return err
		}
		e.markCooldown(authID, successReenable)
		e.audit.add(source, authID, successReenable, "ok", "", 0)
		e.notifyChanged()
		return nil
	case successUnbanAndReenable:
		if !force && e.inCooldown(authID, successUnbanAndReenable) {
			e.audit.add(source, authID, successUnbanAndReenable, "skipped_cooldown", "", 0)
			return nil
		}
		_ = e.bans.clear(authID)
		if err := e.setDisabled(authID, false, ""); err != nil {
			e.audit.add(source, authID, successUnbanAndReenable, "error", err.Error(), 0)
			return err
		}
		e.markCooldown(authID, successUnbanAndReenable)
		e.audit.add(source, authID, successUnbanAndReenable, "ok", "", 0)
		e.notifyChanged()
		return nil
	default:
		return fmt.Errorf("unknown probe_on_success %q", mode)
	}
}

func (e *actionEngine) lookupEmail(authID string) string {
	if e.host == nil || strings.TrimSpace(authID) == "" {
		return ""
	}
	files, err := e.host.AuthList()
	if err != nil {
		return ""
	}
	for _, f := range files {
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || authIDsEqual(authKey(f), authID) {
			return strings.ToLower(strings.TrimSpace(f.Email))
		}
	}
	// also match by email itself
	want := strings.ToLower(strings.TrimSpace(authID))
	for _, f := range files {
		if f.Email != "" && strings.EqualFold(f.Email, want) {
			return want
		}
	}
	return ""
}

func (e *actionEngine) setDisabled(authID string, disabled bool, note string) error {
	e.mu.Lock()
	cfg := e.cfg
	host := e.host
	mgmt := e.mgmt
	e.mu.Unlock()
	if host == nil {
		return fmt.Errorf("host callbacks unavailable")
	}
	files, err := host.AuthList()
	if err != nil {
		return err
	}
	var target *pluginapi.HostAuthFileEntry
	for i := range files {
		f := files[i]
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || authIDsEqual(authKey(f), authID) {
			target = &f
			break
		}
	}
	if target == nil {
		return fmt.Errorf("credential not found: %s", authID)
	}
	if target.AuthIndex == "" && target.Name == "" {
		return fmt.Errorf("credential missing auth_index/name: %s", authID)
	}
	index := target.AuthIndex
	if index == "" {
		index = target.Name
	}
	fileName := strings.TrimSpace(target.Name)
	if fileName == "" {
		fileName = strings.TrimSpace(target.ID)
	}
	if fileName == "" {
		fileName = authID
	}

	// CPA UI toggle = Auth.Disabled, only reliably flipped via:
	//   PATCH /v0/management/auth-files/status
	// host.auth.save writes JSON (note) but buildAuthFromFileData does NOT map
	// metadata.disabled -> Auth.Disabled, so the switch stays "启用".
	reqKey := e.requestManagementKey()
	cfgKey := ""
	if mgmt != nil {
		cfgKey = mgmt.resolveKey()
	}
	key := reqKey
	if key == "" {
		key = cfgKey
	}
	forceMgmt := strings.EqualFold(cfg.DisableVia, disableViaManagementAPI)

	// Always try Management API first when we have any key (request Bearer or plugin config).
	var mgmtErr error
	if mgmt != nil && key != "" {
		if err := mgmt.setAuthDisabledWithKey(fileName, index, disabled, key); err != nil {
			mgmtErr = err
			slog.Warn("xai-autoban: management_api disable failed",
				"auth_id", authID, "name", fileName, "disabled", disabled, "error", err)
			if forceMgmt {
				// still try host json for note, then return mgmt error
				_ = e.patchHostAuthJSON(host, index, fileName, disabled, note)
				return fmt.Errorf("management_api disable failed (CPA 开关未改动): %w；请确认 management_url 可达且密钥正确", err)
			}
		} else {
			slog.Info("xai-autoban: updated credential via management api",
				"auth_id", authID, "name", fileName, "disabled", disabled, "key_source", map[bool]string{true: "request", false: "config"}[reqKey != ""])
			_ = e.patchHostAuthJSON(host, index, fileName, disabled, note)
			return nil
		}
	}

	if err := e.patchHostAuthJSON(host, index, fileName, disabled, note); err != nil {
		if mgmtErr != nil {
			return fmt.Errorf("management_api: %v; host_auth: %w", mgmtErr, err)
		}
		return err
	}
	// Management key was available but PATCH failed after host JSON write.
	if mgmtErr != nil {
		return fmt.Errorf("已写入备注，但 Management API 失败、CPA 开关可能仍为启用: %w（检查 management_url 与密钥；运维台须用已保存的管理密钥操作）", mgmtErr)
	}
	// No management key: host_auth only updates note; CPA UI toggle usually will NOT flip.
	if key == "" && disabled {
		slog.Warn("xai-autoban: disabled JSON written without management key — CPA UI toggle may stay 启用. Save management key in ops console (same as remote-management secret).",
			"auth_id", authID)
	}
	slog.Info("xai-autoban: updated credential disabled flag", "auth_id", authID, "disabled", disabled, "via", "host_auth")
	return nil
}

func (e *actionEngine) patchHostAuthJSON(host HostClient, index, fileName string, disabled bool, note string) error {
	got, err := host.AuthGet(index)
	if err != nil {
		return err
	}
	var obj map[string]any
	if len(got.JSON) == 0 {
		obj = map[string]any{}
	} else if err := json.Unmarshal(got.JSON, &obj); err != nil {
		return fmt.Errorf("decode auth json: %w", err)
	}
	obj["disabled"] = disabled
	// Some CPA builds also honor metadata.disabled when synthesizing auth.
	if meta, ok := obj["metadata"].(map[string]any); ok {
		meta["disabled"] = disabled
		obj["metadata"] = meta
	} else if disabled {
		obj["metadata"] = map[string]any{"disabled": true}
	}
	if note != "" {
		obj["note"] = note
		obj["status_message"] = note
	} else if !disabled {
		if msg, _ := obj["status_message"].(string); strings.HasPrefix(msg, "xai-autoban:") {
			delete(obj, "status_message")
		}
		if n, _ := obj["note"].(string); strings.HasPrefix(n, "xai-autoban:") {
			delete(obj, "note")
		}
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	name := got.Name
	if name == "" {
		name = fileName
	}
	if name == "" {
		return fmt.Errorf("missing auth file name")
	}
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name = name + ".json"
	}
	if _, err := host.AuthSave(name, raw); err != nil {
		return err
	}
	return nil
}

func (e *actionEngine) inCooldown(authID, action string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.ActionCooldownSeconds <= 0 {
		return false
	}
	key := cooldownKey{AuthID: authID, Action: action}
	until, ok := e.cooldown[key]
	if !ok {
		return false
	}
	return time.Now().Before(until)
}

func (e *actionEngine) markCooldown(authID, action string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.ActionCooldownSeconds <= 0 {
		return
	}
	e.cooldown[cooldownKey{AuthID: authID, Action: action}] = time.Now().Add(time.Duration(e.cfg.ActionCooldownSeconds) * time.Second)
}

func (e *actionEngine) notifyChanged() {
	if e.onChanged != nil {
		e.onChanged()
	}
}
