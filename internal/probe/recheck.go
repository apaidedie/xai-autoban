package probe

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/action"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/xai"
)

type Recheck429Result struct {
	Checked    int      `json:"checked"`
	Unbanned   int      `json:"unbanned"`
	Relocked   int      `json:"relocked"`
	Skipped    int      `json:"skipped"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors,omitempty"`
	StartedAt  string   `json:"started_at"`
	FinishedAt string   `json:"finished_at"`
}

// recheck429Bans probes currently isolated 429 credentials only.
// OK / non-429 success path => unban; still 429 => extend ban window from now.
func (p *Service) Recheck429(force bool) (Recheck429Result, error) {
	now := time.Now()
	res := Recheck429Result{StartedAt: now.Format(time.RFC3339)}
	if p.host == nil {
		return res, fmt.Errorf("host unavailable")
	}
	cfg := p.configCopy()
	snap := p.bans.Snapshot(now)
	targets := make([]struct {
		id    string
		entry ban.Entry
	}, 0)
	for id, entry := range snap {
		if entry.StatusCode == http.StatusTooManyRequests {
			targets = append(targets, struct {
				id    string
				entry ban.Entry
			}{id: id, entry: entry})
		}
	}
	if len(targets) == 0 {
		res.FinishedAt = time.Now().Format(time.RFC3339)
		return res, nil
	}

	files, err := p.host.AuthList()
	if err != nil {
		return res, err
	}
	byKey := indexAuthFiles(files)

	for _, t := range targets {
		res.Checked++
		f, ok := resolveAuthFile(byKey, t.id)
		if !ok {
			res.Skipped++
			res.Errors = append(res.Errors, t.id+": credential not found")
			continue
		}
		status, body, perr := p.ProbeOne(cfg, p.host, f)
		if perr == nil && status >= 200 && status < 300 {
			if p.bans.Clear(t.id) {
				res.Unbanned++
				p.audit.Add("recheck429", t.id, "unban", "ok", "429 recheck recovered", 200)
			} else {
				res.Skipped++
			}
			p.RememberProbeResult(t.id, true, status, "")
			continue
		}
		// still rate limited or other failure
		if status == http.StatusTooManyRequests || (perr != nil && status == http.StatusTooManyRequests) {
			entry := t.entry
			if classified, ok := p.engine.ClassifyFailureWithBody(status, nil, body, now); ok {
				entry = classified
			} else {
				entry.StatusCode = http.StatusTooManyRequests
				entry.Reason = "rate_limited"
				entry.ResetAt = now.Add(cfg.DurationForStatus(http.StatusTooManyRequests))
			}
			entry.BannedAt = now
			entry.Source = "recheck429"
			entry.Action = action.Ban
			// force replace window even if previous reset is later
			p.bans.ForceSet(t.id, entry)
			res.Relocked++
			p.audit.Add("recheck429", t.id, "ban", "ok", "429 still limited; window refreshed", 429)
			p.RememberProbeResult(t.id, false, 429, fmt.Sprintf("%v", perr))
			continue
		}
		// non-429 failure: classify body when possible, else keep isolation
		res.Failed++
		entry := t.entry
		if classified, ok := p.engine.ClassifyFailureWithBody(status, nil, body, now); ok {
			entry = classified
		} else {
			if status > 0 {
				entry.StatusCode = status
			}
			entry.Reason = "probe_failed"
			entry.ResetAt = now.Add(cfg.DurationForStatus(statusOrFallback(status, cfg)))
		}
		entry.BannedAt = now
		entry.Source = "recheck429"
		entry.Action = action.Ban
		p.bans.ForceSet(t.id, entry)
		msg := "recheck failed"
		if perr != nil {
			msg = perr.Error()
		}
		p.audit.Add("recheck429", t.id, "ban", "error", msg, entry.StatusCode)
		if len(res.Errors) < 20 {
			res.Errors = append(res.Errors, t.id+": "+msg)
		}
		p.RememberProbeResult(t.id, false, status, msg)
		_ = force
	}
	p.persist.ScheduleSave()
	res.FinishedAt = time.Now().Format(time.RFC3339)
	return res, nil
}

