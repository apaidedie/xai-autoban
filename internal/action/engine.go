package action

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
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/classify"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
	"xai-autoban/internal/reauth"
	"xai-autoban/internal/xai"
)

type cooldownKey struct {
	AuthID string
	Action string
}

type failStreak struct {
	Count    int
	LastAt   time.Time
	LastCode int
}

type Engine struct {
	mu       sync.Mutex
	cooldown map[cooldownKey]time.Time
	streaks  map[string]failStreak
	// lastOK: last real usage success time (ground truth). Probe/recheck false positives skipped while fresh.
	lastOK    map[string]time.Time
	cfg       config.PluginConfig
	bans      *ban.State
	audit     *audit.Log
	host      host.Client
	mgmt      *managementDisabler
	onChanged func()
	// requestMgmtKey is set per Management API request (Bearer from ops console).
	requestMgmtKey string
	// onProbeMemo updates last probe result display (wired to probe.Service).
	onProbeMemo func(authID string, ok bool, status int, errMsg string)
	// onUsingAPI updates using_api meta cache after successful write.
	onUsingAPI func(authID, fileName, index string, enabled bool)
}

func (e *Engine) SetRequestManagementKey(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requestMgmtKey = strings.TrimSpace(key)
}

func (e *Engine) ClearRequestManagementKey() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requestMgmtKey = ""
}

func (e *Engine) RequestManagementKey() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.requestMgmtKey
}

