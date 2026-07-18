package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"xai-autoban/cpasdk/pluginapi"
)

const (
	actionBan                  = "ban"
	actionDisable              = "disable"
	actionDelete               = "delete"
	successNone                = "none"
	successUnban               = "unban"
	successReenable            = "reenable"
	successUnbanAndReenable    = "unban_and_reenable"
	disableViaHostAuth         = "host_auth"
	disableViaManagementAPI    = "management_api"
	defaultManagementURL       = "http://127.0.0.1:8317"
	defaultManagementKeyEnv    = "CPA_MANAGEMENT_KEY"
	defaultMgmtTimeoutSec      = 10
	defaultMgmtAuthCooldownSec = 600

	// Auto using_api (probe/recheck only).
	AutoUsingAPIOff    = "off"
	AutoUsingAPIOn403  = "on_403"
	AutoUsingAPIOnFail = "on_fail"
)

type PluginConfig struct {
	Ban401Seconds         int     `yaml:"ban_401_seconds"`
	Ban402Seconds         int     `yaml:"ban_402_seconds"`
	Ban403Seconds         int     `yaml:"ban_403_seconds"`
	Ban429FallbackSeconds int     `yaml:"ban_429_fallback_seconds"`
	ActionOn401           string  `yaml:"action_on_401"`
	ActionOn402           string  `yaml:"action_on_402"`
	ActionOn403           string  `yaml:"action_on_403"`
	ActionOn429           string  `yaml:"action_on_429"`
	ProbeEnabled          bool    `yaml:"probe_enabled"`
	ProbeIntervalSeconds  int     `yaml:"probe_interval_seconds"`
	ProbeTimeoutSeconds   int     `yaml:"probe_timeout_seconds"`
	ProbeConcurrency      int     `yaml:"probe_concurrency"`
	ProbeQPS              float64 `yaml:"probe_qps"`
	ProbeMode             string  `yaml:"probe_mode"`
	ProbeBaseURL          string  `yaml:"probe_base_url"`
	ProbePath             string  `yaml:"probe_path"`
	ProbeAction           string  `yaml:"probe_action"`
	ProbeOnSuccess        string  `yaml:"probe_on_success"`
	// ProbeIncludeDisabled: scheduled/manual full probe also checks disabled creds.
	ProbeIncludeDisabled bool `yaml:"probe_include_disabled"`
	// ProbeOnlyDisabled: only disabled creds (implies include).
	ProbeOnlyDisabled bool `yaml:"probe_only_disabled"`
	// ProbeDisabledConcurrency/QPS: used when ProbeOnlyDisabled (smaller pool, can be more aggressive).
	// 0 = fall back to ProbeConcurrency / ProbeQPS.
	ProbeDisabledConcurrency int     `yaml:"probe_disabled_concurrency"`
	ProbeDisabledQPS         float64 `yaml:"probe_disabled_qps"`
	// AutoExecute mirrors CPA-Manager-Plus Codex inspection:
	// false = report-only (只输出结果), true = apply probe_action / probe_on_success.
	AutoExecute           bool `yaml:"auto_execute"`
	ActionCooldownSeconds int  `yaml:"action_cooldown_seconds"`
	// FailStreak403: soft 403/permission-denied needs this many consecutive failures
	// before isolate (xAI often returns transient 403 then succeeds). Hard bans
	// (suspended/deactivated) still isolate immediately. Default 3.
	FailStreak403 int `yaml:"fail_streak_403"`
	// FailStreakWindowSeconds: reset streak if gap between failures exceeds this. Default 1800.
	FailStreakWindowSeconds int `yaml:"fail_streak_window_seconds"`
	// AutoUsingAPI: off (default, safer) | on_403 | on_fail — auto enable CPA using_api on probe/recheck.
	AutoUsingAPI      string `yaml:"auto_using_api"`
	DeleteFallback    string `yaml:"delete_fallback"`
	SchedulerDelegate string `yaml:"scheduler_delegate"`
	StateFile         string `yaml:"state_file"`
	AuditMaxEvents    int    `yaml:"audit_max_events"`
	// DisableVia: host_auth (default, via host.auth.save) or management_api (CPA Management PATCH /auth-files/status).
	DisableVia                           string `yaml:"disable_via"`
	ManagementURL                        string `yaml:"management_url"`
	ManagementKey                        string `yaml:"management_key"`
	ManagementKeyEnv                     string `yaml:"management_key_env"`
	ManagementTimeoutSeconds             int    `yaml:"management_timeout_seconds"`
	ManagementAuthFailureCooldownSeconds int    `yaml:"management_auth_failure_cooldown_seconds"`
}

