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
	Email         string    `json:"email,omitempty"`
	AuthID        string    `json:"auth_id,omitempty"` // original auth id when stored under email key
}

type banState struct {
	mu         sync.Mutex
	bans       map[string]banEntry
	emailIndex map[string]string // email(lower) -> storage key
	authIndex  map[string]string // auth id / aliases -> storage key
}

// banStorageKey prefers email when present so one mailbox maps to one isolation row.
func banStorageKey(email, authID string) string {
	em := strings.ToLower(strings.TrimSpace(email))
	if em != "" {
		return em
	}
	return strings.TrimSpace(authID)
}

func (s *banState) set(authID string, entry banEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setLocked(authID, entry, false)
}

// forceSet always overwrites (used by 429 recheck / import).
func (s *banState) forceSet(authID string, entry banEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setLocked(authID, entry, true)
}

func (s *banState) ensureMapsLocked() {
	if s.bans == nil {
		s.bans = make(map[string]banEntry)
	}
	if s.emailIndex == nil {
		s.emailIndex = make(map[string]string)
	}
	if s.authIndex == nil {
		s.authIndex = make(map[string]string)
	}
}

func (s *banState) setLocked(authID string, entry banEntry, force bool) {
	s.ensureMapsLocked()
	entry.Email = strings.ToLower(strings.TrimSpace(entry.Email))
	authID = strings.TrimSpace(authID)
	if entry.AuthID == "" {
		entry.AuthID = authID
	}
	key := banStorageKey(entry.Email, authID)
	if key == "" {
		return
	}
	// Collect related auth ids before collapsing.
	related := map[string]struct{}{}
	if authID != "" {
		related[authID] = struct{}{}
	}
	if entry.AuthID != "" {
		related[entry.AuthID] = struct{}{}
	}
	// Collapse legacy multi-key rows for the same email.
	if entry.Email != "" {
		for id, e := range s.bans {
			if id == key {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(e.Email), entry.Email) || strings.EqualFold(id, entry.Email) {
				if e.AuthID != "" {
					related[e.AuthID] = struct{}{}
				}
				if !strings.Contains(id, "@") {
					related[id] = struct{}{}
				}
				s.unindexLocked(id, e)
				delete(s.bans, id)
			}
		}
		// Drop legacy authID row when we now store under email.
		if authID != "" && authID != key {
			if e, ok := s.bans[authID]; ok {
				if e.AuthID != "" {
					related[e.AuthID] = struct{}{}
				}
				related[authID] = struct{}{}
				s.unindexLocked(authID, e)
				delete(s.bans, authID)
			}
		}
	}
	if !force {
		if current, ok := s.bans[key]; ok && current.ResetAt.After(entry.ResetAt) && !entry.PendingDelete {
			if entry.Email != "" {
				current.Email = entry.Email
			}
			if entry.AuthID != "" {
				current.AuthID = entry.AuthID
			}
			s.bans[key] = current
			s.indexEntryLocked(key, current, related)
			return
		}
	}
	s.bans[key] = entry
	s.indexEntryLocked(key, entry, related)
}

func (s *banState) indexEntryLocked(key string, entry banEntry, related map[string]struct{}) {
	s.ensureMapsLocked()
	if entry.Email != "" {
		s.emailIndex[entry.Email] = key
	}
	if related == nil {
		related = map[string]struct{}{}
	}
	if entry.AuthID != "" {
		related[entry.AuthID] = struct{}{}
	}
	if key != "" && !strings.Contains(key, "@") {
		related[key] = struct{}{}
	}
	for id := range related {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		s.authIndex[id] = key
		s.authIndex[strings.ToLower(id)] = key
		for _, alias := range authIDAliases(id) {
			s.authIndex[alias] = key
			s.authIndex[strings.ToLower(alias)] = key
		}
	}
}

func (s *banState) unindexLocked(key string, entry banEntry) {
	if s.emailIndex != nil && entry.Email != "" {
		if cur, ok := s.emailIndex[entry.Email]; ok && cur == key {
			delete(s.emailIndex, entry.Email)
		}
	}
	if s.authIndex != nil {
		for id, k := range s.authIndex {
			if k == key {
				delete(s.authIndex, id)
			}
		}
	}
}

func (s *banState) active(authID string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeLocked(authID, now)
}