type RecheckSelectedResult struct {
	Checked    int      `json:"checked"`
	OK         int      `json:"ok"`
	Failed     int      `json:"failed"`
	Unbanned   int      `json:"unbanned"`
	Reenabled  int      `json:"reenabled"`
	Banned     int      `json:"banned"`
	Disabled   int      `json:"disabled"`
	Deleted    int      `json:"deleted"`
	Skipped    int      `json:"skipped"`
	Errors     []string `json:"errors,omitempty"`
	StartedAt  string   `json:"started_at"`
	FinishedAt string   `json:"finished_at"`
}

// recheckSelectedCredentials concurrently probes the given auth IDs.
// Includes disabled credentials (full probe skips them).
// On success: unban if isolated, reenable if disabled (when reenableOnOK).
// On classifiable failure: apply action_on_401/402/403/429 (same as usage/full probe when auto_execute).
func (p *Service) RecheckSelected(authIDs []string, reenableOnOK bool) (RecheckSelectedResult, error) {
	now := time.Now()
	res := RecheckSelectedResult{StartedAt: now.Format(time.RFC3339)}
	if p.host == nil {
		return res, fmt.Errorf("host unavailable")
	}
	// de-dup
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(authIDs))
	for _, id := range authIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		res.FinishedAt = time.Now().Format(time.RFC3339)
		return res, fmt.Errorf("missing_auth_ids")
	}
	// safety cap
	const maxSelected = 200
	if len(ids) > maxSelected {
		ids = ids[:maxSelected]
	}

	cfg := p.configCopy()
	files, err := p.host.AuthList()
	if err != nil {
		return res, err
	}
	byKey := indexAuthFiles(files)

	sem := make(chan struct{}, max(1, cfg.ProbeConcurrency))
	var wg sync.WaitGroup
	var mu sync.Mutex
	minInterval := time.Duration(0)
	if cfg.ProbeQPS > 0 {
		minInterval = time.Duration(float64(time.Second) / cfg.ProbeQPS)
	}
	var lastStart time.Time
	triedUsingAPI := map[string]struct{}{}

	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(authID string) {
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

			f, ok := resolveAuthFile(byKey, authID)
			if !ok {
				mu.Lock()
				res.Skipped++
				if len(res.Errors) < 30 {
					res.Errors = append(res.Errors, authID+": credential not found")
				}
				mu.Unlock()
				return
			}
			key := xai.AuthKey(f)
			if key == "" {
				key = authID
			}
			status, body, perr := p.ProbeOne(cfg, p.host, f)
			// Auto using_api: gated by config; once per key per recheck run.
			if perr != nil && p.engine != nil {
				idx := f.AuthIndex
				if idx == "" {
					idx = f.Name
				}
				if got, gerr := p.host.AuthGet(idx); gerr == nil {
					doHeal := false
					if mat, merr := parseAuthMaterial(got.JSON); merr == nil {
						mu.Lock()
						_, tried := triedUsingAPI[key]
						if ShouldAutoUsingAPI(cfg, status, mat, tried) {
							triedUsingAPI[key] = struct{}{}
							doHeal = true
						}
						mu.Unlock()
					}
					if doHeal {
						if healed, st2, body2, err2 := p.tryEnableUsingAPIAndReprobe(cfg, p.host, f, key, got.JSON); healed {
							status, body, perr = st2, body2, err2
						}
					}
				}
			}
			mu.Lock()
			defer mu.Unlock()
			res.Checked++
			if perr == nil && status >= 200 && status < 300 {
				res.OK++
				p.RememberProbeResult(key, true, status, "")
				if p.bans.Clear(key) || p.bans.Clear(authID) {
					res.Unbanned++
					p.audit.Add("recheck-selected", key, "unban", "ok", "selected recheck recovered", 200)
				}
				if reenableOnOK && f.Disabled {
					if err := p.engine.ApplyAction(key, action.SuccessReenable, "recheck-selected", ban.Entry{AuthID: key, Email: f.Email, Source: "recheck-selected"}, true); err != nil {
						if len(res.Errors) < 30 {
							res.Errors = append(res.Errors, key+": reenable "+err.Error())
						}
					} else {
						res.Reenabled++
					}
				}
				return
			}
			res.Failed++
			msg := "探测失败"
			if status > 0 {
				msg = fmt.Sprintf("%d", status)
			} else if perr != nil {
				msg = shortErr(perr.Error())
			}
			p.RememberProbeResult(key, false, status, msg)
			entry, okClass := p.engine.ClassifyFailureWithBody(status, nil, body, time.Now())
			if !okClass {
				if len(res.Errors) < 12 {
					res.Errors = append(res.Errors, shortCredLabel(key, f.Email)+" · "+msg+" · 不隔离")
				}
				return
			}
			// Same status-code actions as full probe / usage (soft-403 streak + usage grace still apply).
			entry.Source = "recheck-selected"
			entry.Email = strings.ToLower(strings.TrimSpace(f.Email))
			entry.AuthID = key
			cfg := p.configCopy()
			if !cfg.AutoExecute {
				// Report-only: isolate only.
				entry.Action = action.Ban
			} else if entry.Action == "" {
				entry.Action = cfg.ActionForStatus(entry.StatusCode)
			}
			// Optional probe_action override only when classifier left bare ban (same as full probe).
			if cfg.AutoExecute && cfg.ProbeAction != "" && entry.Action == action.Ban && cfg.ProbeAction != action.Ban {
				if entry.Classification != "rate_limited" {
					entry.Action = cfg.ProbeAction
				}
			}
			act := entry.Action
			wasBanned := p.bans.Active(key, time.Now()) || p.bans.IsBannedCandidate(key, entry.Email, time.Now())
			_ = p.engine.ApplyFailure(key, "recheck-selected", entry, false)
			nowBanned := p.bans.Active(key, time.Now()) || p.bans.IsBannedCandidate(key, entry.Email, time.Now())
			switch act {
			case action.Disable:
				res.Disabled++
				res.Banned++
			case action.Delete:
				res.Deleted++
				res.Banned++
			default:
				if nowBanned && !wasBanned {
					res.Banned++
				}
			}
			if len(res.Errors) < 12 {
				label := shortCredLabel(key, f.Email)
				actLabel := map[string]string{
					action.Ban: "已隔离", action.Disable: "已禁用", action.Delete: "已删除",
				}[act]
				if actLabel == "" {
					actLabel = act
				}
				if act == action.Ban && !(nowBanned && !wasBanned) {
					res.Errors = append(res.Errors, label+" · "+msg+" · 连击/跳过")
				} else {
					res.Errors = append(res.Errors, label+" · "+msg+" · "+actLabel)
				}
			}
		}(id)
	}
	wg.Wait()
	p.persist.ScheduleSave()
	res.FinishedAt = time.Now().Format(time.RFC3339)
	return res, nil
}

