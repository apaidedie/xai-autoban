package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

var schedulerRR uint64

func handleSchedulerPick(raw []byte) ([]byte, error) {
	var req pluginapi.SchedulerPickRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	now := time.Now()
	available := make([]pluginapi.SchedulerAuthCandidate, 0, len(req.Candidates))
	filtered := 0
	for _, candidate := range req.Candidates {
		if isXAICandidate(candidate) && bans.isBannedCandidate(candidate.ID, candidateEmail(candidate), now) {
			filtered++
			continue
		}
		available = append(available, candidate)
	}
	// Nothing banned: let host handle fully.
	if filtered == 0 {
		return okEnvelope(pluginapi.SchedulerPickResponse{Handled: false})
	}
	// All candidates banned: do not claim a pick; host decides failure/retry path.
	if len(available) == 0 {
		return okEnvelope(pluginapi.SchedulerPickResponse{Handled: false})
	}
	// IMPORTANT: DelegateBuiltin does not receive a filtered candidate list on many CPA builds.
	// We must return an explicit AuthID from the remaining (non-banned) candidates.
	authID := pickFromAvailable(available, currentConfig().SchedulerDelegate)
	if authID == "" {
		return okEnvelope(pluginapi.SchedulerPickResponse{Handled: false})
	}
	return okEnvelope(pluginapi.SchedulerPickResponse{
		AuthID:  authID,
		Handled: true,
	})
}

func isXAICandidate(c pluginapi.SchedulerAuthCandidate) bool {
	if strings.EqualFold(c.Provider, providerXAI) {
		return true
	}
	if c.Attributes != nil {
		if p := c.Attributes["provider"]; strings.EqualFold(p, providerXAI) {
			return true
		}
		if t := c.Attributes["type"]; strings.EqualFold(t, providerXAI) {
			return true
		}
	}
	id := strings.ToLower(c.ID)
	return strings.Contains(id, "xai") || strings.Contains(id, "grok")
}

func candidateEmail(c pluginapi.SchedulerAuthCandidate) string {
	if c.Attributes != nil {
		for _, k := range []string{"email", "Email", "account_email", "user_email"} {
			if v := strings.TrimSpace(c.Attributes[k]); v != "" {
				return strings.ToLower(v)
			}
		}
	}
	if c.Metadata != nil {
		for _, k := range []string{"email", "Email", "account_email"} {
			if v, ok := c.Metadata[k].(string); ok && strings.TrimSpace(v) != "" {
				return strings.ToLower(strings.TrimSpace(v))
			}
		}
	}
	// auth id itself may be an email or email.json
	id := strings.TrimSpace(c.ID)
	base := filepath.Base(id)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if strings.Contains(base, "@") {
		return strings.ToLower(base)
	}
	if strings.Contains(id, "@") {
		return strings.ToLower(id)
	}
	return ""
}

func pickFromAvailable(available []pluginapi.SchedulerAuthCandidate, delegate string) string {
	if len(available) == 0 {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(delegate)) {
	case pluginapi.SchedulerBuiltinFillFirst:
		chosen := available[0]
		for _, c := range available[1:] {
			if c.Priority > chosen.Priority {
				chosen = c
			}
		}
		return chosen.ID
	default:
		// round-robin across remaining candidates
		n := atomic.AddUint64(&schedulerRR, 1)
		return available[int((n-1)%uint64(len(available)))].ID
	}
}

func authIDAliases(id string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	out := []string{id}
	base := filepath.Base(id)
	if base != "" && base != id {
		out = append(out, base)
	}
	if strings.HasSuffix(strings.ToLower(base), ".json") {
		out = append(out, strings.TrimSuffix(base, filepath.Ext(base)))
	}
	// de-dup case-insensitively
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(out))
	for _, v := range out {
		k := strings.ToLower(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		uniq = append(uniq, v)
	}
	return uniq
}

func authIDsEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if strings.EqualFold(a, b) {
		return true
	}
	ab, bb := filepath.Base(a), filepath.Base(b)
	if strings.EqualFold(ab, bb) {
		return true
	}
	strip := func(s string) string {
		s = filepath.Base(s)
		if strings.HasSuffix(strings.ToLower(s), ".json") {
			s = strings.TrimSuffix(s, filepath.Ext(s))
		}
		return s
	}
	return strings.EqualFold(strip(a), strip(b))
}
