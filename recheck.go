package main

import (
	"fmt"
	"net/http"
	"strings"
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

func indexAuthFiles(files []pluginapi.HostAuthFileEntry) map[string]pluginapi.HostAuthFileEntry {
	out := make(map[string]pluginapi.HostAuthFileEntry, len(files)*3)
	for _, f := range files {
		for _, k := range []string{f.ID, f.AuthIndex, f.Name, authKey(f)} {
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
