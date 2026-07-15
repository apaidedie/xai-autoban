package probe

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/action"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/classify"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
	"xai-autoban/internal/persist"
	"xai-autoban/internal/tokenutil"
	"xai-autoban/internal/xai"
)

const defaultXAIBaseURL = "https://api.x.ai/v1"

type CredentialResult struct {
	At     time.Time `json:"at"`
	OK     bool      `json:"ok"`
	Status int       `json:"status"`
	Error  string    `json:"error,omitempty"`
}

type JobStatus struct {
	Running bool    `json:"running"`
	JobID   int64   `json:"job_id"`
	Done    int     `json:"done"`
	Total   int     `json:"total"`
	Result  *Result `json:"result,omitempty"`
	Error   string  `json:"error,omitempty"`
}

type Service struct {
	mu         sync.Mutex
	cfg        config.PluginConfig
	host       host.Client
	engine     *action.Engine
	bans       *ban.State
	audit      *audit.Log
	persist    *persist.Persister
	stopCh     chan struct{}
	running    bool // scheduled loop
	// jobStarted is set when a manual/async job acquires the flight lock.
	jobStarted time.Time
	lastRun    time.Time
	lastErr    string
	lastOK     int
	lastFail   int
	runSeq     int64
	history    []Run
	lastByAuth map[string]CredentialResult
	// async probe job
	jobRunning bool
	jobID      int64
	jobDone    int
	jobTotal   int
	jobResult  *Result
	jobErr     string
}

func NewService(cfg config.PluginConfig, host host.Client, engine *action.Engine) *Service {
	return &Service{cfg: cfg, host: host, engine: engine}
}

func (p *Service) UpdateConfig(cfg config.PluginConfig) {
	p.mu.Lock()
	was := p.running
	p.cfg = cfg
	p.mu.Unlock()
	if was {
		p.Stop()
	}
	if cfg.ProbeEnabled {
		p.Start()
	}
}

func (p *Service) Start() {
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

func (p *Service) Stop() {
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

func (p *Service) loop(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// initial delay: one interval
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if _, err := p.RunOnceTrigger(false, "scheduled"); err != nil {
				if strings.Contains(err.Error(), "already running") {
					slog.Info("xai-autoban: skip scheduled probe (already in flight)")
				} else {
					slog.Warn("xai-autoban: probe run failed", "error", err)
				}
			}
		}
	}
}

// beginProbe acquires exclusive probe flight. Only one full probe at a time
// (manual async, sync wait, or scheduled).
func (p *Service) beginProbe() (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.jobRunning {
		// Stale lock: no finish after 3h (process stall / lost goroutine).
		if !p.jobStarted.IsZero() && time.Since(p.jobStarted) > 3*time.Hour {
			slog.Warn("xai-autoban: clearing stale probe lock", "age", time.Since(p.jobStarted).String(), "job_id", p.jobID)
			p.jobRunning = false
			p.jobErr = "stale job cleared"
		} else {
			return p.jobID, fmt.Errorf("probe already running")
		}
	}
	p.jobRunning = true
	p.jobStarted = time.Now()
	p.runSeq++
	p.jobID = p.runSeq
	p.jobDone = 0
	p.jobTotal = 0
	p.jobResult = nil
	p.jobErr = ""
	return p.jobID, nil
}

// ForceResetJob clears the in-flight lock so a new probe can start (manual recovery).
func (p *Service) ForceResetJob() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobRunning = false
	p.jobErr = "force reset"
	p.jobStarted = time.Time{}
}

func (p *Service) finishProbe(res Result, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobRunning = false
	p.jobStarted = time.Time{}
	cp := res
	p.jobResult = &cp
	if err != nil {
		p.jobErr = err.Error()
	} else {
		p.jobErr = ""
	}
}

