package creds

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/host"
	"xai-autoban/internal/xai"
)

// MetaCache remembers using_api flags to avoid AuthGet storms on large fleets.
type MetaCache struct {
	mu         sync.Mutex
	ttl        time.Duration
	m          map[string]metaEnt
	refreshing bool
	lastFull   time.Time
}

type metaEnt struct {
	UsingAPI bool
	At       time.Time
}

func NewMetaCache(ttl time.Duration) *MetaCache {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &MetaCache{ttl: ttl, m: make(map[string]metaEnt)}
}

func (c *MetaCache) PutUsingAPI(keys []string, usingAPI bool) {
	if c == nil {
		return
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = make(map[string]metaEnt)
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		c.m[k] = metaEnt{UsingAPI: usingAPI, At: now}
	}
}

func (c *MetaCache) Invalidate(keys ...string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.m, strings.TrimSpace(k))
	}
}

func (c *MetaCache) getLocked(key string, now time.Time) (bool, bool) {
	e, ok := c.m[key]
	if !ok {
		return false, false
	}
	if now.Sub(e.At) > c.ttl {
		delete(c.m, key)
		return false, false
	}
	return e.UsingAPI, true
}

// GetUsingAPI returns (value, ok) if a non-expired cache entry exists.
func (c *MetaCache) GetUsingAPI(key string) (bool, bool) {
	if c == nil {
		return false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getLocked(strings.TrimSpace(key), time.Now())
}

// Apply fills Info.UsingAPI from cache when not already set.
func (c *MetaCache) Apply(items []Info) {
	if c == nil || len(items) == 0 {
		return
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range items {
		if items[i].UsingAPI != nil {
			continue
		}
		// Match any key we store (id / name / email / basename).
		keys := []string{
			items[i].AuthID,
			items[i].Name,
			items[i].Email,
			strings.TrimSuffix(items[i].Name, ".json"),
			strings.TrimSuffix(items[i].AuthID, ".json"),
		}
		for _, k := range keys {
			if k == "" {
				continue
			}
			if v, ok := c.getLocked(k, now); ok {
				b := v
				items[i].UsingAPI = &b
				break
			}
			if v, ok := c.getLocked(strings.ToLower(k), now); ok {
				b := v
				items[i].UsingAPI = &b
				break
			}
		}
	}
}

// PutFromJSON stores using_api parsed from credential JSON under several keys.
func (c *MetaCache) PutFromJSON(keys []string, raw json.RawMessage) {
	if c == nil || len(raw) == 0 {
		return
	}
	v, ok := parseUsingAPIFlag(raw)
	if !ok {
		return
	}
	c.PutUsingAPI(keys, v)
}

// SampleAuthJSON loads credential JSON with concurrent AuthGet, preferring to skip
// keys that already have a fresh using_api cache entry when onlyNeedMeta is true.
// When onlyNeedMeta is false (default status path), still fetches until limit for token flags.
func SampleAuthJSON(cli host.Client, files []pluginapi.HostAuthFileEntry, limit int, cache *MetaCache) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	if cli == nil || limit <= 0 || len(files) == 0 {
		return out
	}

	type job struct {
		f     pluginapi.HostAuthFileEntry
		index string
	}
	jobs := make([]job, 0, minInt(limit, len(files)))
	for _, f := range files {
		if len(jobs) >= limit {
			break
		}
		index := f.AuthIndex
		if index == "" {
			index = f.Name
		}
		if index == "" {
			continue
		}
		jobs = append(jobs, job{f: f, index: index})
	}
	if len(jobs) == 0 {
		return out
	}

	workers := 8
	if workers > len(jobs) {
		workers = len(jobs)
	}
	ch := make(chan job, len(jobs))
	for _, j := range jobs {
		ch <- j
	}
	close(ch)

	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				// Skip AuthGet if cache already has using_api and we already have enough raw samples.
				// Still fetch when not cached so token flags + using_api stay accurate.
				got, err := cli.AuthGet(j.index)
				if err != nil || len(got.JSON) == 0 {
					continue
				}
				keys := []string{xai.AuthKey(j.f), j.f.ID, j.f.AuthIndex, j.f.Name, strings.ToLower(strings.TrimSpace(j.f.Email))}
				if cache != nil {
					cache.PutFromJSON(keys, got.JSON)
				}
				mu.Lock()
				for _, k := range keys {
					if k != "" {
						out[k] = got.JSON
					}
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return out
}

// SampleMissingAuthJSON only fetches files whose using_api is not in cache (faster refresh).
func SampleMissingAuthJSON(cli host.Client, files []pluginapi.HostAuthFileEntry, limit int, cache *MetaCache) map[string]json.RawMessage {
	if cache == nil {
		return SampleAuthJSON(cli, files, limit, nil)
	}
	missing := make([]pluginapi.HostAuthFileEntry, 0, len(files))
	for _, f := range files {
		key := xai.AuthKey(f)
		if key == "" {
			key = f.ID
		}
		if _, ok := cache.GetUsingAPI(key); ok {
			continue
		}
		if f.AuthIndex != "" {
			if _, ok := cache.GetUsingAPI(f.AuthIndex); ok {
				continue
			}
		}
		missing = append(missing, f)
		if limit > 0 && len(missing) >= limit {
			break
		}
	}
	if len(missing) == 0 {
		return map[string]json.RawMessage{}
	}
	capN := limit
	if capN <= 0 {
		capN = len(missing)
	}
	return SampleAuthJSON(cli, missing, capN, cache)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RefreshAllAsync scans all files in background (at most one scan at a time).
func (c *MetaCache) RefreshAllAsync(cli host.Client, files []pluginapi.HostAuthFileEntry) {
	if c == nil || cli == nil || len(files) == 0 {
		return
	}
	c.mu.Lock()
	if c.refreshing {
		c.mu.Unlock()
		return
	}
	c.refreshing = true
	c.mu.Unlock()
	go func() {
		defer func() {
			c.mu.Lock()
			c.refreshing = false
			c.lastFull = time.Now()
			c.mu.Unlock()
		}()
		_ = SampleAuthJSON(cli, files, len(files), c)
	}()
}

// NeedsFullRefresh reports whether a background full scan is due.
func (c *MetaCache) NeedsFullRefresh() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refreshing {
		return false
	}
	if c.lastFull.IsZero() {
		return true
	}
	return time.Since(c.lastFull) > c.ttl
}

// CountUsingAPI returns raw cache true-entries (aliases inflate; prefer RecountUsingAPI on fleet list).
// Kept for tests/debug only.
func (c *MetaCache) CountUsingAPI() int {
	if c == nil {
		return 0
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	seen := map[string]struct{}{}
	for k, e := range c.m {
		if now.Sub(e.At) > c.ttl {
			continue
		}
		if !e.UsingAPI {
			continue
		}
		// de-dupe rough: prefer keys without .json suffix as primary
		base := strings.TrimSuffix(k, ".json")
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		n++
	}
	return n
}

// CountUsingAPIAmong counts using_api=true for the current auth file list only (1 per file).
func (c *MetaCache) CountUsingAPIAmong(files []pluginapi.HostAuthFileEntry) int {
	if c == nil || len(files) == 0 {
		return 0
	}
	n := 0
	for _, f := range files {
		keys := []string{
			xai.AuthKey(f),
			f.ID,
			f.AuthIndex,
			f.Name,
			strings.ToLower(strings.TrimSpace(f.Email)),
			strings.TrimSuffix(f.Name, ".json"),
		}
		found := false
		for _, k := range keys {
			if k == "" {
				continue
			}
			if v, ok := c.GetUsingAPI(k); ok {
				if v {
					n++
				}
				found = true
				break
			}
		}
		_ = found
	}
	return n
}

// RecountUsingAPI updates StatusCounts.UsingAPI after cache/JSON enrichment.
func RecountUsingAPI(items []Info, c *StatusCounts) {
	if c == nil {
		return
	}
	n := 0
	for _, it := range items {
		if it.UsingAPI != nil && *it.UsingAPI {
			n++
		}
	}
	c.UsingAPI = n
}