func Default() PluginConfig {
	return PluginConfig{
		Ban401Seconds:           86400,
		Ban402Seconds:           604800,
		Ban403Seconds:           86400,
		Ban429FallbackSeconds:   1800,
		ActionOn401:             actionBan,
		ActionOn402:             actionBan,
		ActionOn403:             actionBan,
		ActionOn429:             actionBan,
		ProbeEnabled:            true,
		ProbeIntervalSeconds:    600,
		ProbeTimeoutSeconds:     20,
		ProbeConcurrency:         3,
		ProbeQPS:                 2,
		ProbeDisabledConcurrency: 8,
		ProbeDisabledQPS:         4,
		ProbeMode:                "responses_mini",
		ProbeBaseURL:             "",
		ProbePath:                "/models",
		ProbeAction:              actionBan,
		ProbeOnSuccess:           successUnban,
		AutoExecute:              true,
		ActionCooldownSeconds:   60,
		FailStreak403:           1, // 401/402/403：出现一次即按状态码动作（软 403 连击默认关）
		FailStreakWindowSeconds: 1800,
		AutoUsingAPI:            AutoUsingAPIOff,
		DeleteFallback:          actionDisable,
		SchedulerDelegate:       pluginapi.SchedulerBuiltinRoundRobin,
		// Default state path: bans + ops-console settings overlay (survives reload).
		StateFile:                            "xai-autoban-state.json",
		AuditMaxEvents:                       200,
		DisableVia:                           disableViaHostAuth,
		ManagementURL:                        defaultManagementURL,
		ManagementKey:                        "",
		ManagementKeyEnv:                     defaultManagementKeyEnv,
		ManagementTimeoutSeconds:             defaultMgmtTimeoutSec,
		ManagementAuthFailureCooldownSeconds: defaultMgmtAuthCooldownSec,
	}
}

func ParseYAML(raw string) (PluginConfig, []string) {
	cfg := Default()
	warnings := []string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cfg, warnings
	}
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		warnings = append(warnings, "invalid config yaml, using defaults: "+err.Error())
		return Default(), warnings
	}
	cfg, more := Normalize(cfg)
	return cfg, append(warnings, more...)
}

