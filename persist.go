package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type persistedState struct {
	Version int                 `json:"version"`
	Bans    map[string]banEntry `json:"bans"`
}

type statePersister struct {
	mu       sync.Mutex
	path     string
	timer    *time.Timer
	bans     *banState
	debounce time.Duration
}

func newStatePersister(path string, bans *banState) *statePersister {
	return &statePersister{path: path, bans: bans, debounce: time.Second}
}

func (p *statePersister) setPath(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.path = path
}

func (p *statePersister) load() {
	p.mu.Lock()
	path := p.path
	p.mu.Unlock()
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("xai-autoban: failed to read state file", "path", path, "error", err)
		}
		return
	}
	var st persistedState
	if err := json.Unmarshal(raw, &st); err != nil {
		bad := path + ".bad"
		_ = os.WriteFile(bad, raw, 0o600)
		slog.Warn("xai-autoban: corrupt state file moved aside", "path", path, "bad", bad, "error", err)
		return
	}
	now := time.Now()
	alive := make(map[string]banEntry)
	for id, entry := range st.Bans {
		if entry.ResetAt.After(now) {
			alive[id] = entry
		}
	}
	p.bans.replaceAll(alive)
	slog.Info("xai-autoban: loaded ban state", "path", path, "count", len(alive))
}

func (p *statePersister) scheduleSave() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.path == "" {
		return
	}
	if p.timer != nil {
		p.timer.Stop()
	}
	p.timer = time.AfterFunc(p.debounce, func() {
		if err := p.saveNow(); err != nil {
			slog.Warn("xai-autoban: failed to persist state", "error", err)
		}
	})
}

func (p *statePersister) saveNow() error {
	p.mu.Lock()
	path := p.path
	p.mu.Unlock()
	if path == "" {
		return nil
	}
	now := time.Now()
	snapshot := p.bans.snapshot(now)
	st := persistedState{Version: 1, Bans: snapshot}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." && filepath.Dir(path) != "" {
		// if path has no dir, MkdirAll may fail depending on platform; ignore when dir is empty/current
		if filepath.Dir(path) != "." {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (p *statePersister) flush() {
	p.mu.Lock()
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	p.mu.Unlock()
	_ = p.saveNow()
}
