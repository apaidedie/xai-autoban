package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

type recheck429Result struct {
	Checked   int      `json:"checked"`
	Unbanned  int      `json:"unbanned"`
	Relocked  int      `json:"relocked"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
	StartedAt string   `json:"started_at"`
	FinishedAt string  `json:"finished_at"`
}

// recheck429Bans probes currently isolated 429 credentials only.
// OK / non-429 success path => unban; still 429 => extend ban window from now.
func recheck429Bans(force bool) (recheck429Result, error) {
	now := time.Now()
	res := recheck429Result{StartedAt: now.Format(time.RFC3339)}
	if hostImpl == nil {
		return res, fmt.Errorf("host unavailable")
	}
	cfg := currentConfig()
	snap := bans.snapshot(now)
	targets := make([]struct {
		id    string
		entry banEntry
	}, 0)
	for id, entry := range snap {
		if entry.StatusCode == http.StatusTooManyRequests {
			targets = append(targets, struct {
				id    string
				entry banEntry
			}{id: id, entry: entry})
		}
	}
	if len(targets) == 0 {
		res.FinishedAt = time.Now().Format(time.RFC3339)
		return res, nil
	}

	files, err := hostImpl.AuthList()
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
		status, perr := probeSvc.probeOne(cfg, hostImpl, f)
		if perr == nil && status >= 200 && status < 300 {
			if bans.clear(t.id) {
				res.Unbanned++
				audit.add("recheck429", t.id, "unban", "ok", "429 recheck recovered", 200)
			} else {
				res.Skipped++
			}
			probeSvc.rememberProbeResult(t.id, true, status, "")
			continue
		}
		// still rate limited or other failure
		if status == http.StatusTooManyRequests || (perr != nil && status == http.StatusTooManyRequests) {
			entry := t.entry
			entry.StatusCode = http.StatusTooManyRequests
			entry.Reason = "rate_limited"
			entry.BannedAt = now
			entry.ResetAt = now.Add(cfg.durationForStatus(http.StatusTooManyRequests))
			entry.Source = "recheck429"
			entry.Action = actionBan
			if entry.Action == "" {
				entry.Action = cfg.actionForStatus(http.StatusTooManyRequests)
			}
			// force replace window even if previous reset is later
			bans.forceSet(t.id, entry)
			res.Relocked++
			audit.add("recheck429", t.id, "ban", "ok", "429 still limited; window refreshed", 429)
			probeSvc.rememberProbeResult(t.id, false, 429, fmt.Sprintf("%v", perr))
			continue
		}
		// non-429 failure: keep isolation but refresh as probe failure
		res.Failed++
		entry := t.entry
		if status > 0 {
			entry.StatusCode = status
		}
		entry.Reason = "probe_failed"
		entry.BannedAt = now
		entry.ResetAt = now.Add(cfg.durationForStatus(statusOrFallback(status, cfg)))
		entry.Source = "recheck429"
		entry.Action = actionBan
		bans.forceSet(t.id, entry)
		msg := "recheck failed"
		if perr != nil {
			msg = perr.Error()
		}
		audit.add("recheck429", t.id, "ban", "error", msg, status)
		if len(res.Errors) < 20 {
			res.Errors = append(res.Errors, t.id+": "+msg)
		}
		probeSvc.rememberProbeResult(t.id, false, status, msg)
		_ = force
	}
	persister.scheduleSave()
	res.FinishedAt = time.Now().Format(time.RFC3339)
	return res, nil
}

type recheckSelectedResult struct {
	Checked   int      `json:"checked"`
	OK        int      `json:"ok"`
	Failed    int      `json:"failed"`
	Unbanned  int      `json:"unbanned"`
	Reenabled int      `json:"reenabled"`
	Banned    int      `json:"banned"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors,omitempty"`
	StartedAt string   `json:"started_at"`
	FinishedAt string  `json:"finished_at"`
}

// recheckSelectedCredentials concurrently probes the given auth IDs.
// Includes disabled credentials (full probe skips them).
// On success: unban if isolated, reenable if disabled (when reenableOnOK).
// On classifiable failure: refresh ban ledger for visibility.
func recheckSelectedCredentials(authIDs []string, reenableOnOK bool) (recheckSelectedResult, error) {
	now := time.Now()
	res := recheckSelectedResult{StartedAt: now.Format(time.RFC3339)}
	if hostImpl == nil {
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

	cfg := currentConfig()
	files, err := hostImpl.AuthList()
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
			key := authKey(f)
			if key == "" {
				key = authID
			}
			status, perr := probeSvc.probeOne(cfg, hostImpl, f)
			mu.Lock()
			defer mu.Unlock()
			res.Checked++
			if perr == nil && status >= 200 && status < 300 {
				res.OK++
				probeSvc.rememberProbeResult(key, true, status, "")
				if bans.clear(key) || bans.clear(authID) {
					res.Unbanned++
					audit.add("recheck-selected", key, "unban", "ok", "selected recheck recovered", 200)
				}
				if reenableOnOK && f.Disabled {
					if err := engine.applyAction(key, successReenable, "recheck-selected", banEntry{AuthID: key, Email: f.Email, Source: "recheck-selected"}, true); err != nil {
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
			msg := "probe failed"
			if perr != nil {
				msg = perr.Error()
			}
			probeSvc.rememberProbeResult(key, false, status, msg)
			entry, okClass := engine.classifyFailure(status, nil, time.Now())
			if !okClass {
				entry = banEntry{
					StatusCode: statusOrFallback(status, cfg),
					Reason:     "probe_failed",
					BannedAt:   time.Now(),
					ResetAt:    time.Now().Add(cfg.durationForStatus(statusOrFallback(status, cfg))),
					Action:     actionBan,
					Source:     "recheck-selected",
					Email:      strings.ToLower(strings.TrimSpace(f.Email)),
					AuthID:     key,
				}
			} else {
				entry.Source = "recheck-selected"
				entry.Action = actionBan
				entry.Email = strings.ToLower(strings.TrimSpace(f.Email))
				entry.AuthID = key
			}
			bans.forceSet(key, entry)
			res.Banned++
			audit.add("recheck-selected", key, "ban", "ok", msg, entry.StatusCode)
			if len(res.Errors) < 30 {
				res.Errors = append(res.Errors, key+": "+msg)
			}
		}(id)
	}
	wg.Wait()
	persister.scheduleSave()
	res.FinishedAt = time.Now().Format(time.RFC3339)
	return res, nil
}

func indexAuthFiles(files []pluginapi.HostAuthFileEntry) map[string]pluginapi.HostAuthFileEntry {
	out := make(map[string]pluginapi.HostAuthFileEntry, len(files)*4)
	for _, f := range files {
		keys := []string{f.ID, f.AuthIndex, f.Name, authKey(f), f.Email}
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
		if authIDsEqual(k, id) {
			return f, true
		}
	}
	return pluginapi.HostAuthFileEntry{}, false
}
