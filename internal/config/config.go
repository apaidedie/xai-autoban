package config

import (
	"fmt"
	"os"
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
	// AutoExecute mirrors CPA-Manager-Plus Codex inspection:
	// false = report-only (只输出结果), true = apply probe_action / probe_on_success.
	AutoExecute           bool   `yaml:"auto_execute"`
	ActionCooldownSeconds int    `yaml:"action_cooldown_seconds"`
	DeleteFallback        string `yaml:"delete_fallback"`
	SchedulerDelegate     string `yaml:"scheduler_delegate"`
	StateFile             string `yaml:"state_file"`
	AuditMaxEvents        int    `yaml:"audit_max_events"`
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
		Ban401Seconds:                        86400,
		Ban402Seconds:                        604800,
		Ban403Seconds:                        86400,
		Ban429FallbackSeconds:                1800,
		ActionOn401:                          actionBan,
		ActionOn402:                          actionBan,
		ActionOn403:                          actionBan,
		ActionOn429:                          actionBan,
		ProbeEnabled:                         true,
		ProbeIntervalSeconds:                 600,
		ProbeTimeoutSeconds:                  20,
		ProbeConcurrency:                     3,
		ProbeQPS:                             2,
		ProbeMode:                            "models",
		ProbeBaseURL:                         "",
		ProbePath:                            "/models",
		ProbeAction:                          actionBan,
		ProbeOnSuccess:                       successUnban,
		AutoExecute:                          true,
		ActionCooldownSeconds:                60,
		DeleteFallback:                       actionDisable,
		SchedulerDelegate:                    pluginapi.SchedulerBuiltinRoundRobin,
		StateFile:                            "",
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
	if cfg.ActionCooldownSeconds < 0 {
		cfg.ActionCooldownSeconds = def.ActionCooldownSeconds
	}
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
	if cfg.ProbeMode != "models" && cfg.ProbeMode != "responses_mini" {
		cfg.ProbeMode = def.ProbeMode
		warnings = append(warnings, "invalid probe_mode; using models")
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
		"probe_mode":                               c.ProbeMode,
		"probe_base_url":                           c.ProbeBaseURL,
		"probe_path":                               c.ProbePath,
		"probe_action":                             c.ProbeAction,
		"probe_on_success":                         c.ProbeOnSuccess,
		"probe_include_disabled":                   c.ProbeIncludeDisabled,
		"probe_only_disabled":                      c.ProbeOnlyDisabled,
		"auto_execute":                             c.AutoExecute,
		"action_cooldown_seconds":                  c.ActionCooldownSeconds,
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

func MergePatch(base PluginConfig, patch map[string]any) (PluginConfig, []string) {
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
