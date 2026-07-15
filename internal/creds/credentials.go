package creds

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/host"
	"xai-autoban/internal/tokenutil"
	"xai-autoban/internal/xai"
)

type Info struct {
	AuthID           string `json:"auth_id"`
	Name             string `json:"name,omitempty"`
	Label            string `json:"label,omitempty"`
	Email            string `json:"email,omitempty"`
	Provider         string `json:"provider,omitempty"`
	Disabled         bool   `json:"disabled"`
	Banned           bool   `json:"banned"`
	StatusCode       int    `json:"status_code"`
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
	Classification   string `json:"classification,omitempty"`
	Action           string `json:"action,omitempty"`
	BannedAt         string `json:"banned_at,omitempty"`
	ResetAt          string `json:"reset_at,omitempty"`
	RemainingSeconds int64  `json:"remaining_seconds,omitempty"`
	PendingDelete    bool   `json:"pending_delete,omitempty"`
	Source           string `json:"source,omitempty"`
	LastProbeAt      string `json:"last_probe_at,omitempty"`
	LastProbeOK      *bool  `json:"last_probe_ok,omitempty"`
	LastProbeStatus  int    `json:"last_probe_status,omitempty"`
	TokenExpired     bool   `json:"token_expired,omitempty"`
	NeedsRefresh     bool   `json:"needs_refresh,omitempty"`
	HasRefreshToken  bool   `json:"has_refresh_token,omitempty"`
}

type StatusCounts struct {
	All      int `json:"all"`
	Healthy  int `json:"healthy"`
	Banned   int `json:"banned"`
	Code401  int `json:"401"`
	Code402  int `json:"402"`
	Code403  int `json:"403"`
	Code429  int `json:"429"`
	Disabled int `json:"disabled"`
}

type ProbeResult struct {
	At     time.Time `json:"at"`
	OK     bool      `json:"ok"`
	Status int       `json:"status"`
	Error  string    `json:"error,omitempty"`
}

func Build(files []pluginapi.HostAuthFileEntry, banSnap map[string]ban.Entry, probeLast map[string]ProbeResult, now time.Time) ([]Info, StatusCounts) {
	return BuildWithJSON(files, banSnap, probeLast, nil, now)
}

