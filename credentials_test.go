package main

import (
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

func TestBuildCredentialsMergeAndCounts(t *testing.T) {
	now := time.Now()
	files := []pluginapi.HostAuthFileEntry{
		{ID: "ok-1", Name: "xai-ok", Provider: "xai", Disabled: false},
		{ID: "ban-401", Name: "xai-401", Provider: "xai"},
		{ID: "off-1", Name: "xai-off", Provider: "xai", Disabled: true},
		{ID: "other", Name: "openai-1", Provider: "openai"},
	}
	bans := map[string]banEntry{
		"ban-401": {
			StatusCode: 401,
			Reason:     "unauthorized",
			BannedAt:   now.Add(-time.Hour),
			ResetAt:    now.Add(2 * time.Hour),
			Action:     actionBan,
			Source:     "probe",
		},
	}
	probeLast := map[string]probeCredentialResult{
		"ok-1":    {At: now.Add(-time.Minute), OK: true, Status: 200},
		"ban-401": {At: now.Add(-time.Minute), OK: false, Status: 401},
	}

	items, counts := buildCredentials(files, bans, probeLast, now)
	if counts.All != 3 {
		t.Fatalf("expected 3 xAI creds, got all=%d items=%d", counts.All, len(items))
	}
	if counts.Healthy != 1 || counts.Banned != 1 || counts.Disabled != 1 || counts.Code401 != 1 {
		t.Fatalf("unexpected counts: %+v", counts)
	}

	byID := map[string]credentialInfo{}
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
	bans := map[string]banEntry{
		"ghost": {StatusCode: 403, ResetAt: now.Add(time.Hour), Reason: "forbidden"},
	}
	items, counts := buildCredentials(nil, bans, nil, now)
	if counts.All != 1 || counts.Banned != 1 || counts.Code403 != 1 {
		t.Fatalf("counts=%+v", counts)
	}
	if items[0].AuthID != "ghost" || items[0].Status != "403" {
		t.Fatalf("item=%+v", items[0])
	}
}

func TestDeriveCredentialStatusPriority(t *testing.T) {
	c := credentialInfo{Disabled: true, Banned: true, StatusCode: 401}
	if deriveCredentialStatus(c) != "disabled" {
		t.Fatal("disabled should win")
	}
}

func TestPageCredentialsFilterAndSlice(t *testing.T) {
	items := []credentialInfo{
		{AuthID: "a", Status: "healthy"},
		{AuthID: "b", Status: "401", Banned: true, StatusCode: 401, Reason: "unauthorized"},
		{AuthID: "c", Status: "402", Banned: true, StatusCode: 402},
		{AuthID: "d", Status: "disabled", Disabled: true, Name: "xai-off"},
		{AuthID: "e", Status: "healthy", Name: "search-me"},
	}
	page, meta := pageCredentials(items, parsePageQuery(1, 2, "401", ""))
	if meta.Total != 1 || len(page) != 1 || page[0].AuthID != "b" {
		t.Fatalf("401 page=%+v meta=%+v", page, meta)
	}
	page, meta = pageCredentials(items, parsePageQuery(1, 2, "all", "search"))
	if meta.Total != 1 || page[0].AuthID != "e" {
		t.Fatalf("search page=%+v meta=%+v", page, meta)
	}
	page, meta = pageCredentials(items, parsePageQuery(2, 2, "all", ""))
	if meta.Pages != 3 || meta.Page != 2 || len(page) != 2 {
		t.Fatalf("page2=%+v meta=%+v", page, meta)
	}
}
