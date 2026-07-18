package mgmt

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/audit"
	"xai-autoban/internal/creds"
	"xai-autoban/internal/host"
	"xai-autoban/internal/xai"
)

type StatusInfo struct {
	Plugin      string             `json:"plugin"`
	Version     string             `json:"version"`
	Count       int                `json:"count"`
	Bans        []BanInfo          `json:"bans"`
	Credentials []creds.Info       `json:"credentials,omitempty"`
	Counts      creds.StatusCounts `json:"counts"`
	Page        creds.PageMeta     `json:"page"`
	Probe       map[string]any     `json:"probe,omitempty"`
	Settings    map[string]any     `json:"settings,omitempty"`
	Audit       []audit.Event      `json:"audit,omitempty"`
}

type BanInfo struct {
	AuthID           string `json:"auth_id"`
	Email            string `json:"email,omitempty"`
	StatusCode       int    `json:"status_code"`
	Reason           string `json:"reason"`
	Classification   string `json:"classification,omitempty"`
	BannedAt         string `json:"banned_at"`
	ResetAt          string `json:"reset_at"`
	RemainingSeconds int64  `json:"remaining_seconds"`
	PendingDelete    bool   `json:"pending_delete,omitempty"`
	Action           string `json:"action,omitempty"`
	Source           string `json:"source,omitempty"`
}

func (h *Handler) CurrentStatus() StatusInfo {
	return h.CurrentStatusPaged(nil)
}

func (h *Handler) CurrentStatusPaged(query url.Values) StatusInfo {
	now := time.Now()
	snapshot := h.Bans.Snapshot(now)
	items := make([]BanInfo, 0, len(snapshot))
	for id, entry := range snapshot {
		authID := id
		if entry.AuthID != "" {
			authID = entry.AuthID
		}
		items = append(items, BanInfo{
			AuthID:           authID,
			Email:            entry.Email,
			StatusCode:       entry.StatusCode,
			Reason:           entry.Reason,
			Classification:   entry.Classification,
			BannedAt:         entry.BannedAt.Format(time.RFC3339),
			ResetAt:          entry.ResetAt.Format(time.RFC3339),
			RemainingSeconds: int64(entry.ResetAt.Sub(now).Seconds()),
			PendingDelete:    entry.PendingDelete,
			Action:           entry.Action,
			Source:           entry.Source,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ResetAt < items[j].ResetAt })

	files, _ := collectXAIAuthFiles(h.Host)
	probeLast := map[string]creds.ProbeResult{}
	for k, v := range h.Probe.LastResults() {
		probeLast[k] = creds.ProbeResult{At: v.At, OK: v.OK, Status: v.Status, Error: v.Error}
	}
	pq := pageQueryFromValues(query)
	var soft403 map[string]int
	softNeed := 0
	if h.Engine != nil {
		soft403 = h.Engine.Soft403StreakSnapshot()
		softNeed = h.Engine.Soft403Need()
	}
	// Status page: using_api from MetaCache + optional AuthGet samples.
	// Card count and list total MUST use the same allCreds enrichment (no separate inflated cache sum).
	allCreds, counts := creds.BuildFull(files, snapshot, probeLast, nil, soft403, softNeed, now)
	if h.Meta == nil {
		h.Meta = creds.NewMetaCache(15 * time.Minute)
	}
	h.Meta.Apply(allCreds)
	// Small sample for token flags on first page only (not full fleet).
	tokenSample := 24
	if pq.Filter == "using_api" {
		// Pull all cache-miss credentials so filter list matches true fleet state.
		jsonByID := creds.SampleMissingAuthJSON(h.Host, files, 0, h.Meta)
		if len(jsonByID) > 0 {
			allCreds, counts = creds.BuildFull(files, snapshot, probeLast, jsonByID, soft403, softNeed, now)
			h.Meta.Apply(allCreds)
		}
	} else {
		more := creds.SampleAuthJSON(h.Host, files, tokenSample, h.Meta)
		if len(more) > 0 {
			allCreds, counts = creds.BuildFull(files, snapshot, probeLast, more, soft403, softNeed, now)
			h.Meta.Apply(allCreds)
		}
	}
	// Single source of truth: count using_api=true on current auth list after cache apply.
	creds.RecountUsingAPI(allCreds, &counts)
	if h.Meta.NeedsFullRefresh() {
		h.Meta.RefreshAllAsync(h.Host, files)
	}
	pageCreds, page := creds.Page(allCreds, pq)

	settings := h.Cfg().PublicView()
	if h.Persist != nil && h.Persist.Path() != "" {
		settings["state_file"] = h.Persist.Path()
		settings["state_file_resolved"] = h.Persist.Path()
	}
	st := StatusInfo{
		Plugin:      h.Name,
		Version:     h.Version,
		Count:       len(items),
		Bans:        items,
		Credentials: pageCreds,
		Counts:      counts,
		Page:        page,
		Probe:       h.Probe.Status(),
		Settings:    settings,
		Audit:       h.Audit.List(),
	}
	if h.Engine != nil {
		if st.Probe == nil {
			st.Probe = map[string]any{}
		}
		st.Probe["management"] = h.Engine.ManagementStatus()
	}
	return st
}

func pageQueryFromValues(q url.Values) creds.PageQuery {
	if q == nil {
		return creds.ParsePageQuery(1, creds.DefaultPageSize, "all", "")
	}
	page, _ := strconv.Atoi(strings.TrimSpace(q.Get("page")))
	pageSize, _ := strconv.Atoi(strings.TrimSpace(firstNonEmpty(q.Get("page_size"), q.Get("limit"))))
	filter := firstNonEmpty(q.Get("filter"), q.Get("status"))
	search := firstNonEmpty(q.Get("q"), q.Get("search"))
	return creds.ParsePageQuery(page, pageSize, filter, search)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func extractBearer(h http.Header) string {
	if h == nil {
		return ""
	}
	for _, key := range []string{"Authorization", "authorization", "X-Management-Key", "X-Api-Key"} {
		v := strings.TrimSpace(h.Get(key))
		if v == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "bearer ") {
			return strings.TrimSpace(v[7:])
		}
		return v
	}
	return ""
}

func jsonResponse(status int, value any) pluginapi.ManagementResponse {
	raw, _ := json.MarshalIndent(value, "", "  ")
	return pluginapi.ManagementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Body:       raw,
	}
}

func collectXAIAuthFiles(cli host.Client) ([]pluginapi.HostAuthFileEntry, error) {
	if cli == nil {
		return nil, nil
	}
	files, err := cli.AuthList()
	if err != nil {
		return nil, err
	}
	out := make([]pluginapi.HostAuthFileEntry, 0)
	for _, f := range files {
		if xai.IsAuth(f) {
			out = append(out, f)
		}
	}
	return out, nil
}

// sampleAuthJSON is retained for tests / callers; prefers concurrent cache-aware loader.
func sampleAuthJSON(cli host.Client, files []pluginapi.HostAuthFileEntry, limit int) map[string]json.RawMessage {
	return creds.SampleAuthJSON(cli, files, limit, nil)
}
