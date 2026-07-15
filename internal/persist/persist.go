package persist

import (
	"encoding/json"
	"log/slog"
	"os"

	"path/filepath"
	"sync"
	"time"
	"xai-autoban/internal/ban"
)

type persistedState struct {
	Version int                  `json:"version"`
	Bans    map[string]ban.Entry `json:"bans"`
}

type Persister struct {
	mu       sync.Mutex
	path     string
	timer    *time.Timer
	bans     *ban.State
	debounce time.Duration
}

func New(path string, bans *ban.State) *Persister {
	return &Persister{path: path, bans: bans, debounce: time.Second}
}

func (p *Persister) SetPath(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.path = path
}

func (p *Persister) Load() {
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
	alive := make(map[string]ban.Entry)
	for id, entry := range st.Bans {
		if entry.ResetAt.After(now) {
			alive[id] = entry
		}
	}
	p.bans.ReplaceAll(alive)
	slog.Info("xai-autoban: loaded ban state", "path", path, "count", len(alive))
}

func (p *Persister) ScheduleSave() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.path == "" {
		return
	}
	if p.timer != nil {
		p.timer.Stop()
	}
	p.timer = time.AfterFunc(p.debounce, func() {
		if err := p.SaveNow(); err != nil {
			slog.Warn("xai-autoban: failed to persist state", "error", err)
		}
	})
}

func (p *Persister) SaveNow() error {
	p.mu.Lock()
	path := p.path
	p.mu.Unlock()
	if path == "" {
		return nil
	}
	now := time.Now()
	snapshot := p.bans.Snapshot(now)
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

func (p *Persister) Flush() {
	p.mu.Lock()
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	p.mu.Unlock()
	_ = p.SaveNow()
}