type Result struct {
	Checked     int    `json:"checked"`
	OK          int    `json:"ok"`
	Failed      int    `json:"failed"`
	Skipped     int    `json:"skipped"`
	Banned      int    `json:"banned"`
	Disabled    int    `json:"disabled"`
	Deleted     int    `json:"deleted"`
	Unbanned    int    `json:"unbanned"`
	Reenabled   int    `json:"reenabled"`
	LocalSkip   int    `json:"local_skip"`
	ReportOnly  bool   `json:"report_only"`
	Trigger     string `json:"trigger,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
	AutoExecute bool   `json:"auto_execute"`
	ProbeAction string `json:"probe_action,omitempty"`
	OnSuccess   string `json:"probe_on_success,omitempty"`
}

type Run struct {
	ID     int64  `json:"id"`
	Result Result `json:"result"`
	Error  string `json:"error,omitempty"`
}

const maxProbeHistory = 30

func (p *Service) RunOnce(force bool) (Result, error) {
	return p.RunOnceTrigger(force, "manual")
}

// StartJob runs a probe in the background. Returns job id or error if already running.
// If force is true and a job is already running, the lock is cleared first (new job starts).
func (p *Service) StartJob(force bool, trigger string) (int64, error) {
	id, err := p.beginProbe()
	if err != nil {
		if force && strings.Contains(err.Error(), "already running") {
			p.ForceResetJob()
			id, err = p.beginProbe()
		}
		if err != nil {
			return id, err
		}
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.finishProbe(Result{}, fmt.Errorf("panic: %v", r))
			}
		}()
		res, err := p.runOnceBody(force, trigger)
		p.finishProbe(res, err)
	}()
	return id, nil
}

func (p *Service) JobStatus() JobStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	st := JobStatus{
		Running: p.jobRunning,
		JobID:   p.jobID,
		Done:    p.jobDone,
		Total:   p.jobTotal,
		Error:   p.jobErr,
	}
	if p.jobResult != nil {
		cp := *p.jobResult
		st.Result = &cp
	}
	return st
}

func (p *Service) bumpJobProgress(done, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if total >= 0 {
		p.jobTotal = total
	}
	if done >= 0 {
		p.jobDone = done
	}
}

func (p *Service) RunOnceTrigger(force bool, trigger string) (Result, error) {
	if _, err := p.beginProbe(); err != nil {
		return Result{}, err
	}
	res, err := p.runOnceBody(force, trigger)
	p.finishProbe(res, err)
	return res, err
}

func (p *Service) runOnceBody(force bool, trigger string) (Result, error) {
	p.mu.Lock()
	cfg := p.cfg
	host := p.host
	p.mu.Unlock()
	started := time.Now()
	if host == nil {
		return Result{}, fmt.Errorf("host unavailable")
	}
	files, err := host.AuthList()
	if err != nil {
		return Result{}, err
	}
	targets := make([]pluginapi.HostAuthFileEntry, 0)
	for _, f := range files {
		if !xai.IsAuth(f) {
			continue
		}
		if cfg.ProbeOnlyDisabled {
			if !f.Disabled {
				continue
			}
		} else if !cfg.ProbeIncludeDisabled && f.Disabled {
			continue
		}
		targets = append(targets, f)
	}
	res := Result{
		Checked:     len(targets),
		ReportOnly:  !cfg.AutoExecute,
		AutoExecute: cfg.AutoExecute,
		ProbeAction: cfg.ProbeAction,
		OnSuccess:   cfg.ProbeOnSuccess,
		Trigger:     trigger,
		StartedAt:   started.Format(time.RFC3339),
	}
	p.bumpJobProgress(0, len(targets))
	if len(targets) == 0 {
		res.FinishedAt = time.Now().Format(time.RFC3339)
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
	var finished int

	for _, file := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(f pluginapi.HostAuthFileEntry) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				mu.Lock()
				finished++
				n := finished
				mu.Unlock()
				p.bumpJobProgress(n, -1)
			}()
			if minInterval > 0 {
				mu.Lock()
				wait := minInterval - time.Since(lastStart)
				if wait > 0 {
					time.Sleep(wait)
				}
				lastStart = time.Now()
				mu.Unlock()
			}
			key := xai.AuthKey(f)
			email := strings.ToLower(strings.TrimSpace(f.Email))
			// Single AuthGet: local expiry pre-check then upstream probe.
			authJSON, localEntry, localFail, getErr := p.loadAuthJSON(cfg, host, f, key, email)
			if getErr != nil {
				mu.Lock()
				res.Failed++
				p.RememberProbeResult(key, false, 0, getErr.Error())
				mu.Unlock()
				return
			}
			if localFail {
				mu.Lock()
				res.Failed++
				res.LocalSkip++
				p.RememberProbeResult(key, false, localEntry.StatusCode, localEntry.Reason)
				if cfg.AutoExecute {
					_ = p.engine.ApplyFailure(key, "probe-local", localEntry, force)
					res.Banned++
				} else {
					localEntry.Action = action.Ban
					_ = p.engine.ApplyAction(key, action.Ban, "probe-report", localEntry, force)
					res.Banned++
				}
				mu.Unlock()
				return
			}
			status, body, perr := p.ProbeOneWithJSON(cfg, host, authJSON)
			mu.Lock()
			defer mu.Unlock()
			if perr != nil {
				res.Failed++
				p.RememberProbeResult(key, false, status, perr.Error())
				entry, ok := p.engine.ClassifyFailureWithBody(status, nil, body, time.Now())
				if !ok {
					// model_unavailable / probe_error without isolate: skip ledger
					return
				}
				entry.Source = "probe"
				entry.Email = email
				entry.AuthID = key
				// Prefer classifier action; allow probe_action override only for ban/disable/delete when auto
				if cfg.ProbeAction != "" && entry.Action == action.Ban && cfg.ProbeAction != action.Ban {
					// keep bare-429 as ban; other failures may use configured probe_action
					if entry.Classification != "rate_limited" {
						entry.Action = cfg.ProbeAction
					}
				}
				if !cfg.AutoExecute {
					// 只输出结果: only isolate via ban ledger; never disable/delete; respect soft-403 streak.
					entry.Action = action.Ban
					was := p.engine != nil && (p.bans.Active(key, time.Now()) || p.bans.IsBannedCandidate(key, email, time.Now()))
					_ = p.engine.ApplyFailure(key, "probe-report", entry, false)
					nowBan := p.bans.Active(key, time.Now()) || p.bans.IsBannedCandidate(key, email, time.Now())
					if nowBan && !was {
						res.Banned++
					}
					return
				}
				act := entry.Action
				// force flag is for job unlock only — never force-isolate soft 403s from probe.
				_ = p.engine.ApplyFailure(key, "probe", entry, false)
				switch act {
				case action.Disable:
					res.Disabled++
					res.Banned++
				case action.Delete:
					res.Deleted++
					res.Banned++
				default:
					res.Banned++
				}
				return
			}
			res.OK++
			p.RememberProbeResult(key, true, status, "")
			if !cfg.AutoExecute {
				return
			}
			_ = p.engine.ApplySuccess(key, "probe", force)
			switch cfg.ProbeOnSuccess {
			case action.SuccessUnban:
				res.Unbanned++
			case action.SuccessReenable:
				res.Reenabled++
			case action.SuccessUnbanAndReenable:
				res.Unbanned++
				res.Reenabled++
			}
		}(file)
	}
	wg.Wait()
	res.FinishedAt = time.Now().Format(time.RFC3339)
	p.recordRun(res, "")
	return res, nil
}

func statusOrFallback(status int, cfg config.PluginConfig) int {
	if status == 401 || status == 402 || status == 403 || status == 429 {
		return status
	}
	return 403
}

// loadAuthJSON fetches credential JSON once; reports local expiry without upstream probe.
func (p *Service) loadAuthJSON(cfg config.PluginConfig, h host.Client, f pluginapi.HostAuthFileEntry, key, email string) (json.RawMessage, ban.Entry, bool, error) {
	index := f.AuthIndex
	if index == "" {
		index = f.Name
	}
	got, err := h.AuthGet(index)
	if err != nil {
		return nil, ban.Entry{}, false, err
	}
	now := time.Now()
	local := tokenutil.InspectAuthJSON(got.JSON, now)
	if !local.TokenExpired {
		return got.JSON, ban.Entry{}, false, nil
	}
	entry := ban.Entry{
		StatusCode:     http.StatusUnauthorized,
		Reason:         "token_expired",
		Classification: classify.Reauth,
		BannedAt:       now,
		ResetAt:        now.Add(cfg.DurationForStatus(http.StatusUnauthorized)),
		Action:         cfg.ActionOn401,
		Source:         "probe-local",
		Email:          email,
		AuthID:         key,
	}
	if entry.Action == "" {
		entry.Action = action.Ban
	}
	return got.JSON, entry, true, nil
}

// ProbeOne returns status, response body (for semantic classify), and error on non-2xx / transport failure.
func (p *Service) ProbeOne(cfg config.PluginConfig, host host.Client, f pluginapi.HostAuthFileEntry) (int, string, error) {
	index := f.AuthIndex
	if index == "" {
		index = f.Name
	}
	got, err := host.AuthGet(index)
	if err != nil {
		return 0, "", err
	}
	return p.ProbeOneWithJSON(cfg, host, got.JSON)
}

// ProbeOneWithJSON probes using already-loaded credential JSON (avoids second AuthGet).
// Default (responses / responses_mini): real POST /v1/responses with grok model.
// models: lightweight GET /models only.
func (p *Service) ProbeOneWithJSON(cfg config.PluginConfig, host host.Client, authJSON json.RawMessage) (int, string, error) {
	token, err := ExtractAccessToken(authJSON)
	if err != nil {
		return 0, "", err
	}
	base := strings.TrimRight(cfg.ProbeBaseURL, "/")
	if base == "" {
		base = defaultXAIBaseURL
	}

	authH := func(jsonBody bool) http.Header {
		h := http.Header{
			"Authorization": {"Bearer " + token},
			"Accept":        {"application/json"},
		}
		if jsonBody {
			h.Set("Content-Type", "application/json")
		}
		return h
	}

	do := func(req pluginapi.HTTPRequest) (int, string, error) {
		resp, err := host.HTTPDo(req)
		if err != nil {
			return 0, "", err
		}
		return resp.StatusCode, string(resp.Body), nil
	}

	// One retry on bare 429 (not free-usage exhaustion).
	with429Retry := func(status int, body string, err error, redo func() (int, string, error)) (int, string, error) {
		if err != nil || status != http.StatusTooManyRequests {
			return status, body, err
		}
		parts := classify.ExtractError(body)
		if classify.IsFreeUsageExhausted(parts.Code, parts.Message) {
			return status, body, fmt.Errorf("probe status %d", status)
		}
		time.Sleep(350 * time.Millisecond)
		st2, body2, err2 := redo()
		if err2 == nil && st2 >= 200 && st2 < 300 {
			return st2, body2, nil
		}
		if err2 == nil {
			return st2, body2, fmt.Errorf("probe status %d", st2)
		}
		return st2, body2, err2
	}

	model := "grok-4.5"
	mode := strings.ToLower(strings.TrimSpace(cfg.ProbeMode))
	if mode == "" || mode == "responses" {
		mode = "responses_mini"
	}

	// Optional models list to pick a real model id (best-effort; never required for responses).
	path := cfg.ProbePath
	if path == "" {
		path = "/models"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	modelsStatus, modelsBody, modelsErr := do(pluginapi.HTTPRequest{
		Method:  http.MethodGet,
		URL:     base + path,
		Headers: authH(false),
	})
	if modelsErr == nil && modelsStatus >= 200 && modelsStatus < 300 {
		if m := pickModel(modelsBody); m != "" {
			model = m
		}
		if mode == "models" {
			return modelsStatus, modelsBody, nil
		}
	}

	// Real Responses API request (same family as production grok traffic).
	runResponses := func() (int, string, error) {
		body, _ := json.Marshal(map[string]any{
			"model": model,
			"input": []map[string]any{
				{
					"role": "user",
					"content": []map[string]string{
						{"type": "input_text", "text": "Reply with exactly: OK"},
					},
				},
			},
			"stream": false,
			"max_output_tokens": 16,
		})
		return do(pluginapi.HTTPRequest{
			Method:  http.MethodPost,
			URL:     base + "/responses",
			Headers: authH(true),
			Body:    body,
		})
	}
	// Fallback if responses path denied for this credential shape.
	runCompletions := func() (int, string, error) {
		body, _ := json.Marshal(map[string]any{
			"model":  model,
			"stream": false,
			"max_tokens": 16,
			"messages": []map[string]string{
				{"role": "user", "content": "Reply with exactly: OK"},
			},
		})
		return do(pluginapi.HTTPRequest{
			Method:  http.MethodPost,
			URL:     base + "/chat/completions",
			Headers: authH(true),
			Body:    body,
		})
	}

	// models-only: lightweight list; on hard failures dual-check with real requests.
	if mode == "models" {
		if modelsErr != nil {
			return 0, "", modelsErr
		}
		if modelsStatus >= 200 && modelsStatus < 300 {
			return modelsStatus, modelsBody, nil
		}
		if modelsStatus == 401 || modelsStatus == 402 || modelsStatus == 403 || modelsStatus == 429 {
			st, body, err := runResponses()
			st, body, err = with429Retry(st, body, err, runResponses)
			if err == nil && st >= 200 && st < 300 {
				return st, body, nil
			}
			st2, body2, err2 := runCompletions()
			st2, body2, err2 = with429Retry(st2, body2, err2, runCompletions)
			if err2 == nil && st2 >= 200 && st2 < 300 {
				return st2, body2, nil
			}
			if err == nil {
				return st, body, fmt.Errorf("probe status %d", st)
			}
			if err2 == nil {
				return st2, body2, fmt.Errorf("probe status %d", st2)
			}
		}
		return modelsStatus, modelsBody, fmt.Errorf("probe status %d", modelsStatus)
	}

	// Default: real POST /responses first, then chat/completions fallback.
	st, body, err := runResponses()
	st, body, err = with429Retry(st, body, err, runResponses)
	if err == nil && st >= 200 && st < 300 {
		return st, body, nil
	}
	needFallback := err != nil || st == 401 || st == 402 || st == 403 || st == 429
	if needFallback {
		st2, body2, err2 := runCompletions()
		st2, body2, err2 = with429Retry(st2, body2, err2, runCompletions)
		if err2 == nil && st2 >= 200 && st2 < 300 {
			return st2, body2, nil
		}
		// Prefer responses body for classify (primary path).
		if err == nil && body != "" {
			return st, body, fmt.Errorf("probe status %d", st)
		}
		if err2 == nil {
			return st2, body2, fmt.Errorf("probe status %d", st2)
		}
		if err != nil {
			return st, body, err
		}
		return st2, body2, err2
	}
	return st, body, fmt.Errorf("probe status %d", st)
}

// pickModel chooses a preferred grok model id from /models JSON body.
func pickModel(body string) string {
	var data struct {
		Data []struct {
			ID    string `json:"id"`
			Model string `json:"model"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(body), &data)
	ids := make([]string, 0, len(data.Data))
	for _, item := range data.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = strings.TrimSpace(item.Model)
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	for _, preferred := range []string{"grok-4.5", "grok-4", "grok-3-mini", "grok-3"} {
		for _, id := range ids {
			if id == preferred {
				return preferred
			}
		}
	}
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

