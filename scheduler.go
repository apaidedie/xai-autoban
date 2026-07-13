package main

import (
	"encoding/json"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

func handleSchedulerPick(raw []byte) ([]byte, error) {
	var req pluginapi.SchedulerPickRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	now := time.Now()
	available := make([]pluginapi.SchedulerAuthCandidate, 0, len(req.Candidates))
	filtered := 0
	for _, candidate := range req.Candidates {
		if strings.EqualFold(candidate.Provider, providerXAI) && bans.active(candidate.ID, now) {
			filtered++
			continue
		}
		available = append(available, candidate)
	}
	if filtered == 0 || len(available) == 0 {
		return okEnvelope(pluginapi.SchedulerPickResponse{Handled: false})
	}
	delegate := currentConfig().SchedulerDelegate
	if delegate == "" {
		delegate = pluginapi.SchedulerBuiltinRoundRobin
	}
	return okEnvelope(pluginapi.SchedulerPickResponse{
		Handled:         true,
		DelegateBuiltin: delegate,
	})
}
