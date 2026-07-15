package schedule

import (
	"strings"
	"sync/atomic"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/xai"
)

var rr uint64

func Pick(req pluginapi.SchedulerPickRequest, bans *ban.State, delegate string) pluginapi.SchedulerPickResponse {
	now := time.Now()
	available := make([]pluginapi.SchedulerAuthCandidate, 0, len(req.Candidates))
	filtered := 0
	for _, candidate := range req.Candidates {
		if xai.IsCandidate(candidate) && bans.IsBannedCandidate(candidate.ID, xai.CandidateEmail(candidate), now) {
			filtered++
			continue
		}
		available = append(available, candidate)
	}
	if filtered == 0 {
		return pluginapi.SchedulerPickResponse{Handled: false}
	}
	if len(available) == 0 {
		return pluginapi.SchedulerPickResponse{Handled: false}
	}
	authID := pickFromAvailable(available, delegate)
	if authID == "" {
		return pluginapi.SchedulerPickResponse{Handled: false}
	}
	return pluginapi.SchedulerPickResponse{AuthID: authID, Handled: true}
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
		n := atomic.AddUint64(&rr, 1)
		return available[int((n-1)%uint64(len(available)))].ID
	}
}
