package main

import (
	"log/slog"
	"strings"
	"sync"
	"time"
)

type banEntry struct {
	StatusCode    int       `json:"status_code"`
	Reason        string    `json:"reason"`
	BannedAt      time.Time `json:"banned_at"`
	ResetAt       time.Time `json:"reset_at"`
	PendingDelete bool      `json:"pending_delete,omitempty"`
	Source        string    `json:"source,omitempty"`
	Action        string    `json:"action,omitempty"`
}

type banState struct {
	mu   sync.Mutex
	bans map[string]banEntry
}

func (s *banState) set(authID string, entry banEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bans == nil {
		s.bans = make(map[string]banEntry)
	}
	if current, ok := s.bans[authID]; ok && current.ResetAt.After(entry.ResetAt) && !entry.PendingDelete {
		return
	}
	s.bans[authID] = entry
}

func (s *banState) active(authID string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeLocked(authID, now)
}

func (s *banState) activeLocked(authID string, now time.Time) bool {
	// exact key first
	if entry, ok := s.bans[authID]; ok {
		if !now.Before(entry.ResetAt) {
			delete(s.bans, authID)
			slog.Info("xai-autoban: automatically released credential", "auth_id", authID)
			return false
		}
		return true
	}
	// fuzzy match aliases (filename vs bare id, case, etc.)
	for id, entry := range s.bans {
		if !authIDsEqual(id, authID) {
			continue
		}
		if !now.Before(entry.ResetAt) {
			delete(s.bans, id)
			slog.Info("xai-autoban: automatically released credential", "auth_id", id)
			return false
		}
		return true
	}
	return false
}

func (s *banState) isBannedCandidateID(id string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeLocked(id, now) {
		return true
	}
	for _, alias := range authIDAliases(id) {
		if alias == id {
			continue
		}
		if s.activeLocked(alias, now) {
			return true
		}
	}
	return false
}

func (s *banState) clear(authID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.bans[authID]
	delete(s.bans, authID)
	return ok
}

func (s *banState) clearAll() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.bans)
	s.bans = make(map[string]banEntry)
	return n
}

func (s *banState) clearStatus(status int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for id, entry := range s.bans {
		if entry.StatusCode == status {
			delete(s.bans, id)
			removed++
		}
	}
	return removed
}

func (s *banState) clearMany(authIDs []string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for _, id := range authIDs {
		id = strings.TrimSpace(id)
		if _, ok := s.bans[id]; ok {
			delete(s.bans, id)
			removed++
		}
	}
	return removed
}

func (s *banState) snapshot(now time.Time) map[string]banEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]banEntry)
	for id, entry := range s.bans {
		if !now.Before(entry.ResetAt) {
			delete(s.bans, id)
			continue
		}
		out[id] = entry
	}
	return out
}

func (s *banState) replaceAll(entries map[string]banEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bans = make(map[string]banEntry, len(entries))
	for id, entry := range entries {
		s.bans[id] = entry
	}
}
