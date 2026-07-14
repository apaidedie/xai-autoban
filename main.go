package main

import (
	"encoding/json"
	"log/slog"
	"sync"

	"xai-autoban/cpasdk/pluginabi"
	"xai-autoban/cpasdk/pluginapi"
)

const (
	pluginName    = "xai-autoban"
	pluginVersion = "0.4.0"
	providerXAI   = "xai"
)

var (
	bans      banState
	audit     = newAuditLog(200)
	cfgMu     sync.RWMutex
	activeCfg = defaultConfig()
	hostImpl  HostClient = realHostClient{}
	persister            = newStatePersister("", &bans)
	engine               = newActionEngine(defaultConfig(), &bans, audit, hostImpl, func() {
		persister.scheduleSave()
	})
	probeSvc = newProbeService(defaultConfig(), hostImpl, engine)
)

func currentConfig() PluginConfig {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return activeCfg
}

func setConfig(cfg PluginConfig) {
	cfgMu.Lock()
	activeCfg = cfg
	cfgMu.Unlock()
	audit.setMax(cfg.AuditMaxEvents)
	engine.updateConfig(cfg)
	persister.setPath(cfg.StateFile)
	probeSvc.updateConfig(cfg)
}

func main() {}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		return handleRegister(request)
	case pluginabi.MethodPluginShutdown:
		probeSvc.stop()
		persister.flush()
		return okEnvelope(map[string]any{})
	case pluginabi.MethodUsageHandle:
		return handleUsage(request)
	case pluginabi.MethodSchedulerPick:
		return handleSchedulerPick(request)
	case pluginabi.MethodManagementRegister:
		return okEnvelope(managementRegistration())
	case pluginabi.MethodManagementHandle:
		return handleManagement(request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

type registerRequest struct {
	ConfigYAML string `json:"config_yaml"`
}

type registration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      pluginapi.Metadata     `json:"metadata"`
	Capabilities  registrationCapability `json:"capabilities"`
}

type registrationCapability struct {
	UsagePlugin   bool `json:"usage_plugin"`
	Scheduler     bool `json:"scheduler"`
	ManagementAPI bool `json:"management_api"`
}

func handleRegister(raw []byte) ([]byte, error) {
	var req registerRequest
	_ = json.Unmarshal(raw, &req)
	cfg, warnings := parseConfigYAML(req.ConfigYAML)
	for _, w := range warnings {
		slog.Warn("xai-autoban: config warning", "warning", w)
		audit.add("system", "", "config", "warn", w, 0)
	}
	setConfig(cfg)
	persister.load()
	if cfg.ProbeEnabled {
		probeSvc.start()
	}
	return okEnvelope(pluginRegistration())
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           "apaidedie",
			GitHubRepository: "https://github.com/apaidedie/xai-autoban",
			ConfigFields:     configFields(),
		},
		Capabilities: registrationCapability{
			UsagePlugin:   true,
			Scheduler:     true,
			ManagementAPI: true,
		},
	}
}

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func okEnvelope(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	return raw
}
