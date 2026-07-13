package main

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"xai-autoban/cpasdk/pluginapi"
)

type PluginConfig struct {
	Ban401Seconds           int    `yaml:"ban_401_seconds"`
	Ban402Seconds           int    `yaml:"ban_402_seconds"`
	Ban403Seconds           int    `yaml:"ban_403_seconds"`
	Ban429FallbackSeconds   int    `yaml:"ban_429_fallback_seconds"`
	ActionOn401             string `yaml:"action_on_401"`
	ActionOn402             string `yaml:"action_on_402"`
	ActionOn403             string `yaml:"action_on_403"`
	ActionOn429             string `yaml:"action_on_429"`
	ProbeEnabled            bool   `yaml:"probe_enabled"`
	ProbeIntervalSeconds    int    `yaml:"probe_interval_seconds"`
	ProbeTimeoutSeconds     int    `yaml:"probe_timeout_seconds"`
	ProbeConcurrency        int    `yaml:"probe_concurrency"`
	ProbeQPS                float64 `yaml:"probe_qps"`
	ProbeMode               string `yaml:"probe_mode"`
	ProbeBaseURL            string `yaml:"probe_base_url"`
	ProbePath               string `yaml:"probe_path"`
	ProbeAction             string `yaml:"probe_action"`
	ProbeOnSuccess          string `yaml:"probe_on_success"`
	ActionCooldownSeconds   int    `yaml:"action_cooldown_seconds"`
	DeleteFallback          string `yaml:"delete_fallback"`
	SchedulerDelegate       string `yaml:"scheduler_delegate"`
	StateFile               string `yaml:"state_file"`
	AuditMaxEvents          int    `yaml:"audit_max_events"`
}

func defaultConfig() PluginConfig {
	return PluginConfig{
		Ban401Seconds:         86400,
		Ban402Seconds:         604800,
		Ban403Seconds:         86400,
		Ban429FallbackSeconds: 1800,
		ActionOn401:           actionBan,
		ActionOn402:           actionBan,
		ActionOn403:           actionBan,
		ActionOn429:           actionBan,
		ProbeEnabled:          true,
		ProbeIntervalSeconds:  600,
		ProbeTimeoutSeconds:   20,
		ProbeConcurrency:      3,
		ProbeQPS:              2,
		ProbeMode:             "models",
		ProbeBaseURL:          "",
		ProbePath:             "/models",
		ProbeAction:           actionBan,
		ProbeOnSuccess:        successUnban,
		ActionCooldownSeconds: 60,
		DeleteFallback:        actionDisable,
		SchedulerDelegate:     pluginapi.SchedulerBuiltinRoundRobin,
		StateFile:             "",
		AuditMaxEvents:        200,
	}
}

func parseConfigYAML(raw string) (PluginConfig, []string) {
	cfg := defaultConfig()
	warnings := []string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cfg, warnings
	}
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		warnings = append(warnings, "invalid config yaml, using defaults: "+err.Error())
		return defaultConfig(), warnings
	}
	cfg, more := normalizeConfig(cfg)
	return cfg, append(warnings, more...)
}