func shortCredLabel(key, email string) string {
	em := strings.ToLower(strings.TrimSpace(email))
	if em != "" {
		return em
	}
	k := strings.TrimSpace(key)
	if i := strings.LastIndexAny(k, `/\`); i >= 0 {
		k = k[i+1:]
	}
	k = strings.TrimSuffix(k, ".json")
	if len(k) > 40 {
		return k[:18] + "…" + k[len(k)-14:]
	}
	return k
}

func shortErr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "失败"
	}
	// Drop long probe status prefixes; keep last meaningful bit short.
	if i := strings.LastIndex(s, ": "); i >= 0 && i+2 < len(s) {
		s = strings.TrimSpace(s[i+2:])
	}
	if len(s) > 48 {
		return s[:48] + "…"
	}
	return s
}

func indexAuthFiles(files []pluginapi.HostAuthFileEntry) map[string]pluginapi.HostAuthFileEntry {
	out := make(map[string]pluginapi.HostAuthFileEntry, len(files)*4)
	for _, f := range files {
		keys := []string{f.ID, f.AuthIndex, f.Name, xai.AuthKey(f), f.Email}
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = f
			out[strings.ToLower(k)] = f
		}
	}
	return out
}

func resolveAuthFile(byKey map[string]pluginapi.HostAuthFileEntry, id string) (pluginapi.HostAuthFileEntry, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return pluginapi.HostAuthFileEntry{}, false
	}
	if f, ok := byKey[id]; ok {
		return f, true
	}
	if f, ok := byKey[strings.ToLower(id)]; ok {
		return f, true
	}
	for k, f := range byKey {
		if ban.AuthIDsEqual(k, id) {
			return f, true
		}
	}
	return pluginapi.HostAuthFileEntry{}, false
}
