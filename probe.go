package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

const defaultXAIBaseURL = "https://api.x.ai/v1"

type probeService struct {
	mu       sync.Mutex
	cfg      PluginConfig
	host     HostClient
	engine   *actionEngine
	stopCh   chan struct{}
	running  bool
	lastRun  time.Time
	lastErr  string
	lastOK   int
	lastFail int
}

func newProbeService(cfg PluginConfig, host HostClient, engine *actionEngine) *probeService {
	return &probeService{cfg: cfg, host: host, engine: engine}
}

func (p *probeService) updateConfig(cfg PluginConfig) {
	p.mu.Lock()
	was := p.running
	p.cfg = cfg
	p.mu.Unlock()
	if was {
		p.stop()
	}
	if cfg.ProbeEnabled {
		p.start()
	}
}

func (p *probeService) start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running || !p.cfg.ProbeEnabled {
		return
	}
	p.stopCh = make(chan struct{})
	p.running = true
	interval := time.Duration(p.cfg.ProbeIntervalSeconds) * time.Second
	go p.loop(interval, p.stopCh)
	slog.Info("xai-autoban: probe loop started", "interval_seconds", p.cfg.ProbeIntervalSeconds)
}

func (p *probeService) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	close(p.stopCh)
	p.running = false
	p.stopCh = nil
	slog.Info("xai-autoban: probe loop stopped")
}

func (p *probeService) loop(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// initial delay: one interval
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if _, err := p.runOnce(false); err != nil {
				slog.Warn("xai-autoban: probe run failed", "error", err)
			}
		}
	}
}

type probeResult struct {
	Checked int `json:"checked"`
	OK      int `json:"ok"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

func (p *probeService) runOnce(force bool) (probeResult, error) {
	p.mu.Lock()
	cfg := p.cfg
	host := p.host
	p.mu.Unlock()
	if host == nil {
		return probeResult{}, fmt.Errorf("host unavailable")
	}
	files, err := host.AuthList()
	if err != nil {
		return probeResult{}, err
	}
	targets := make([]pluginapi.HostAuthFileEntry, 0)
	for _, f := range files {
		if f.Disabled {
			continue
		}
		if !isXAIAuth(f) {
			continue
		}
		targets = append(targets, f)
	}
	res := probeResult{Checked: len(targets)}
	if len(targets) == 0 {
		p.recordRun(res, "")
		return res, nil
	}

	sem := make(chan struct{}, max(1, cfg.ProbeConcurrency))
	var wg sync.WaitGroup
	var mu sync.Mutex
	minInterval := time.Duration(0)
	if cfg.ProbeQPS > 0 {
		minInterval = time.Duration(float64(time.Second) / cfg.ProbeQPS)
	}
	var lastStart time.Time

	for _, file := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(f pluginapi.HostAuthFileEntry) {
			defer wg.Done()
			defer func() { <-sem }()
			if minInterval > 0 {
				mu.Lock()
				wait := minInterval - time.Since(lastStart)
				if wait > 0 {
					time.Sleep(wait)
				}
				lastStart = time.Now()
				mu.Unlock()
			}
			status, perr := p.probeOne(cfg, host, f)
			mu.Lock()
			defer mu.Unlock()
			if perr != nil {
				res.Failed++
				entry, ok := p.engine.classifyFailure(status, nil, time.Now())
				if !ok {
					entry = banEntry{
						StatusCode: status,
						Reason:     "probe_failed",
						BannedAt:   time.Now(),
						ResetAt:    time.Now().Add(cfg.durationForStatus(statusOrFallback(status, cfg))),
						Action:     cfg.ProbeAction,
						Source:     "probe",
					}
					if entry.ResetAt.Equal(entry.BannedAt) {
						entry.ResetAt = time.Now().Add(time.Duration(cfg.Ban403Seconds) * time.Second)
					}
				} else {
					entry.Action = cfg.ProbeAction
					if entry.Action == "" {
						entry.Action = cfg.actionForStatus(status)
					}
					entry.Source = "probe"
				}
				_ = p.engine.applyFailure(authKey(f), "probe", entry, force)
				return
			}
			res.OK++
			_ = p.engine.applySuccess(authKey(f), "probe", force)
		}(file)
	}
	wg.Wait()
	p.recordRun(res, "")
	return res, nil
}

func statusOrFallback(status int, cfg PluginConfig) int {
	if status == 401 || status == 402 || status == 403 || status == 429 {
		return status
	}
	return 403
}

func (p *probeService) probeOne(cfg PluginConfig, host HostClient, f pluginapi.HostAuthFileEntry) (int, error) {
	index := f.AuthIndex
	if index == "" {
		index = f.Name
	}
	got, err := host.AuthGet(index)
	if err != nil {
		return 0, err
	}
	token, err := extractAccessToken(got.JSON)
	if err != nil {
		return 0, err
	}
	base := strings.TrimRight(cfg.ProbeBaseURL, "/")
	if base == "" {
		base = defaultXAIBaseURL
	}
	var req pluginapi.HTTPRequest
	switch cfg.ProbeMode {
	case "responses_mini":
		body, _ := json.Marshal(map[string]any{
			"model":  "grok-3",
			"stream": false,
			"input":  "ping",
		})
		req = pluginapi.HTTPRequest{
			Method: http.MethodPost,
			URL:    base + "/responses",
			Headers: http.Header{
				"Authorization": {"Bearer " + token},
				"Content-Type":  {"application/json"},
				"Accept":        {"application/json"},
			},
			Body: body,
		}
	default:
		path := cfg.ProbePath
		if path == "" {
			path = "/models"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		req = pluginapi.HTTPRequest{
			Method: http.MethodGet,
			URL:    base + path,
			Headers: http.Header{
				"Authorization": {"Bearer " + token},
				"Accept":        {"application/json"},
			},
		}
	}
	resp, err := host.HTTPDo(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, fmt.Errorf("probe status %d", resp.StatusCode)
}

func extractAccessToken(raw json.RawMessage) (string, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", err
	}
	for _, key := range []string{"access_token", "accessToken", "token", "api_key", "apiKey"} {
		if v, ok := obj[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), nil
		}
	}
	if nested, ok := obj["token"].(map[string]any); ok {
		if v, ok := nested["access_token"].(string); ok && v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("access token not found in credential json")
}

func isXAIAuth(f pluginapi.HostAuthFileEntry) bool {
	if strings.EqualFold(f.Provider, providerXAI) || strings.EqualFold(f.Type, providerXAI) {
		return true
	}
	name := strings.ToLower(f.Name)
	return strings.Contains(name, "xai") || strings.Contains(name, "grok")
}

func authKey(f pluginapi.HostAuthFileEntry) string {
	if f.ID != "" {
		return f.ID
	}
	if f.AuthIndex != "" {
		return f.AuthIndex
	}
	return f.Name
}

func (p *probeService) recordRun(res probeResult, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastRun = time.Now()
	p.lastOK = res.OK
	p.lastFail = res.Failed
	p.lastErr = errMsg
}

func (p *probeService) status() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]any{
		"enabled":   p.cfg.ProbeEnabled,
		"running":   p.running,
		"last_run":  p.lastRun.Format(time.RFC3339),
		"last_ok":   p.lastOK,
		"last_fail": p.lastFail,
		"last_err":  p.lastErr,
		"mode":      p.cfg.ProbeMode,
		"interval":  p.cfg.ProbeIntervalSeconds,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