// SetProbeMemoHook lets usage success clear "巡检失败" labels after real traffic OK.
func (e *Engine) SetProbeMemoHook(fn func(authID string, ok bool, status int, errMsg string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onProbeMemo = fn
}

// SetUsingAPIHook updates cache after SetUsingAPI succeeds.
func (e *Engine) SetUsingAPIHook(fn func(authID, fileName, index string, enabled bool)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onUsingAPI = fn
}

func (e *Engine) noteProbeOK(authID string) {
	e.mu.Lock()
	fn := e.onProbeMemo
	e.mu.Unlock()
	if fn != nil {
		fn(authID, true, 200, "usage success")
	}
}

func NewEngine(cfg config.PluginConfig, bans *ban.State, audit *audit.Log, host host.Client, onChanged func()) *Engine {
	return &Engine{
		cooldown:  make(map[cooldownKey]time.Time),
		streaks:   make(map[string]failStreak),
		lastOK:    make(map[string]time.Time),
		cfg:       cfg,
		bans:      bans,
		audit:     audit,
		host:      host,
		mgmt:      newManagementDisabler(cfg, host),
		onChanged: onChanged,
	}
}

// usageSuccessGrace is how long real traffic success protects against probe/recheck false isolates.
const usageSuccessGrace = 30 * time.Minute

func (e *Engine) markUsageOK(authID string) {
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.lastOK == nil {
		e.lastOK = make(map[string]time.Time)
	}
	e.lastOK[authID] = time.Now()
}

func (e *Engine) recentUsageOK(authID string) bool {
	return e.RecentUsageOK(authID)
}

// RecentUsageOK reports whether real traffic succeeded within the grace window.
// Probe may skip these accounts to cut load on large fleets.
func (e *Engine) RecentUsageOK(authID string) bool {
	authID = strings.TrimSpace(authID)
	if authID == "" || e == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	t, ok := e.lastOK[authID]
	if !ok {
		return false
	}
	return time.Since(t) < usageSuccessGrace
}

// UsageSuccessGrace returns the probe-skip / false-positive grace duration.
func UsageSuccessGrace() time.Duration { return usageSuccessGrace }

func probeLikeSource(source string) bool {
	s := strings.ToLower(strings.TrimSpace(source))
	return strings.HasPrefix(s, "probe") || strings.HasPrefix(s, "recheck")
}

func (e *Engine) UpdateConfig(cfg config.PluginConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = cfg
	if e.mgmt != nil {
		e.mgmt.updateConfig(cfg)
	}
}

// ClassifyFailure maps upstream failures into a ban ledger entry.
// body is optional response text (used for free-usage / reauth / permission semantics).
func (e *Engine) ClassifyFailure(status int, headers http.Header, now time.Time) (ban.Entry, bool) {
	return e.ClassifyFailureWithBody(status, headers, "", now)
}

func (e *Engine) ClassifyFailureWithBody(status int, headers http.Header, body string, now time.Time) (ban.Entry, bool) {
	e.mu.Lock()
	cfg := e.cfg
	e.mu.Unlock()

	judged := classify.Probe(classify.Input{Status: status, Body: body})
	if !judged.Isolate {
		// Preserve legacy status-only isolation for classic codes when body is empty
		// and classifier said no-isolate only for non-classic failures.
		if body == "" {
			switch status {
			case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusTooManyRequests:
				// continue below with status-based path
			default:
				return ban.Entry{}, false
			}
		} else {
			return ban.Entry{}, false
		}
	}

	// Prefer semantic classification; fall back to status-only for empty body.
	sc := judged.StatusCode
	if sc == 0 {
		sc = status
	}
	// Remap free-usage (often 429) duration to 402 window when classified as quota.
	durationStatus := sc
	if judged.Classification == classify.QuotaExhausted {
		durationStatus = http.StatusPaymentRequired
	}

	entry := ban.Entry{
		StatusCode:     sc,
		BannedAt:       now,
		Classification: judged.Classification,
		Reason:         judged.Reason,
		Action:         cfg.ActionForStatus(durationStatus),
	}

	// Override action from recommended when safer / more specific.
	switch judged.RecommendedAction {
	case classify.ActionBan:
		entry.Action = Ban
	case classify.ActionDisable:
		// For bare rate limit we force ban; for quota/permission use config or disable.
		if judged.Classification == classify.RateLimited {
			entry.Action = Ban
		} else if entry.Action == "" {
			entry.Action = Disable
		}
	case classify.ActionDelete:
		// Keep configured action for 401 unless user set delete; still record classification.
		if entry.Action == "" {
			entry.Action = Ban
		}
	}

	switch {
	case judged.Classification == classify.RateLimited || sc == http.StatusTooManyRequests:
		if entry.Reason == "" {
			entry.Reason = "rate_limited"
		}
		entry.ResetAt = rateLimitReset(headers, now)
		if entry.ResetAt.IsZero() {
			entry.ResetAt = now.Add(cfg.DurationForStatus(http.StatusTooManyRequests))
			if entry.Reason == "rate_limited" || entry.Reason == "temporary rate limit (HTTP 429)" {
				entry.Reason = "rate_limited_fallback"
			}
		}
		// Bare 429: always ban for isolation (never disable from auto path via recommended).
		if judged.Classification == classify.RateLimited {
			entry.Action = Ban
		}
	case sc == http.StatusUnauthorized || judged.Classification == classify.Reauth:
		if entry.Reason == "" {
			entry.Reason = "unauthorized"
		}
		entry.ResetAt = now.Add(cfg.DurationForStatus(http.StatusUnauthorized))
	case sc == http.StatusPaymentRequired || judged.Classification == classify.QuotaExhausted:
		if entry.Reason == "" {
			entry.Reason = "payment_required"
		}
		// Always honor action_on_402 for quota / free-usage exhaustion.
		entry.Action = cfg.ActionOn402
		if entry.Action == "" {
			entry.Action = cfg.ActionForStatus(http.StatusPaymentRequired)
		}
		entry.ResetAt = now.Add(cfg.DurationForStatus(http.StatusPaymentRequired))
	case sc == http.StatusForbidden || judged.Classification == classify.PermissionDenied:
		if entry.Reason == "" {
			entry.Reason = "forbidden"
		}
		entry.ResetAt = now.Add(cfg.DurationForStatus(http.StatusForbidden))
	default:
		// Isolated by classifier with non-classic status: use 403 window as fallback.
		if !judged.Isolate {
			return ban.Entry{}, false
		}
		entry.ResetAt = now.Add(cfg.DurationForStatus(http.StatusForbidden))
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

func (e *Engine) ApplyFailure(authID, source string, entry ban.Entry, force bool) error {
	e.mu.Lock()
	cfg := e.cfg
	e.mu.Unlock()
	action := entry.Action
	if action == "" {
		action = cfg.ActionForStatus(entry.StatusCode)
	}
	// Real chat success is ground truth: do not re-isolate from probe/recheck false positives.
	if !force && probeLikeSource(source) && e.recentUsageOK(authID) {
		e.audit.Add(source, authID, action, "skipped_usage_ok",
			fmt.Sprintf("recent real traffic success within %s; probe path may differ from chat", usageSuccessGrace),
			entry.StatusCode)
		return nil
	}
	// Soft 403 / permission-denied: optional consecutive failures (default fail_streak_403=1 → once).
	if !force && soft403NeedsStreak(entry) {
		n, need, ok := e.bumpSoft403Streak(authID, entry.StatusCode, cfg.FailStreak403, cfg.FailStreakWindowSeconds)
		if !ok {
			e.audit.Add(source, authID, action, "streak", fmt.Sprintf("soft_403 %d/%d (not isolating yet)", n, need), entry.StatusCode)
			return nil
		}
		entry.Reason = fmt.Sprintf("%s · streak %d", entry.Reason, n)
	} else {
		e.clearStreak(authID)
	}
	return e.ApplyAction(authID, action, source, entry, force)
}

// soft403NeedsStreak: transient HTTP 403 permission denials only.
// Never applies to 401/402/429 or quota_exhausted (even if reason text looks similar).
func soft403NeedsStreak(entry ban.Entry) bool {
	if entry.Classification == classify.QuotaExhausted || entry.StatusCode == http.StatusPaymentRequired {
		return false
	}
	if entry.StatusCode == http.StatusUnauthorized || entry.StatusCode == http.StatusTooManyRequests {
		return false
	}
	// Status set and not 403 → not soft 403 (e.g. legacy mis-tags).
	if entry.StatusCode != 0 && entry.StatusCode != http.StatusForbidden {
		return false
	}
	if entry.StatusCode != http.StatusForbidden && entry.Classification != classify.PermissionDenied {
		return false
	}
	blob := strings.ToLower(entry.Reason + " " + entry.Classification)
	// Permanent account issues: isolate immediately.
	for _, hard := range []string{"deactivated", "suspended", "banned", "invalid_grant", "token is expired", "token has been invalidated"} {
		if strings.Contains(blob, hard) {
			return false
		}
	}
	return true
}

func (e *Engine) bumpSoft403Streak(authID string, statusCode, need, windowSec int) (count, threshold int, isolate bool) {
	if need <= 1 {
		return 1, 1, true
	}
	if windowSec <= 0 {
		windowSec = 1800
	}
	now := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.streaks == nil {
		e.streaks = make(map[string]failStreak)
	}
	key := strings.TrimSpace(authID)
	st := e.streaks[key]
	if !st.LastAt.IsZero() && now.Sub(st.LastAt) > time.Duration(windowSec)*time.Second {
		st = failStreak{}
	}
	st.Count++
	st.LastAt = now
	st.LastCode = statusCode
	e.streaks[key] = st
	if st.Count >= need {
		delete(e.streaks, key)
		return st.Count, need, true
	}
	return st.Count, need, false
}

func (e *Engine) clearStreak(authID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.streaks == nil {
		return
	}
	delete(e.streaks, strings.TrimSpace(authID))
}

// Soft403StreakSnapshot returns current soft-403 fail counts by auth key (in-memory).
func (e *Engine) Soft403StreakSnapshot() map[string]int {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make(map[string]int, len(e.streaks))
	for k, st := range e.streaks {
		if st.Count > 0 {
			out[k] = st.Count
		}
	}
	return out
}

// Soft403Need returns configured consecutive soft-403 threshold.
func (e *Engine) Soft403Need() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.FailStreak403 <= 0 {
		return 1
	}
	return e.cfg.FailStreak403
}

func (e *Engine) ApplyAction(authID, action, source string, entry ban.Entry, force bool) error {
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return fmt.Errorf("missing auth_id")
	}
	action = strings.ToLower(strings.TrimSpace(action))
	// Prefer email as isolation key when available (one mailbox → one ban row).
	if entry.Email == "" {
		entry.Email = e.lookupEmail(authID)
	}
	cooldownKeyID := ban.StorageKey(entry.Email, authID)
	if !force && e.inCooldown(cooldownKeyID, action) {
		e.audit.Add(source, authID, action, "skipped_cooldown", "action skipped due to cooldown", entry.StatusCode)
		return nil
	}

	switch action {
	case Ban:
		// 禁用与隔离互斥：已禁用号只保留「禁用」，不再写隔离账本。
		if e.hostCredentialDisabled(authID) {
			e.clearIsolation(authID, entry.Email)
			e.audit.Add(source, authID, Ban, "skipped_disabled", "already disabled; isolation not applied", entry.StatusCode)
			e.notifyChanged()
			return nil
		}
		entry.Action = Ban
		entry.Source = source
		entry.AuthID = authID
		e.bans.Set(authID, entry)
		e.markCooldown(cooldownKeyID, action)
		e.audit.Add(source, authID, Ban, "ok", entry.Reason, entry.StatusCode)
		e.notifyChanged()
		return nil
	case Disable:
		// Disable = 仅 CPA 开关关闭；唯一状态直到巡检成功启用或永久禁用。
		if err := e.SetDisabled(authID, true, fmt.Sprintf("xai-autoban:%s", entry.Reason)); err != nil {
			e.audit.Add(source, authID, Disable, "error", err.Error(), entry.StatusCode)
			return err
		}
		e.clearIsolation(authID, entry.Email)
		e.markCooldown(cooldownKeyID, action)
		e.audit.Add(source, authID, Disable, "ok", entry.Reason, entry.StatusCode)
		e.notifyChanged()
		return nil
	case Delete:
		return e.applyDelete(authID, source, entry, force)
	case Reauth:
		return e.applyReauth(authID, source, entry, force)
	case UsingAPI, "enable_api", "api_mode":
		if err := e.SetUsingAPI(authID, true); err != nil {
			e.audit.Add(source, authID, UsingAPI, "error", err.Error(), entry.StatusCode)
			return err
		}
		// Clear isolation so account can be scheduled after switching to API mode.
		_ = e.bans.Clear(authID)
		if entry.Email != "" {
			_ = e.bans.Clear(entry.Email)
		}
		e.markCooldown(cooldownKeyID, UsingAPI)
		e.audit.Add(source, authID, UsingAPI, "ok", "using_api=true", 0)
		e.notifyChanged()
		return nil
	case UsingAPIOff, "disable_api", "disable_api_mode", "using_api_false":
		if err := e.SetUsingAPI(authID, false); err != nil {
			e.audit.Add(source, authID, UsingAPIOff, "error", err.Error(), entry.StatusCode)
			return err
		}
		e.markCooldown(cooldownKeyID, UsingAPIOff)
		e.audit.Add(source, authID, UsingAPIOff, "ok", "using_api=false", 0)
		e.notifyChanged()
		return nil
	case SuccessReenable:
		if err := e.SetDisabled(authID, false, ""); err != nil {
			e.audit.Add(source, authID, SuccessReenable, "error", err.Error(), entry.StatusCode)
			return err
		}
		e.markCooldown(authID, SuccessReenable)
		e.audit.Add(source, authID, SuccessReenable, "ok", "manual reenable", entry.StatusCode)
		e.notifyChanged()
		return nil
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}

func (e *Engine) applyDelete(authID, source string, entry ban.Entry, force bool) error {
	e.mu.Lock()
	cfg := e.cfg
	mgmt := e.mgmt
	hostClient := e.host
	e.mu.Unlock()

	entry.Action = Delete
	entry.Source = source
	entry.AuthID = authID

	// Prefer real Management DELETE when a key is available.
	reqKey := e.RequestManagementKey()
	cfgKey := ""
	if mgmt != nil {
		cfgKey = mgmt.resolveKey()
	}
	key := reqKey
	if key == "" {
		key = cfgKey
	}
	if mgmt != nil && key != "" {
		fileName, index := e.resolveAuthNames(authID, hostClient)
		if err := mgmt.deleteAuthFileWithKey(fileName, index, key); err == nil {
			entry.PendingDelete = false
			e.bans.Clear(authID)
			e.markCooldown(authID, Delete)
			e.audit.Add(source, authID, Delete, "ok", "deleted via management api", entry.StatusCode)
			e.notifyChanged()
			_ = force
			return nil
		} else {
			slog.Warn("xai-autoban: management delete failed; falling back",
				"auth_id", authID, "error", err)
		}
	}

	// Best-effort fallback: ban or disable + pending_delete.
	fallback := cfg.DeleteFallback
	if fallback == "" {
		fallback = Disable
	}
	entry.PendingDelete = true
	if fallback == Ban {
		e.bans.Set(authID, entry)
		e.markCooldown(authID, Delete)
		e.audit.Add(source, authID, Delete, "fallback", "delete unavailable; ban only", entry.StatusCode)
		e.notifyChanged()
		return nil
	}
	if err := e.SetDisabled(authID, true, "xai-autoban:pending_delete"); err != nil {
		e.bans.Set(authID, entry)
		e.markCooldown(authID, Delete)
		e.audit.Add(source, authID, Delete, "fallback", "delete unavailable; ban only (disable incomplete: "+err.Error()+")", entry.StatusCode)
		e.notifyChanged()
		_ = force
		return nil
	}
	e.bans.Set(authID, entry)
	e.markCooldown(authID, Delete)
	e.audit.Add(source, authID, Delete, "fallback", "delete unavailable; disabled and pending_delete", entry.StatusCode)
	e.notifyChanged()
	_ = force
	return nil
}

func (e *Engine) applyReauth(authID, source string, entry ban.Entry, force bool) error {
	e.mu.Lock()
	hostClient := e.host
	e.mu.Unlock()
	if hostClient == nil {
		return fmt.Errorf("host unavailable")
	}
	files, err := hostClient.AuthList()
	if err != nil {
		e.audit.Add(source, authID, Reauth, "error", err.Error(), entry.StatusCode)
		return err
	}
	var target *pluginapi.HostAuthFileEntry
	for i := range files {
		f := files[i]
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || ban.AuthIDsEqual(xai.AuthKey(f), authID) {
			target = &f
			break
		}
	}
	if target == nil {
		err := fmt.Errorf("credential not found: %s", authID)
		e.audit.Add(source, authID, Reauth, "error", err.Error(), entry.StatusCode)
		return err
	}
	res, err := reauth.RefreshOne(hostClient, *target, "")
	if err != nil {
		// Keep isolation with reauth classification when refresh fails.
		entry.Action = Ban
		entry.Source = source
		entry.AuthID = authID
		if entry.Classification == "" {
			entry.Classification = classify.Reauth
		}
		if entry.Reason == "" {
			entry.Reason = "reauth_failed"
		}
		e.bans.Set(authID, entry)
		e.markCooldown(authID, Reauth)
		e.audit.Add(source, authID, Reauth, "error", res.Message, entry.StatusCode)
		e.notifyChanged()
		return err
	}
	// Success: clear ban so scheduler can use the refreshed credential.
	// If post-refresh probe failed, re-isolate as reauth so ops see it.
	if res.OK && !res.ProbeOK && res.ProbeStatus > 0 {
		entry.Action = Ban
		entry.Source = source
		entry.AuthID = authID
		entry.Classification = classify.Reauth
		entry.Reason = "reauth_probe_failed"
		entry.StatusCode = res.ProbeStatus
		if entry.StatusCode == 0 {
			entry.StatusCode = http.StatusUnauthorized
		}
		entry.BannedAt = time.Now()
		entry.ResetAt = time.Now().Add(e.cfgDuration401())
		e.bans.Set(authID, entry)
		e.markCooldown(authID, Reauth)
		e.audit.Add(source, authID, Reauth, "partial", res.Message, res.ProbeStatus)
		e.notifyChanged()
		_ = force
		return fmt.Errorf("%s", res.Message)
	}
	e.bans.Clear(authID)
	e.markCooldown(authID, Reauth)
	e.audit.Add(source, authID, Reauth, "ok", res.Message, 200)
	e.notifyChanged()
	_ = force
	return nil
}

func (e *Engine) cfgDuration401() time.Duration {
	e.mu.Lock()
	cfg := e.cfg
	e.mu.Unlock()
	return cfg.DurationForStatus(http.StatusUnauthorized)
}

// resolveAuthNames returns (fileName, authIndex) for management API calls.
func (e *Engine) resolveAuthNames(authID string, hostClient host.Client) (fileName, index string) {
	fileName, index = authID, ""
	if hostClient == nil {
		return fileName, index
	}
	files, err := hostClient.AuthList()
	if err != nil {
		return fileName, index
	}
	for _, f := range files {
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || ban.AuthIDsEqual(xai.AuthKey(f), authID) {
			if strings.TrimSpace(f.Name) != "" {
				fileName = f.Name
			} else if strings.TrimSpace(f.ID) != "" {
				fileName = f.ID
			}
			index = f.AuthIndex
			if index == "" {
				index = f.Name
			}
			return fileName, index
		}
	}
	return fileName, index
}

// ApplyUsageSuccess heals isolation when real model traffic succeeds.
// Always clears ban ledger (ground truth: account can call). Reenable follows probe_on_success
// only when auto_execute is on (disable/enable is a stronger CPA-side action).
func (e *Engine) ApplyUsageSuccess(authID string) error {
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return nil
	}
	e.markUsageOK(authID)
	e.noteProbeOK(authID)
	now := time.Now()
	email := e.lookupEmail(authID)
	banned := e.bans.Active(authID, now) || (email != "" && e.bans.Active(email, now)) ||
		e.bans.IsBannedCandidate(authID, email, now)

	e.mu.Lock()
	cfg := e.cfg
	e.mu.Unlock()

	// Real success resets soft-403 streak even if not currently banned.
	e.clearStreak(authID)
	if email != "" {
		e.clearStreak(email)
	}

	// Always clear isolation on real success (even report-only mode).
	if banned {
		if !e.inCooldown(authID, SuccessUnban) {
			removed := e.bans.Clear(authID)
			if email != "" {
				_ = e.bans.Clear(email)
			}
			e.markCooldown(authID, SuccessUnban)
			e.audit.Add("usage", authID, SuccessUnban, "ok", fmt.Sprintf("real traffic success removed=%v", removed), 0)
			e.notifyChanged()
		}
	}

	// Reenable disabled credentials only when auto_execute and success policy asks for it.
	if !cfg.AutoExecute {
		return nil
	}
	switch cfg.ProbeOnSuccess {
	case SuccessReenable, SuccessUnbanAndReenable:
		return e.ApplySuccess(authID, "usage", false)
	default:
		return nil
	}
}

func (e *Engine) ApplySuccess(authID, source string, force bool) error {
	e.mu.Lock()
	cfg := e.cfg
	mode := cfg.ProbeOnSuccess
	e.mu.Unlock()

	switch mode {
	case SuccessNone:
		e.audit.Add(source, authID, SuccessNone, "ok", "probe success no-op", 0)
		return nil
	case SuccessUnban:
		if !force && e.inCooldown(authID, SuccessUnban) {
			e.audit.Add(source, authID, SuccessUnban, "skipped_cooldown", "", 0)
			return nil
		}
		removed := e.bans.Clear(authID)
		e.markCooldown(authID, SuccessUnban)
		e.audit.Add(source, authID, SuccessUnban, "ok", fmt.Sprintf("removed=%v", removed), 0)
		e.notifyChanged()
		return nil
	case SuccessReenable:
		if !force && e.inCooldown(authID, SuccessReenable) {
			e.audit.Add(source, authID, SuccessReenable, "skipped_cooldown", "", 0)
			return nil
		}
		if err := e.SetDisabled(authID, false, ""); err != nil {
			e.audit.Add(source, authID, SuccessReenable, "error", err.Error(), 0)
			return err
		}
		e.markCooldown(authID, SuccessReenable)
		e.audit.Add(source, authID, SuccessReenable, "ok", "", 0)
		e.notifyChanged()
		return nil
	case SuccessUnbanAndReenable:
		if !force && e.inCooldown(authID, SuccessUnbanAndReenable) {
			e.audit.Add(source, authID, SuccessUnbanAndReenable, "skipped_cooldown", "", 0)
			return nil
		}
		_ = e.bans.Clear(authID)
		if err := e.SetDisabled(authID, false, ""); err != nil {
			e.audit.Add(source, authID, SuccessUnbanAndReenable, "error", err.Error(), 0)
			return err
		}
		e.markCooldown(authID, SuccessUnbanAndReenable)
		e.audit.Add(source, authID, SuccessUnbanAndReenable, "ok", "", 0)
		e.notifyChanged()
		return nil
	default:
		return fmt.Errorf("unknown probe_on_success %q", mode)
	}
}

func (e *Engine) lookupEmail(authID string) string {
	if e.host == nil || strings.TrimSpace(authID) == "" {
		return ""
	}
	files, err := e.host.AuthList()
	if err != nil {
		return ""
	}
	for _, f := range files {
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || ban.AuthIDsEqual(xai.AuthKey(f), authID) {
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

// clearIsolation drops ban-ledger rows for authID and optional email.
func (e *Engine) clearIsolation(authID, email string) {
	if e == nil || e.bans == nil {
		return
	}
	_ = e.bans.Clear(authID)
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" {
		_ = e.bans.Clear(email)
	}
}

// hostCredentialDisabled reports CPA Disabled flag for authID (best-effort AuthList).
func (e *Engine) hostCredentialDisabled(authID string) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	h := e.host
	e.mu.Unlock()
	if h == nil {
		return false
	}
	files, err := h.AuthList()
	if err != nil {
		return false
	}
	authID = strings.TrimSpace(authID)
	for _, f := range files {
		if !f.Disabled {
			continue
		}
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || ban.AuthIDsEqual(xai.AuthKey(f), authID) {
			return true
		}
		if email := strings.ToLower(strings.TrimSpace(f.Email)); email != "" && strings.EqualFold(email, authID) {
			return true
		}
	}
	return false
}

// ReconcileDisabledExclusive ensures disabled credentials never keep isolation ledger rows.
// Returns how many ban rows were cleared. Call after state load / on ops status refresh.
func (e *Engine) ReconcileDisabledExclusive() int {
	if e == nil || e.bans == nil {
		return 0
	}
	e.mu.Lock()
	h := e.host
	e.mu.Unlock()
	if h == nil {
		return 0
	}
	files, err := h.AuthList()
	if err != nil || len(files) == 0 {
		return 0
	}
	now := time.Now()
	cleared := 0
	for _, f := range files {
		if !f.Disabled || !xai.IsAuth(f) {
			continue
		}
		key := xai.AuthKey(f)
		email := strings.ToLower(strings.TrimSpace(f.Email))
		had := e.bans.Active(key, now) || (email != "" && e.bans.Active(email, now)) ||
			e.bans.IsBannedCandidate(f.ID, email, now) || e.bans.IsBannedCandidate(key, email, now)
		if !had {
			continue
		}
		e.clearIsolation(key, email)
		if f.ID != "" && f.ID != key {
			_ = e.bans.Clear(f.ID)
		}
		if f.Name != "" {
			_ = e.bans.Clear(f.Name)
		}
		cleared++
	}
	if cleared > 0 {
		e.audit.Add("system", "", "reconcile", "ok", fmt.Sprintf("cleared isolation for %d disabled credentials", cleared), 0)
		e.notifyChanged()
		slog.Info("xai-autoban: disabled exclusive — cleared isolation", "count", cleared)
	}
	return cleared
}

func (e *Engine) SetDisabled(authID string, disabled bool, note string) error {
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
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || ban.AuthIDsEqual(xai.AuthKey(f), authID) {
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
	//
	// CRITICAL: do NOT host.auth.save after a successful management disable.
	// CPA pluginhost.buildAuthFromFileData always sets Status=Active and does NOT
	// map metadata.disabled → Auth.Disabled. AuthSave then manager.Update overwrites
	// the disabled runtime auth and the CPA switch snaps back to 启用 (only note survives).
	reqKey := e.RequestManagementKey()
	cfgKey := ""
	if mgmt != nil {
		cfgKey = mgmt.resolveKey()
	}
	key := reqKey
	if key == "" {
		key = cfgKey
	}
	forceMgmt := strings.EqualFold(cfg.DisableVia, DisableViaManagementAPI)

	// Always try Management API first when we have any key (request Bearer or plugin config).
	var mgmtErr error
	if mgmt != nil && key != "" {
		if err := mgmt.setAuthDisabledWithKey(fileName, index, disabled, key); err != nil {
			mgmtErr = err
			slog.Warn("xai-autoban: management_api disable failed",
				"auth_id", authID, "name", fileName, "disabled", disabled, "error", err)
			if forceMgmt {
				// Do not AuthSave here either — it cannot flip Auth.Disabled and confuses ops.
				return fmt.Errorf("management_api disable failed (CPA 开关未改动): %w；请确认 management_url 可达且密钥正确", err)
			}
		} else {
			// Optional note via fields API (preserves Auth.Disabled). Never AuthSave.
			if note != "" {
				if noteErr := mgmt.patchAuthNoteWithKey(fileName, index, note, key); noteErr != nil {
					slog.Warn("xai-autoban: management note patch failed (disabled state kept)",
						"auth_id", authID, "name", fileName, "error", noteErr)
				}
			}
			// Verify host list reflects Auth.Disabled when available.
			if vErr := e.verifyHostDisabled(host, authID, fileName, index, disabled); vErr != nil {
				slog.Warn("xai-autoban: post-disable verify mismatch",
					"auth_id", authID, "disabled", disabled, "error", vErr)
				// Still treat management success as OK — list may lag one tick; do not AuthSave to "fix".
			}
			slog.Info("xai-autoban: updated credential via management api",
				"auth_id", authID, "name", fileName, "disabled", disabled,
				"key_source", map[bool]string{true: "request", false: "config"}[reqKey != ""],
				"skipped_host_auth_save", true)
			return nil
		}
	}

	// Fallback: host.auth.save only updates JSON/note. Cannot reliably flip CPA UI toggle.
	if err := e.patchHostAuthJSON(host, index, fileName, disabled, note); err != nil {
		if mgmtErr != nil {
			return fmt.Errorf("management_api: %v; host_auth: %w", mgmtErr, err)
		}
		return err
	}
	if mgmtErr != nil {
		return fmt.Errorf("已写入备注，但 Management API 失败、CPA 开关可能仍为启用: %w（检查 management_url 与密钥；运维台须用已保存的管理密钥操作）", mgmtErr)
	}
	if key == "" && disabled {
		return fmt.Errorf("已写入备注，但未调用 Management API（无管理密钥）：CPA 开关不会关闭。请在运维台保存与 remote-management 相同的管理密钥后再禁用")
	}
	slog.Info("xai-autoban: updated credential disabled flag", "auth_id", authID, "disabled", disabled, "via", "host_auth")
	return nil
}

// verifyHostDisabled checks host.auth.list for Auth.Disabled after a management toggle.
func (e *Engine) verifyHostDisabled(host host.Client, authID, fileName, index string, wantDisabled bool) error {
	if host == nil {
		return nil
	}
	files, err := host.AuthList()
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID ||
			f.Name == fileName || f.AuthIndex == index || ban.AuthIDsEqual(xai.AuthKey(f), authID) {
			if f.Disabled != wantDisabled {
				return fmt.Errorf("host list disabled=%v want=%v (name=%s)", f.Disabled, wantDisabled, f.Name)
			}
			return nil
		}
	}
	return fmt.Errorf("credential not found in host list after toggle")
}

func (e *Engine) patchHostAuthJSON(host host.Client, index, fileName string, disabled bool, note string) error {
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

func (e *Engine) inCooldown(authID, action string) bool {
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

func (e *Engine) markCooldown(authID, action string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.ActionCooldownSeconds <= 0 {
		return
	}
	e.cooldown[cooldownKey{AuthID: authID, Action: action}] = time.Now().Add(time.Duration(e.cfg.ActionCooldownSeconds) * time.Second)
}

func (e *Engine) notifyChanged() {
	if e.onChanged != nil {
		e.onChanged()
	}
}

func (e *Engine) ManagementStatus() map[string]any {
	e.mu.Lock()
	mgmt := e.mgmt
	e.mu.Unlock()
	if mgmt == nil {
		return map[string]any{}
	}
	return mgmt.status()
}
func (e *Engine) SetManagementHTTP(fn HTTPDoer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.mgmt != nil {
		e.mgmt.httpDo = fn
	}
}