func Normalize(cfg PluginConfig) (PluginConfig, []string) {
	warnings := []string{}
	def := Default()
	if cfg.Ban401Seconds <= 0 {
		cfg.Ban401Seconds = def.Ban401Seconds
	}
	if cfg.Ban402Seconds <= 0 {
		cfg.Ban402Seconds = def.Ban402Seconds
	}
	if cfg.Ban403Seconds <= 0 {
		cfg.Ban403Seconds = def.Ban403Seconds
	}
	if cfg.Ban429FallbackSeconds <= 0 {
		cfg.Ban429FallbackSeconds = def.Ban429FallbackSeconds
	}
	if cfg.ProbeIntervalSeconds <= 0 {
		cfg.ProbeIntervalSeconds = def.ProbeIntervalSeconds
	}
	if cfg.ProbeTimeoutSeconds <= 0 {
		cfg.ProbeTimeoutSeconds = def.ProbeTimeoutSeconds
	}
	if cfg.ProbeConcurrency <= 0 {
		cfg.ProbeConcurrency = def.ProbeConcurrency
	}
	if cfg.ProbeQPS <= 0 {
		cfg.ProbeQPS = def.ProbeQPS
	}
	if cfg.ProbeDisabledConcurrency < 0 {
		cfg.ProbeDisabledConcurrency = 0
	}
	if cfg.ProbeDisabledQPS < 0 {
		cfg.ProbeDisabledQPS = 0
	}
	if cfg.ActionCooldownSeconds < 0 {
		cfg.ActionCooldownSeconds = def.ActionCooldownSeconds
	}
	if cfg.FailStreak403 <= 0 {
		cfg.FailStreak403 = def.FailStreak403
	}
	if cfg.FailStreakWindowSeconds <= 0 {
		cfg.FailStreakWindowSeconds = def.FailStreakWindowSeconds
	}
	cfg.AutoUsingAPI = normalizeAutoUsingAPI(cfg.AutoUsingAPI, def.AutoUsingAPI, &warnings)
	if cfg.AuditMaxEvents <= 0 {
		cfg.AuditMaxEvents = def.AuditMaxEvents
	}
	cfg.ActionOn401 = normalizeAction(cfg.ActionOn401, def.ActionOn401, &warnings, "action_on_401")
	cfg.ActionOn402 = normalizeAction(cfg.ActionOn402, def.ActionOn402, &warnings, "action_on_402")
	cfg.ActionOn403 = normalizeAction(cfg.ActionOn403, def.ActionOn403, &warnings, "action_on_403")
	cfg.ActionOn429 = normalizeAction(cfg.ActionOn429, def.ActionOn429, &warnings, "action_on_429")
	if cfg.ActionOn429 != actionBan {
		warnings = append(warnings, "action_on_429 is not ban; rate limits are often transient")
	}
	cfg.ProbeAction = normalizeAction(cfg.ProbeAction, def.ProbeAction, &warnings, "probe_action")
	cfg.DeleteFallback = normalizeAction(cfg.DeleteFallback, def.DeleteFallback, &warnings, "delete_fallback")
	if cfg.DeleteFallback == actionDelete {
		cfg.DeleteFallback = actionDisable
		warnings = append(warnings, "delete_fallback cannot be delete; using disable")
	}
	cfg.ProbeOnSuccess = normalizeSuccess(cfg.ProbeOnSuccess, def.ProbeOnSuccess, &warnings)
	cfg.ProbeMode = strings.ToLower(strings.TrimSpace(cfg.ProbeMode))
	if cfg.ProbeMode == "responses" {
		cfg.ProbeMode = "responses_mini"
	}
	if cfg.ProbeMode != "models" && cfg.ProbeMode != "responses_mini" {
		cfg.ProbeMode = def.ProbeMode
		warnings = append(warnings, "invalid probe_mode; using responses_mini")
	}
	if strings.TrimSpace(cfg.ProbePath) == "" {
		cfg.ProbePath = def.ProbePath
	}
	switch strings.ToLower(strings.TrimSpace(cfg.SchedulerDelegate)) {
	case pluginapi.SchedulerBuiltinRoundRobin, pluginapi.SchedulerBuiltinFillFirst:
		cfg.SchedulerDelegate = strings.ToLower(strings.TrimSpace(cfg.SchedulerDelegate))
	default:
		cfg.SchedulerDelegate = def.SchedulerDelegate
		warnings = append(warnings, "invalid scheduler_delegate; using round-robin")
	}
	cfg.DisableVia = strings.ToLower(strings.TrimSpace(cfg.DisableVia))
	if cfg.DisableVia != disableViaHostAuth && cfg.DisableVia != disableViaManagementAPI {
		cfg.DisableVia = def.DisableVia
		warnings = append(warnings, "invalid disable_via; using host_auth")
	}
	if strings.TrimSpace(cfg.ManagementURL) == "" {
		cfg.ManagementURL = def.ManagementURL
	}
	if strings.TrimSpace(cfg.ManagementKeyEnv) == "" {
		cfg.ManagementKeyEnv = def.ManagementKeyEnv
	}
	if cfg.ManagementTimeoutSeconds <= 0 {
		cfg.ManagementTimeoutSeconds = def.ManagementTimeoutSeconds
	}
	if cfg.ManagementAuthFailureCooldownSeconds <= 0 {
		cfg.ManagementAuthFailureCooldownSeconds = def.ManagementAuthFailureCooldownSeconds
	}
	return cfg, warnings
}

