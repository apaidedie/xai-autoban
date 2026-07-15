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

const stateVersion = 2

type persistedState struct {
	Version  int                  `json:"version"`
	Bans     map[string]ban.Entry `json:"bans"`
	Settings map[string]any       `json:"settings,omitempty"`
}

type Persister struct {
	mu       sync.Mutex
	path     string
	timer    *time.Timer
	bans     *ban.State
	settings map[string]any
	debounce time.Duration
}

func New(path string, bans *ban.State) *Persister {
	return &Persister{path: path, bans: bans, debounce: time.Second, settings: map[string]any{}}
}

func (p *Persister) SetPath(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.path = path
}

func (p *Persister) Path() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.path
}

// Settings returns a shallow copy of last loaded/saved ops settings overlay.
func (p *Persister) Settings() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.settings) == 0 {
		return nil
	}
	out := make(map[string]any, len(p.settings))
	for k, v := range p.settings {
		out[k] = v
	}
	return out
}

// SetSettings stores ops-console settings for the next save (and in-memory).
func (p *Persister) SetSettings(s map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s == nil {
		p.settings = map[string]any{}
		return
	}
	out := make(map[string]any, len(s))
	for k, v := range s {
		// never persist secrets
		if k == "management_key" || k == "management_key_configured" {
			continue
		}
		out[k] = v
	}
	p.settings = out
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
	p.mu.Lock()
	if st.Settings != nil {
		p.settings = st.Settings
	} else {
		p.settings = map[string]any{}
	}
	p.mu.Unlock()
	slog.Info("xai-autoban: loaded state", "path", path, "bans", len(alive), "settings", len(st.Settings))
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
	settings := p.settings
	p.mu.Unlock()
	if path == "" {
		return nil
	}
	now := time.Now()
	snapshot := p.bans.Snapshot(now)
	st := persistedState{Version: stateVersion, Bans: snapshot, Settings: settings}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
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