func (s *banState) activeLocked(authID string, now time.Time) bool {
	key, entry, ok := s.lookupLocked(authID)
	if !ok {
		return false
	}
	if !now.Before(entry.ResetAt) {
		s.unindexLocked(key, entry)
		delete(s.bans, key)
		slog.Info("xai-autoban: automatically released credential", "auth_id", key)
		return false
	}
	return true
}

func (s *banState) lookupLocked(idOrEmail string) (key string, entry banEntry, ok bool) {
	idOrEmail = strings.TrimSpace(idOrEmail)
	if idOrEmail == "" {
		return "", banEntry{}, false
	}
	if e, found := s.bans[idOrEmail]; found {
		return idOrEmail, e, true
	}
	// auth reverse index
	if s.authIndex != nil {
		if k, found := s.authIndex[idOrEmail]; found {
			if e, ok2 := s.bans[k]; ok2 {
				return k, e, true
			}
		}
		if k, found := s.authIndex[strings.ToLower(idOrEmail)]; found {
			if e, ok2 := s.bans[k]; ok2 {
				return k, e, true
			}
		}
		for _, alias := range authIDAliases(idOrEmail) {
			if k, found := s.authIndex[alias]; found {
				if e, ok2 := s.bans[k]; ok2 {
					return k, e, true
				}
			}
			if k, found := s.authIndex[strings.ToLower(alias)]; found {
				if e, ok2 := s.bans[k]; ok2 {
					return k, e, true
				}
			}
		}
	}
	// email index
	em := strings.ToLower(idOrEmail)
	if s.emailIndex != nil {
		if k, found := s.emailIndex[em]; found {
			if e, ok2 := s.bans[k]; ok2 {
				return k, e, true
			}
		}
	}
	// fuzzy scan fallback
	for id, e := range s.bans {
		if authIDsEqual(id, idOrEmail) {
			return id, e, true
		}
		if e.AuthID != "" && authIDsEqual(e.AuthID, idOrEmail) {
			return id, e, true
		}
		if e.Email != "" && strings.EqualFold(e.Email, idOrEmail) {
			return id, e, true
		}
	}
	return "", banEntry{}, false
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

// isBannedCandidate checks auth id aliases and optional email from host file.
func (s *banState) isBannedCandidate(id, email string, now time.Time) bool {
	if s.isBannedCandidateID(id, now) {
		return true
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	return s.active(email, now)
}

func (s *banState) clear(authID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, entry, ok := s.lookupLocked(authID)
	if !ok {
		// still try exact delete for legacy
		if e, ok2 := s.bans[authID]; ok2 {
			s.unindexLocked(authID, e)
			delete(s.bans, authID)
			return true
		}
		return false
	}
	s.unindexLocked(key, entry)
	delete(s.bans, key)
	// also drop any remaining alias rows for same email
	if entry.Email != "" {
		for id, e := range s.bans {
			if strings.EqualFold(e.Email, entry.Email) || strings.EqualFold(id, entry.Email) {
				s.unindexLocked(id, e)
				delete(s.bans, id)
			}
		}
	}
	return true
}

func (s *banState) clearAll() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.bans)
	s.bans = make(map[string]banEntry)
	s.emailIndex = make(map[string]string)
	s.authIndex = make(map[string]string)
	return n
}

func (s *banState) clearStatus(status int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for id, entry := range s.bans {
		if entry.StatusCode == status {
			s.unindexLocked(id, entry)
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
		key, entry, ok := s.lookupLocked(id)
		if !ok {
			continue
		}
		s.unindexLocked(key, entry)
		delete(s.bans, key)
		removed++
	}
	return removed
}

func (s *banState) snapshot(now time.Time) map[string]banEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]banEntry)
	for id, entry := range s.bans {
		if !now.Before(entry.ResetAt) {
			s.unindexLocked(id, entry)
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
	s.emailIndex = make(map[string]string)
	s.authIndex = make(map[string]string)
	for id, entry := range entries {
		entry.Email = strings.ToLower(strings.TrimSpace(entry.Email))
		if entry.AuthID == "" {
			entry.AuthID = id
		}
		key := banStorageKey(entry.Email, id)
		if key == "" {
			continue
		}
		s.bans[key] = entry
		related := map[string]struct{}{id: {}, entry.AuthID: {}}
		s.indexEntryLocked(key, entry, related)
	}
}