func normalizeAction(value, fallback string, warnings *[]string, field string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case actionBan, actionDisable, actionDelete:
		return strings.ToLower(strings.TrimSpace(value))
	case "":
		return fallback
	default:
		*warnings = append(*warnings, fmt.Sprintf("invalid %s=%q; using %s", field, value, fallback))
		return fallback
	}
}

func normalizeSuccess(value, fallback string, warnings *[]string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case successNone, successUnban, successReenable, successUnbanAndReenable:
		return strings.ToLower(strings.TrimSpace(value))
	case "":
		return fallback
	default:
		*warnings = append(*warnings, fmt.Sprintf("invalid probe_on_success=%q; using %s", value, fallback))
		return fallback
	}
}

func normalizeAutoUsingAPI(value, fallback string, warnings *[]string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AutoUsingAPIOff, "false", "0", "no", "disabled":
		return AutoUsingAPIOff
	case AutoUsingAPIOn403, "true", "1", "yes", "403", "on":
		return AutoUsingAPIOn403
	case AutoUsingAPIOnFail, "all", "fail", "any":
		return AutoUsingAPIOnFail
	case "":
		return fallback
	default:
		*warnings = append(*warnings, fmt.Sprintf("invalid auto_using_api=%q; using %s", value, fallback))
		return fallback
	}
}

// EffectiveProbeConcurrency returns concurrency for a probe run.
func (c PluginConfig) EffectiveProbeConcurrency() int {
	if c.ProbeOnlyDisabled && c.ProbeDisabledConcurrency > 0 {
		return c.ProbeDisabledConcurrency
	}
	if c.ProbeConcurrency > 0 {
		return c.ProbeConcurrency
	}
	return 3
}

// EffectiveProbeQPS returns QPS for a probe run.
func (c PluginConfig) EffectiveProbeQPS() float64 {
	if c.ProbeOnlyDisabled && c.ProbeDisabledQPS > 0 {
		return c.ProbeDisabledQPS
	}
	if c.ProbeQPS > 0 {
		return c.ProbeQPS
	}
	return 2
}

func (c PluginConfig) DurationForStatus(status int) time.Duration {
	switch status {
	case 401:
		return time.Duration(c.Ban401Seconds) * time.Second
	case 402:
		return time.Duration(c.Ban402Seconds) * time.Second
	case 403:
		return time.Duration(c.Ban403Seconds) * time.Second
	case 429:
		return time.Duration(c.Ban429FallbackSeconds) * time.Second
	default:
		return 0
	}
}

func (c PluginConfig) ActionForStatus(status int) string {
	switch status {
	case 401:
		return c.ActionOn401
	case 402:
		return c.ActionOn402
	case 403:
		return c.ActionOn403
	case 429:
		return c.ActionOn429
	default:
		return actionBan
	}
}