// BuildWithJSON is like Build but can enrich TokenExpired/NeedsRefresh from auth JSON by id.
func BuildWithJSON(files []pluginapi.HostAuthFileEntry, banSnap map[string]ban.Entry, probeLast map[string]ProbeResult, jsonByID map[string]json.RawMessage, now time.Time) ([]Info, StatusCounts) {
	items := make([]Info, 0)
	seen := make(map[string]struct{})

	for _, f := range files {
		if !xai.IsAuth(f) {
			continue
		}
		id := xai.AuthKey(f)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		item := Info{
			AuthID:   id,
			Name:     f.Name,
			Label:    f.Label,
			Email:    strings.ToLower(strings.TrimSpace(f.Email)),
			Provider: f.Provider,
			Disabled: f.Disabled,
		}
		if item.Provider == "" {
			item.Provider = xai.Provider
		}

		if entry, ok := LookupBan(banSnap, id, f); ok {
			item.Banned = true
			item.StatusCode = entry.StatusCode
			item.Reason = entry.Reason
			item.Classification = entry.Classification
			item.Action = entry.Action
			item.Source = entry.Source
			item.PendingDelete = entry.PendingDelete
			if entry.Email != "" && item.Email == "" {
				item.Email = entry.Email
			}
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

		if pr, ok := LookupProbe(probeLast, id, f); ok {
			item.LastProbeAt = pr.At.Format(time.RFC3339)
			okCopy := pr.OK
			item.LastProbeOK = &okCopy
			item.LastProbeStatus = pr.Status
		}
		applyLocalFlags(&item)
		if raw, ok := lookupJSON(jsonByID, id, f); ok {
			applyTokenJSON(&item, raw, now)
		}

		item.Status = DeriveStatus(item)
		items = append(items, item)
	}

	// orphan bans not present in auth list (still show for ops)
	for id, entry := range banSnap {
		if _, ok := seen[id]; ok {
			continue
		}
		// skip if already represented via alias or email
		dup := false
		for sid := range seen {
			if ban.AuthIDsEqual(sid, id) {
				dup = true
				break
			}
			if entry.Email != "" && strings.EqualFold(sid, entry.Email) {
				dup = true
				break
			}
			if entry.AuthID != "" && ban.AuthIDsEqual(sid, entry.AuthID) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		// also skip if any built credential already carries this email
		if entry.Email != "" {
			for _, it := range items {
				if strings.EqualFold(it.Email, entry.Email) {
					dup = true
					break
				}
			}
		}
		if dup {
			continue
		}
		seen[id] = struct{}{}
		displayID := id
		if entry.AuthID != "" {
			displayID = entry.AuthID
		}
		item := Info{
			AuthID:           displayID,
			Name:             id,
			Email:            entry.Email,
			Provider:         xai.Provider,
			Banned:           true,
			StatusCode:       entry.StatusCode,
			Reason:           entry.Reason,
			Classification:   entry.Classification,
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
		applyLocalFlags(&item)
		item.Status = DeriveStatus(item)
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

func applyLocalFlags(item *Info) {
	if item == nil {
		return
	}
	cls := strings.ToLower(item.Classification)
	reason := strings.ToLower(item.Reason)
	if cls == "reauth" || reason == "token_expired" || reason == "unauthorized" || item.StatusCode == 401 {
		item.NeedsRefresh = true
		if reason == "token_expired" {
			item.TokenExpired = true
		}
	}
	if cls == "reauth" && reason == "token_expired" {
		item.TokenExpired = true
	}
}

func applyTokenJSON(item *Info, raw json.RawMessage, now time.Time) {
	if item == nil || len(raw) == 0 {
		return
	}
	local := tokenutil.InspectAuthJSON(raw, now)
	item.HasRefreshToken = local.HasRefreshToken
	if local.TokenExpired {
		item.TokenExpired = true
		item.NeedsRefresh = true
	} else if local.NeedsRefresh {
		item.NeedsRefresh = true
	}
}

func lookupJSON(m map[string]json.RawMessage, id string, f pluginapi.HostAuthFileEntry) (json.RawMessage, bool) {
	if m == nil {
		return nil, false
	}
	for _, k := range []string{id, f.ID, f.AuthIndex, f.Name} {
		if k == "" {
			continue
		}
		if raw, ok := m[k]; ok && len(raw) > 0 {
			return raw, true
		}
	}
	return nil, false
}

func DeriveStatus(c Info) string {
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
	// Unbanned but locally expired still surface as needs attention (401 bucket).
	if c.TokenExpired {
		return "401"
	}
	return "healthy"
}

func statusSortPriority(c Info) int {
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

func countCredentials(items []Info) StatusCounts {
	var c StatusCounts
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
		// Local expiry without ban still counts as 401 attention (matches DeriveStatus).
		if !item.Banned && !item.Disabled && item.TokenExpired {
			c.Code401++
			continue
		}
		if !item.Disabled && !item.Banned {
			c.Healthy++
		}
	}
	return c
}

func LookupBan(banSnap map[string]ban.Entry, id string, f pluginapi.HostAuthFileEntry) (ban.Entry, bool) {
	if entry, ok := banSnap[id]; ok {
		return entry, true
	}
	email := strings.ToLower(strings.TrimSpace(f.Email))
	if email != "" {
		if entry, ok := banSnap[email]; ok {
			return entry, true
		}
	}
	candidates := []string{f.ID, f.AuthIndex, f.Name, id, email}
	for key, entry := range banSnap {
		for _, c := range candidates {
			if c != "" && ban.AuthIDsEqual(key, c) {
				return entry, true
			}
		}
		if email != "" && strings.EqualFold(entry.Email, email) {
			return entry, true
		}
		if entry.AuthID != "" && ban.AuthIDsEqual(entry.AuthID, id) {
			return entry, true
		}
	}
	return ban.Entry{}, false
}

func LookupProbe(probeLast map[string]ProbeResult, id string, f pluginapi.HostAuthFileEntry) (ProbeResult, bool) {
	if probeLast == nil {
		return ProbeResult{}, false
	}
	if pr, ok := probeLast[id]; ok {
		return pr, true
	}
	candidates := []string{f.ID, f.AuthIndex, f.Name, id}
	for key, pr := range probeLast {
		for _, c := range candidates {
			if c != "" && ban.AuthIDsEqual(key, c) {
				return pr, true
			}
		}
	}
	return ProbeResult{}, false
}

func collectXAIAuthFiles(host host.Client) ([]pluginapi.HostAuthFileEntry, error) {
	if host == nil {
		return nil, nil
	}
	files, err := host.AuthList()
	if err != nil {
		return nil, err
	}
	out := make([]pluginapi.HostAuthFileEntry, 0, len(files))
	for _, f := range files {
		if xai.IsAuth(f) {
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

const (
	DefaultPageSize       = 50
	maxCredentialPageSize = 200
)

type PageQuery struct {
	Page     int
	PageSize int
	Filter   string
	Q        string
}

type PageMeta struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Total    int    `json:"total"`
	Pages    int    `json:"pages"`
	Filter   string `json:"filter"`
	Q        string `json:"q,omitempty"`
}

func ParsePageQuery(page, pageSize int, filter, q string) PageQuery {
	out := PageQuery{
		Page:     page,
		PageSize: pageSize,
		Filter:   normalizeStatusFilter(filter),
		Q:        strings.TrimSpace(q),
	}
	if out.Page < 1 {
		out.Page = 1
	}
	if out.PageSize <= 0 {
		out.PageSize = DefaultPageSize
	}
	if out.PageSize > maxCredentialPageSize {
		out.PageSize = maxCredentialPageSize
	}
	return out
}

func matchCredentialFilter(c Info, filter string) bool {
	switch normalizeStatusFilter(filter) {
	case "all":
		return true
	case "healthy":
		return !c.Disabled && !c.Banned
	case "banned":
		return c.Banned
	case "disabled":
		return c.Disabled
	case "401", "402", "403", "429":
		return c.Banned && strconv.Itoa(c.StatusCode) == filter
	default:
		return true
	}
}

func matchCredentialQuery(c Info, q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	fields := []string{c.AuthID, c.Name, c.Label, c.Email, c.Reason, c.Action, c.Status, c.Source}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func Filter(items []Info, filter, q string) []Info {
	out := make([]Info, 0, len(items))
	for _, c := range items {
		if !matchCredentialFilter(c, filter) {
			continue
		}
		if !matchCredentialQuery(c, q) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func slicePage[T any](all []T, page, pageSize int) (items []T, total, pages, pageOut int) {
	total = len(all)
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	pages = total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	pageOut = page
	if pageOut < 1 {
		pageOut = 1
	}
	if pageOut > pages {
		pageOut = pages
	}
	start := (pageOut - 1) * pageSize
	if start >= total {
		return []T{}, total, pages, pageOut
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, pages, pageOut
}

func Page(items []Info, pq PageQuery) ([]Info, PageMeta) {
	filtered := Filter(items, pq.Filter, pq.Q)
	pageItems, total, pages, pageOut := slicePage(filtered, pq.Page, pq.PageSize)
	return pageItems, PageMeta{
		Page:     pageOut,
		PageSize: pq.PageSize,
		Total:    total,
		Pages:    pages,
		Filter:   pq.Filter,
		Q:        pq.Q,
	}
}
