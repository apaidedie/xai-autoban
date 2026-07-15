package main

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"xai-autoban/cpasdk/pluginabi"
	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/action"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
	"xai-autoban/internal/mgmt"
	"xai-autoban/internal/persist"
	"xai-autoban/internal/probe"
	"xai-autoban/internal/schedule"
	"xai-autoban/internal/usage"
)

const (
	pluginName    = "xai-autoban"
	pluginVersion = "0.5.40"
)

type App struct {
	mu      sync.RWMutex
	cfg     config.PluginConfig
	bans    *ban.State
	audit   *audit.Log
	host    host.Client
	engine  *action.Engine
	probe   *probe.Service
	persist *persist.Persister
	mgmt    *mgmt.Handler
}

func NewApp(h host.Client) *App {
	if h == nil {
		h = host.Real{}
	}
	bans := &ban.State{}
	auditLog := audit.New(200)
	persister := persist.New("", bans)
	cfg := config.Default()
	engine := action.NewEngine(cfg, bans, auditLog, h, persister.ScheduleSave)
	probeSvc := probe.NewService(cfg, h, engine)
	probeSvc.Attach(bans, auditLog, persister)
	// Real usage success clears "巡检失败" labels and isolation.
	engine.SetProbeMemoHook(probeSvc.RememberProbeResult)
	app := &App{
		cfg:     cfg,
		bans:    bans,
		audit:   auditLog,
		host:    h,
		engine:  engine,
		probe:   probeSvc,
		persist: persister,
	}
	app.mgmt = &mgmt.Handler{
		Name:    pluginName,
		Version: pluginVersion,
		Cfg:     app.Config,
		SetCfg:  app.SetConfig,
		Bans:    bans,
		Audit:   auditLog,
		Engine:  engine,
		Probe:   probeSvc,
		Persist: persister,
		Host:    h,
	}
	return app
}

func (a *App) Config() config.PluginConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *App) SetConfig(cfg config.PluginConfig) {
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	a.audit.SetMax(cfg.AuditMaxEvents)
	a.engine.UpdateConfig(cfg)
	a.persist.SetPath(cfg.StateFile)
	a.probe.UpdateConfig(cfg)
}

func (a *App) Shutdown() {
	a.probe.Stop()
	a.persist.Flush()
}

// rebindMgmt refreshes handler pointers after tests swap host/engine/probe.
func (a *App) rebindMgmt() {
	if a.mgmt == nil {
		return
	}
	a.mgmt.Cfg = a.Config
	a.mgmt.SetCfg = a.SetConfig
	a.mgmt.Bans = a.bans
	a.mgmt.Audit = a.audit
	a.mgmt.Engine = a.engine
	a.mgmt.Probe = a.probe
	a.mgmt.Persist = a.persist
	a.mgmt.Host = a.host
	if a.probe != nil {
		a.probe.Attach(a.bans, a.audit, a.persist)
	}
}

func (a *App) HandleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		return a.handleRegister(request)
	case pluginabi.MethodPluginShutdown:
		a.Shutdown()
		return okEnvelope(map[string]any{})
	case pluginabi.MethodUsageHandle:
		usage.Handle(request, a.engine)
		return okEnvelope(map[string]any{})
	case pluginabi.MethodSchedulerPick:
		var req pluginapi.SchedulerPickRequest
		if err := json.Unmarshal(request, &req); err != nil {
			return nil, err
		}
		resp := schedule.Pick(req, a.bans, a.Config().SchedulerDelegate)
		return okEnvelope(resp)
	case pluginabi.MethodManagementRegister:
		return okEnvelope(a.mgmt.Registration())
	case pluginabi.MethodManagementHandle:
		var req pluginapi.ManagementRequest
		if err := json.Unmarshal(request, &req); err != nil {
			return nil, err
		}
		return okEnvelope(a.mgmt.Handle(req))
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

func (a *App) handleRegister(raw []byte) ([]byte, error) {
	var req registerRequest
	_ = json.Unmarshal(raw, &req)
	cfg, warnings := config.ParseYAML(req.ConfigYAML)
	for _, w := range warnings {
		slog.Warn("xai-autoban: config warning", "warning", w)
		a.audit.Add("system", "", "config", "warn", w, 0)
	}
	if strings.TrimSpace(cfg.StateFile) == "" {
		cfg.StateFile = config.Default().StateFile
	}
	a.SetConfig(cfg)
	a.persist.Load()
	// Overlay ops-console settings saved in state file (survives yaml reconfigure).
	if overlay := a.persist.Settings(); len(overlay) > 0 {
		merged, more := config.MergePatch(a.Config(), overlay)
		for _, w := range more {
			slog.Warn("xai-autoban: settings overlay warning", "warning", w)
		}
		a.SetConfig(merged)
		slog.Info("xai-autoban: applied persisted ops settings", "keys", len(overlay))
	}
	if a.Config().ProbeEnabled {
		a.probe.Start()
	}
	return okEnvelope(a.pluginRegistration())
}

func (a *App) pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           "apaidedie",
			GitHubRepository: "https://github.com/apaidedie/xai-autoban",
			ConfigFields:     config.Fields(),
		},
		Capabilities: registrationCapability{
			UsagePlugin:   true,
			Scheduler:     true,
			ManagementAPI: true,
		},
	}
}

var defaultApp = NewApp(host.Real{})

func main() {}

func handleMethod(method string, request []byte) ([]byte, error) {
	return defaultApp.HandleMethod(method, request)
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