func ExtractAccessToken(raw json.RawMessage) (string, error) {
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

func (p *Service) RememberProbeResult(authID string, ok bool, status int, errMsg string) {
	if strings.TrimSpace(authID) == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.lastByAuth == nil {
		p.lastByAuth = make(map[string]CredentialResult)
	}
	p.lastByAuth[authID] = CredentialResult{
		At:     time.Now(),
		OK:     ok,
		Status: status,
		Error:  errMsg,
	}
}

func (p *Service) LastResults() map[string]CredentialResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]CredentialResult, len(p.lastByAuth))
	for k, v := range p.lastByAuth {
		out[k] = v
	}
	return out
}

func (p *Service) recordRun(res Result, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastRun = time.Now()
	p.lastOK = res.OK
	p.lastFail = res.Failed
	p.lastErr = errMsg
	p.runSeq++
	run := Run{ID: p.runSeq, Result: res, Error: errMsg}
	p.history = append([]Run{run}, p.history...)
	if len(p.history) > maxProbeHistory {
		p.history = p.history[:maxProbeHistory]
	}
}

func (p *Service) HistorySnapshot() []Run {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Run, len(p.history))
	copy(out, p.history)
	return out
}

func (p *Service) Status() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]any{
		"enabled":      p.cfg.ProbeEnabled,
		"running":      p.running,
		"job_running":  p.jobRunning,
		"job_id":       p.jobID,
		"job_done":     p.jobDone,
		"job_total":    p.jobTotal,
		"last_run":     p.lastRun.Format(time.RFC3339),
		"last_ok":      p.lastOK,
		"last_fail":    p.lastFail,
		"last_err":     p.lastErr,
		"mode":         p.cfg.ProbeMode,
		"interval":     p.cfg.ProbeIntervalSeconds,
		"auto_execute": p.cfg.AutoExecute,
		"history":      append([]Run(nil), p.history...),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (p *Service) Attach(bans *ban.State, audit *audit.Log, persister *persist.Persister) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bans = bans
	p.audit = audit
	p.persist = persister
}
func (p *Service) configCopy() config.PluginConfig {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cfg
}