func normalizeConfig(cfg PluginConfig) (PluginConfig, []string) {
	warnings := []string{}
	def := defaultConfig()
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

func (c PluginConfig) durationForStatus(status int) time.Duration {
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

func (c PluginConfig) actionForStatus(status int) string {
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

func (c PluginConfig) publicView() map[string]any {
	return map[string]any{
		"ban_401_seconds":           c.Ban401Seconds,
		"ban_402_seconds":           c.Ban402Seconds,
		"ban_403_seconds":           c.Ban403Seconds,
		"ban_429_fallback_seconds":  c.Ban429FallbackSeconds,
		"action_on_401":             c.ActionOn401,
		"action_on_402":             c.ActionOn402,
		"action_on_403":             c.ActionOn403,
		"action_on_429":             c.ActionOn429,
		"probe_enabled":             c.ProbeEnabled,
		"probe_interval_seconds":    c.ProbeIntervalSeconds,
		"probe_timeout_seconds":     c.ProbeTimeoutSeconds,
		"probe_concurrency":         c.ProbeConcurrency,
		"probe_qps":                 c.ProbeQPS,
		"probe_mode":                c.ProbeMode,
		"probe_base_url":            c.ProbeBaseURL,
		"probe_path":                c.ProbePath,
		"probe_action":              c.ProbeAction,
		"probe_on_success":          c.ProbeOnSuccess,
		"action_cooldown_seconds":   c.ActionCooldownSeconds,
		"delete_fallback":           c.DeleteFallback,
		"scheduler_delegate":        c.SchedulerDelegate,
		"state_file":                c.StateFile,
		"audit_max_events":          c.AuditMaxEvents,
	}
}

func mergeConfigPatch(base PluginConfig, patch map[string]any) (PluginConfig, []string) {
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
	return parseConfigYAML(string(out))
}

func configFields() []pluginapi.ConfigField {
	return []pluginapi.ConfigField{
		{Name: "ban_401_seconds", Type: pluginapi.ConfigFieldTypeInteger, Description: "Ban duration for 401 in seconds."},
		{Name: "ban_402_seconds", Type: pluginapi.ConfigFieldTypeInteger, Description: "Ban duration for 402 in seconds."},
		{Name: "ban_403_seconds", Type: pluginapi.ConfigFieldTypeInteger, Description: "Ban duration for 403 in seconds."},
		{Name: "ban_429_fallback_seconds", Type: pluginapi.ConfigFieldTypeInteger, Description: "Fallback ban duration for 429 when Retry-After is missing."},
		{Name: "action_on_401", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{actionBan, actionDisable, actionDelete}, Description: "Action for 401 failures."},
		{Name: "action_on_402", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{actionBan, actionDisable, actionDelete}, Description: "Action for 402 failures."},
		{Name: "action_on_403", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{actionBan, actionDisable, actionDelete}, Description: "Action for 403 failures."},
		{Name: "action_on_429", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{actionBan, actionDisable, actionDelete}, Description: "Action for 429 failures (prefer ban)."},
		{Name: "probe_enabled", Type: pluginapi.ConfigFieldTypeBoolean, Description: "Enable timed credential probing."},
		{Name: "probe_interval_seconds", Type: pluginapi.ConfigFieldTypeInteger, Description: "Probe interval seconds."},
		{Name: "probe_timeout_seconds", Type: pluginapi.ConfigFieldTypeInteger, Description: "Probe HTTP timeout seconds."},
		{Name: "probe_concurrency", Type: pluginapi.ConfigFieldTypeInteger, Description: "Max concurrent probes."},
		{Name: "probe_qps", Type: pluginapi.ConfigFieldTypeInteger, Description: "Global probe requests per second."},
		{Name: "probe_mode", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{"models", "responses_mini"}, Description: "Probe strategy."},
		{Name: "probe_action", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{actionBan, actionDisable, actionDelete}, Description: "Default action when probe fails."},
		{Name: "probe_on_success", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{successNone, successUnban, successReenable, successUnbanAndReenable}, Description: "Action when probe succeeds."},
		{Name: "action_cooldown_seconds", Type: pluginapi.ConfigFieldTypeInteger, Description: "Cooldown between repeated actions for the same credential."},
		{Name: "delete_fallback", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{actionDisable, actionBan}, Description: "Fallback when delete is unavailable."},
		{Name: "scheduler_delegate", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{pluginapi.SchedulerBuiltinRoundRobin, pluginapi.SchedulerBuiltinFillFirst}, Description: "Builtin scheduler after filtering bans."},
		{Name: "state_file", Type: pluginapi.ConfigFieldTypeString, Description: "Optional path to persist ban state."},
		{Name: "audit_max_events", Type: pluginapi.ConfigFieldTypeInteger, Description: "Max in-memory audit events."},
	}
}