func (c PluginConfig) PublicView() map[string]any {
	return map[string]any{
		"ban_401_seconds":                          c.Ban401Seconds,
		"ban_402_seconds":                          c.Ban402Seconds,
		"ban_403_seconds":                          c.Ban403Seconds,
		"ban_429_fallback_seconds":                 c.Ban429FallbackSeconds,
		"action_on_401":                            c.ActionOn401,
		"action_on_402":                            c.ActionOn402,
		"action_on_403":                            c.ActionOn403,
		"action_on_429":                            c.ActionOn429,
		"probe_enabled":                            c.ProbeEnabled,
		"probe_interval_seconds":                   c.ProbeIntervalSeconds,
		"probe_timeout_seconds":                    c.ProbeTimeoutSeconds,
		"probe_concurrency":                        c.ProbeConcurrency,
		"probe_qps":                                c.ProbeQPS,
		"probe_disabled_concurrency":               c.ProbeDisabledConcurrency,
		"probe_disabled_qps":                       c.ProbeDisabledQPS,
		"probe_mode":                               c.ProbeMode,
		"probe_base_url":                           c.ProbeBaseURL,
		"probe_path":                               c.ProbePath,
		"probe_action":                             c.ProbeAction,
		"probe_on_success":                         c.ProbeOnSuccess,
		"probe_include_disabled":                   c.ProbeIncludeDisabled,
		"probe_only_disabled":                      c.ProbeOnlyDisabled,
		"auto_execute":                             c.AutoExecute,
		"action_cooldown_seconds":                  c.ActionCooldownSeconds,
		"fail_streak_403":                          c.FailStreak403,
		"fail_streak_window_seconds":               c.FailStreakWindowSeconds,
		"auto_using_api":                           c.AutoUsingAPI,
		"delete_fallback":                          c.DeleteFallback,
		"scheduler_delegate":                       c.SchedulerDelegate,
		"state_file":                               c.StateFile,
		"audit_max_events":                         c.AuditMaxEvents,
		"disable_via":                              c.DisableVia,
		"management_url":                           c.ManagementURL,
		"management_key_env":                       c.ManagementKeyEnv,
		"management_key_configured":                c.ResolveManagementKey() != "",
		"management_timeout_seconds":               c.ManagementTimeoutSeconds,
		"management_auth_failure_cooldown_seconds": c.ManagementAuthFailureCooldownSeconds,
	}
}

// ResolveManagementKey returns plugin-configured or env management key (never log this).
func (c PluginConfig) ResolveManagementKey() string {
	if k := strings.TrimSpace(c.ManagementKey); k != "" {
		return k
	}
	envName := strings.TrimSpace(c.ManagementKeyEnv)
	if envName == "" {
		envName = defaultManagementKeyEnv
	}
	if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
		return v
	}
	for _, e := range []string{"CPA_MANAGEMENT_KEY", "MANAGEMENT_PASSWORD", "MANAGEMENT_KEY", "CPA_MANAGEMENT_PASSWORD", "CLIPROXYAPI_MANAGEMENT_KEY"} {
		if e == envName {
			continue
		}
		if v := strings.TrimSpace(os.Getenv(e)); v != "" {
			return v
		}
	}
	return ""
}

// OpsSettingsKeys are fields the ops console may persist (excludes install secrets).
// FROZEN for 0.9+/1.0: do not remove or rename without a major version after 1.0.
// See STABILITY.md §3.
var OpsSettingsKeys = []string{
	"ban_401_seconds", "ban_402_seconds", "ban_403_seconds", "ban_429_fallback_seconds",
	"action_on_401", "action_on_402", "action_on_403", "action_on_429",
	"probe_enabled", "probe_interval_seconds", "probe_timeout_seconds",
	"probe_concurrency", "probe_qps", "probe_disabled_concurrency", "probe_disabled_qps",
	"probe_mode", "probe_base_url", "probe_path",
	"probe_action", "probe_on_success", "probe_include_disabled", "probe_only_disabled",
	"auto_execute", "action_cooldown_seconds", "fail_streak_403", "fail_streak_window_seconds",
	"auto_using_api", "delete_fallback", "scheduler_delegate", "audit_max_events",
}

// InstallConfigKeys are plugin-manage / install-time fields (may hold secrets).
// FROZEN for 0.9+/1.0 alongside OpsSettingsKeys.
var InstallConfigKeys = []string{
	"disable_via",
	"management_url",
	"management_key",
	"management_key_env",
	"management_timeout_seconds",
	"management_auth_failure_cooldown_seconds",
	"state_file",
}

