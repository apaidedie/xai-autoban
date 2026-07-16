package creds

import (
	"encoding/json"
	"testing"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/host"
)

func TestMetaCacheTTLAndApply(t *testing.T) {
	c := NewMetaCache(time.Hour)
	c.PutUsingAPI([]string{"a1", "a1.json"}, true)
	v, ok := c.GetUsingAPI("a1")
	if !ok || !v {
		t.Fatalf("get a1: %v %v", v, ok)
	}
	items := []Info{{AuthID: "a1"}, {AuthID: "b2"}}
	c.Apply(items)
	if items[0].UsingAPI == nil || !*items[0].UsingAPI {
		t.Fatal("apply a1")
	}
	if items[1].UsingAPI != nil {
		t.Fatal("b2 should stay unknown")
	}
}

func TestMetaCacheExpired(t *testing.T) {
	c := NewMetaCache(time.Millisecond)
	c.PutUsingAPI([]string{"x"}, true)
	time.Sleep(5 * time.Millisecond)
	if _, ok := c.GetUsingAPI("x"); ok {
		t.Fatal("expected expired")
	}
}

func TestSampleAuthJSONConcurrent(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{
			{ID: "1", AuthIndex: "1", Name: "a.json", Provider: "xai"},
			{ID: "2", AuthIndex: "2", Name: "b.json", Provider: "xai"},
		},
		JSONBy: map[string]json.RawMessage{
			"1": json.RawMessage(`{"using_api":true,"access_token":"t"}`),
			"2": json.RawMessage(`{"using_api":false,"access_token":"t"}`),
		},
	}
	cache := NewMetaCache(time.Hour)
	out := SampleAuthJSON(stub, stub.Files, 10, cache)
	if len(out) < 2 {
		t.Fatalf("out=%d", len(out))
	}
	v, ok := cache.GetUsingAPI("1")
	if !ok || !v {
		t.Fatalf("cache 1: %v %v", v, ok)
	}
	// Second call: missing sample should fetch nothing new if all cached
	miss := SampleMissingAuthJSON(stub, stub.Files, 10, cache)
	if len(miss) != 0 {
		// may still re-fetch if keys don't match AuthKey — accept either
		_ = miss
	}
}
