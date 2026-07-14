package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

type credentialInfo struct {
	AuthID           string `json:"auth_id"`
	Name             string `json:"name,omitempty"`
	Label            string `json:"label,omitempty"`
	Provider         string `json:"provider,omitempty"`
	Disabled         bool   `json:"disabled"`
	Banned           bool   `json:"banned"`
	StatusCode       int    `json:"status_code"`
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
	Action           string `json:"action,omitempty"`
	BannedAt         string `json:"banned_at,omitempty"`
	ResetAt          string `json:"reset_at,omitempty"`
	RemainingSeconds int64  `json:"remaining_seconds,omitempty"`
	PendingDelete    bool   `json:"pending_delete,omitempty"`
	Source           string `json:"source,omitempty"`
	LastProbeAt      string `json:"last_probe_at,omitempty"`
	LastProbeOK      *bool  `json:"last_probe_ok,omitempty"`
	LastProbeStatus  int    `json:"last_probe_status,omitempty"`
}

type statusCounts struct {
	All      int `json:"all"`
	Healthy  int `json:"healthy"`
	Banned   int `json:"banned"`
	Code401  int `json:"401"`
	Code402  int `json:"402"`
	Code403  int `json:"403"`
	Code429  int `json:"429"`
	Disabled int `json:"disabled"`
}

type probeCredentialResult struct {
	At     time.Time `json:"at"`
	OK     bool      `json:"ok"`
	Status int       `json:"status"`
	Error  string    `json:"error,omitempty"`
}

func buildCredentials(files []pluginapi.HostAuthFileEntry, banSnap map[string]banEntry, probeLast map[string]probeCredentialResult, now time.Time) ([]credentialInfo, statusCounts) {
	items := make([]credentialInfo, 0)
	seen := make(map[string]struct{})

	for _, f := range files {
		if !isXAIAuth(f) {
			continue
		}
		id := authKey(f)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		item := credentialInfo{
			AuthID:   id,
			Name:     f.Name,
			Label:    f.Label,
			Provider: f.Provider,
			Disabled: f.Disabled,
		}
		if item.Provider == "" {
			item.Provider = providerXAI
		}

		if entry, ok := lookupBan(banSnap, id, f); ok {
			item.Banned = true
			item.StatusCode = entry.StatusCode
			item.Reason = entry.Reason
			item.Action = entry.Action
			item.Source = entry.Source
			item.PendingDelete = entry.PendingDelete
			if !entry.BannedAt.IsZero() {
				item.BannedAt = entry.BannedAt.Format(time.RFC3339)
			}
			if !entry.ResetAt.IsZero() {
				item.ResetAt = entry.ResetAt.Format(time.RFC3339)
				item.RemainingSeconds = int64(entry.ResetAt.Sub(now).Seconds())
				if item.RemainingSeconds < 0 {
					item.RemainingSeconds = 0
				}
			}
		}

		if pr, ok := lookupProbe(probeLast, id, f); ok {
			item.LastProbeAt = pr.At.Format(time.RFC3339)
			okCopy := pr.OK
			item.LastProbeOK = &okCopy
			item.LastProbeStatus = pr.Status
		}

		item.Status = deriveCredentialStatus(item)
		items = append(items, item)
	}

	// orphan bans not present in auth list (still show for ops)
	for id, entry := range banSnap {
		if _, ok := seen[id]; ok {
			continue
		}
		// skip if already represented via alias
		dup := false
		for sid := range seen {
			if authIDsEqual(sid, id) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		seen[id] = struct{}{}
		item := credentialInfo{
			AuthID:           id,
			Name:             id,
			Provider:         providerXAI,
			Banned:           true,
			StatusCode:       entry.StatusCode,
			Reason:           entry.Reason,
			Action:           entry.Action,
			Source:           entry.Source,
			PendingDelete:    entry.PendingDelete,
			RemainingSeconds: int64(entry.ResetAt.Sub(now).Seconds()),
		}
		if item.RemainingSeconds < 0 {
			item.RemainingSeconds = 0
		}
		if !entry.BannedAt.IsZero() {
			item.BannedAt = entry.BannedAt.Format(time.RFC3339)
		}
		if !entry.ResetAt.IsZero() {
			item.ResetAt = entry.ResetAt.Format(time.RFC3339)
		}
		if pr, ok := probeLast[id]; ok {
			item.LastProbeAt = pr.At.Format(time.RFC3339)
			okCopy := pr.OK
			item.LastProbeOK = &okCopy
			item.LastProbeStatus = pr.Status
		}
		item.Status = deriveCredentialStatus(item)
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		pi, pj := statusSortPriority(items[i]), statusSortPriority(items[j])
		if pi != pj {
			return pi < pj
		}
		if items[i].RemainingSeconds != items[j].RemainingSeconds {
			return items[i].RemainingSeconds < items[j].RemainingSeconds
		}
		return items[i].AuthID < items[j].AuthID
	})

	return items, countCredentials(items)
}

