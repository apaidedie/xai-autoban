package creds

import (
	"encoding/json"
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/ban"
)

func TestBuildCredentialsMergeAndCounts(t *testing.T) {
	now := time.Now()
	files := []pluginapi.HostAuthFileEntry{
		{ID: "ok-1", Name: "xai-ok", Provider: "xai", Disabled: false},
		{ID: "ban-401", Name: "xai-401", Provider: "xai"},
		{ID: "off-1", Name: "xai-off", Provider: "xai", Disabled: true},
		{ID: "other", Name: "openai-1", Provider: "openai"},
	}
	bans := map[string]ban.Entry{
		"ban-401": {
			StatusCode: 401,
			Reason:     "unauthorized",
			BannedAt:   now.Add(-time.Hour),
			ResetAt:    now.Add(2 * time.Hour),
			Action:     "ban",
			Source:     "probe",
		},
	}
	probeLast := map[string]ProbeResult{
		"ok-1":    {At: now.Add(-time.Minute), OK: true, Status: 200},
		"ban-401": {At: now.Add(-time.Minute), OK: false, Status: 401},
	}

	items, counts := Build(files, bans, probeLast, now)
	if counts.All != 3 {
		t.Fatalf("expected 3 xAI creds, got all=%d items=%d", counts.All, len(items))
	}
	if counts.Healthy != 1 || counts.Banned != 1 || counts.Disabled != 1 || counts.Code401 != 1 {
		t.Fatalf("unexpected counts: %+v", counts)
	}

	byID := map[string]Info{}
	for _, c := range items {
		byID[c.AuthID] = c
	}
	if byID["ok-1"].Status != "healthy" {
		t.Fatalf("ok-1 status=%s", byID["ok-1"].Status)
	}
	if byID["ban-401"].Status != "401" || !byID["ban-401"].Banned {
		t.Fatalf("ban-401: %+v", byID["ban-401"])
	}
	if byID["off-1"].Status != "disabled" {
		t.Fatalf("off-1 status=%s", byID["off-1"].Status)
	}
	if byID["ok-1"].LastProbeOK == nil || !*byID["ok-1"].LastProbeOK {
		t.Fatal("expected last probe ok")
	}
}

func TestBuildCredentialsIncludesOrphanBan(t *testing.T) {
	now := time.Now()
	bans := map[string]ban.Entry{
		"ghost": {StatusCode: 403, ResetAt: now.Add(time.Hour), Reason: "forbidden"},
	}
	items, counts := Build(nil, bans, nil, now)
	if counts.All != 1 || counts.Banned != 1 || counts.Code403 != 1 {
		t.Fatalf("counts=%+v", counts)
	}
	if items[0].AuthID != "ghost" || items[0].Status != "403" {
		t.Fatalf("item=%+v", items[0])
	}
}

func TestDeriveCredentialStatusPriority(t *testing.T) {
	c := Info{Disabled: true, Banned: true, StatusCode: 401}
	if DeriveStatus(c) != "disabled" {
		t.Fatal("disabled should win")
	}
}

func TestBuildFullUsingAPIAndSoft403(t *testing.T) {
	now := time.Now()
	files := []pluginapi.HostAuthFileEntry{
		{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"},
		{ID: "a2", AuthIndex: "2", Name: "xai-2", Provider: "xai"},
	}
	jsonBy := map[string]json.RawMessage{
		"a1": json.RawMessage(`{"access_token":"t","using_api":true}`),
		"a2": json.RawMessage(`{"access_token":"t","using_api":false}`),
	}
	soft := map[string]int{"a1": 2}
	items, counts := BuildFull(files, nil, nil, jsonBy, soft, 3, now)
	if len(items) != 2 {
		t.Fatalf("items=%d", len(items))
	}
	if items[0].UsingAPI == nil || !*items[0].UsingAPI {
		t.Fatalf("using_api=%v", items[0].UsingAPI)
	}
	if items[0].Soft403Streak != 2 || items[0].Soft403Need != 3 {
		t.Fatalf("streak=%d need=%d", items[0].Soft403Streak, items[0].Soft403Need)
	}
	if counts.UsingAPI != 1 {
		t.Fatalf("using_api count=%d", counts.UsingAPI)
	}
	apiOnly := Filter(items, "using_api", "")
	if len(apiOnly) != 1 || apiOnly[0].AuthID != "a1" {
		t.Fatalf("filter using_api: %+v", apiOnly)
	}
}

func TestPageCredentialsFilterAndSlice(t *testing.T) {
	items := []Info{
		{AuthID: "a", Status: "healthy"},
		{AuthID: "b", Status: "401", Banned: true, StatusCode: 401, Reason: "unauthorized"},
		{AuthID: "c", Status: "402", Banned: true, StatusCode: 402},
		{AuthID: "d", Status: "disabled", Disabled: true, Name: "xai-off"},
		{AuthID: "e", Status: "healthy", Name: "search-me"},
	}
	page, meta := Page(items, ParsePageQuery(1, 2, "401", ""))
	if meta.Total != 1 || len(page) != 1 || page[0].AuthID != "b" {
		t.Fatalf("401 page=%+v meta=%+v", page, meta)
	}
	page, meta = Page(items, ParsePageQuery(1, 2, "all", "search"))
	if meta.Total != 1 || page[0].AuthID != "e" {
		t.Fatalf("search page=%+v meta=%+v", page, meta)
	}
	page, meta = Page(items, ParsePageQuery(2, 2, "all", ""))
	if meta.Pages != 3 || meta.Page != 2 || len(page) != 2 {
		t.Fatalf("page2=%+v meta=%+v", page, meta)
	}
}
