package xai

import (
	"path/filepath"
	"strings"

	"xai-autoban/cpasdk/pluginapi"
)

const Provider = "xai"

func IsAuth(f pluginapi.HostAuthFileEntry) bool {
	if strings.EqualFold(f.Provider, Provider) || strings.EqualFold(f.Type, Provider) {
		return true
	}
	name := strings.ToLower(f.Name)
	return strings.Contains(name, "xai") || strings.Contains(name, "grok")
}

func AuthKey(f pluginapi.HostAuthFileEntry) string {
	if f.ID != "" {
		return f.ID
	}
	if f.AuthIndex != "" {
		return f.AuthIndex
	}
	return f.Name
}

func IsCandidate(c pluginapi.SchedulerAuthCandidate) bool {
	if strings.EqualFold(c.Provider, Provider) {
		return true
	}
	if c.Attributes != nil {
		if p := c.Attributes["provider"]; strings.EqualFold(p, Provider) {
			return true
		}
		if t := c.Attributes["type"]; strings.EqualFold(t, Provider) {
			return true
		}
	}
	id := strings.ToLower(c.ID)
	return strings.Contains(id, "xai") || strings.Contains(id, "grok")
}

func CandidateEmail(c pluginapi.SchedulerAuthCandidate) string {
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