func deriveCredentialStatus(c credentialInfo) string {
	if c.Disabled {
		return "disabled"
	}
	if c.Banned {
		switch c.StatusCode {
		case 401, 402, 403, 429:
			return strconv.Itoa(c.StatusCode)
		default:
			return "banned"
		}
	}
	return "healthy"
}

func statusSortPriority(c credentialInfo) int {
	switch c.Status {
	case "403":
		return 1
	case "401":
		return 2
	case "402":
		return 3
	case "429":
		return 4
	case "disabled":
		return 5
	case "banned":
		return 6
	default:
		return 10
	}
}

func countCredentials(items []credentialInfo) statusCounts {
	var c statusCounts
	c.All = len(items)
	for _, item := range items {
		if item.Disabled {
			c.Disabled++
		}
		if item.Banned {
			c.Banned++
			switch item.StatusCode {
			case 401:
				c.Code401++
			case 402:
				c.Code402++
			case 403:
				c.Code403++
			case 429:
				c.Code429++
			}
		}
		if !item.Disabled && !item.Banned {
			c.Healthy++
		}
	}
	return c
}

func lookupBan(banSnap map[string]banEntry, id string, f pluginapi.HostAuthFileEntry) (banEntry, bool) {
	if entry, ok := banSnap[id]; ok {
		return entry, true
	}
	candidates := []string{f.ID, f.AuthIndex, f.Name, id}
	for key, entry := range banSnap {
		for _, c := range candidates {
			if c != "" && authIDsEqual(key, c) {
				return entry, true
			}
		}
	}
	return banEntry{}, false
}

func lookupProbe(probeLast map[string]probeCredentialResult, id string, f pluginapi.HostAuthFileEntry) (probeCredentialResult, bool) {
	if probeLast == nil {
		return probeCredentialResult{}, false
	}
	if pr, ok := probeLast[id]; ok {
		return pr, true
	}
	candidates := []string{f.ID, f.AuthIndex, f.Name, id}
	for key, pr := range probeLast {
		for _, c := range candidates {
			if c != "" && authIDsEqual(key, c) {
				return pr, true
			}
		}
	}
	return probeCredentialResult{}, false
}

func collectXAIAuthFiles(host HostClient) ([]pluginapi.HostAuthFileEntry, error) {
	if host == nil {
		return nil, nil
	}
	files, err := host.AuthList()
	if err != nil {
		return nil, err
	}
	out := make([]pluginapi.HostAuthFileEntry, 0, len(files))
	for _, f := range files {
		if isXAIAuth(f) {
			out = append(out, f)
		}
	}
	return out, nil
}

func normalizeStatusFilter(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "", "all":
		return "all"
	case "healthy", "banned", "disabled", "401", "402", "403", "429":
		return v
	default:
		return "all"
	}
}