// OpsSettingsView returns only ops-console fields suitable for state-file overlay.
func (c PluginConfig) OpsSettingsView() map[string]any {
	full := c.PublicView()
	out := make(map[string]any, len(OpsSettingsKeys))
	for _, k := range OpsSettingsKeys {
		if v, ok := full[k]; ok {
			out[k] = v
		}
	}
	return out
}

// CoerceOpsPatch normalizes query-string types (bool/int/float) for ops settings.
func CoerceOpsPatch(patch map[string]any) map[string]any {
	if patch == nil {
		return map[string]any{}
	}
	boolKeys := map[string]struct{}{
		"probe_enabled": {}, "probe_include_disabled": {}, "probe_only_disabled": {}, "auto_execute": {},
	}
	intKeys := map[string]struct{}{
		"ban_401_seconds": {}, "ban_402_seconds": {}, "ban_403_seconds": {}, "ban_429_fallback_seconds": {},
		"probe_interval_seconds": {}, "probe_timeout_seconds": {}, "probe_concurrency": {},
		"probe_disabled_concurrency": {},
		"action_cooldown_seconds": {}, "fail_streak_403": {}, "fail_streak_window_seconds": {},
		"audit_max_events": {},
	}
	floatKeys := map[string]struct{}{"probe_qps": {}, "probe_disabled_qps": {}}
	out := make(map[string]any, len(patch))
	for k, v := range patch {
		if v == nil {
			continue
		}
		s, isStr := v.(string)
		if !isStr {
			out[k] = v
			continue
		}
		s = strings.TrimSpace(s)
		if _, ok := boolKeys[k]; ok {
			lv := strings.ToLower(s)
			out[k] = lv == "1" || lv == "true" || lv == "yes" || lv == "on"
			continue
		}
		if _, ok := intKeys[k]; ok {
			if n, err := strconv.Atoi(s); err == nil {
				out[k] = n
				continue
			}
		}
		if _, ok := floatKeys[k]; ok {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				out[k] = n
				continue
			}
		}
		out[k] = s
	}
	return out
}

func MergePatch(base PluginConfig, patch map[string]any) (PluginConfig, []string) {
	patch = CoerceOpsPatch(patch)
	raw, _ := yaml.Marshal(base)
	var asMap map[string]any
	_ = yaml.Unmarshal(raw, &asMap)
	if asMap == nil {
		asMap = map[string]any{}
	}
	for k, v := range patch {
		if v == nil {
			continue
		}
		asMap[k] = v
	}
	out, err := yaml.Marshal(asMap)
	if err != nil {
		return base, []string{"marshal patch failed: " + err.Error()}
	}
	return ParseYAML(string(out))
}

// Fields is the CPA「插件管理」schema only.
// Daily ops (probe/actions) are configured in the ops console (编辑配置), not here.
// Host still always shows Enable / Priority; we only expose management-key related install fields.
func Fields() []pluginapi.ConfigField {
	return []pluginapi.ConfigField{
		{Name: "management_key_env", Type: pluginapi.ConfigFieldTypeString, Description: "服务端管理密钥环境变量名（默认 CPA_MANAGEMENT_KEY）。日常巡检策略请在运维台「编辑配置」修改。"},
		{Name: "management_key", Type: pluginapi.ConfigFieldTypeString, Description: "服务端管理密钥（不推荐明文；优先用环境变量 management_key_env）。用于插件进程调用 CPA 禁用/删除。"},
		{Name: "management_url", Type: pluginapi.ConfigFieldTypeString, Description: "CPA Management 地址（默认 http://127.0.0.1:8317），用于禁用/删除。"},
		{Name: "disable_via", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{disableViaHostAuth, disableViaManagementAPI}, Description: "禁用凭证路径：host_auth 或 management_api（推荐 management_api + 服务端密钥）。"},
	}
}
