package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

const (
	disableViaHostAuth       = "host_auth"
	disableViaManagementAPI  = "management_api"
	defaultManagementURL     = "http://127.0.0.1:8317"
	defaultManagementKeyEnv  = "CPA_MANAGEMENT_KEY"
	defaultMgmtTimeoutSec    = 10
	defaultMgmtAuthCooldownSec = 600
)

var errManagementKeyMissing = errors.New("management key missing (set management_key or management_key_env)")

type managementHTTPError struct {
	StatusCode int
	Body       string
}

func (e *managementHTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("management api HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("management api HTTP %d: %s", e.StatusCode, e.Body)
}

type managementAuthFile struct {
	ID        string `json:"id"`
	AuthIndex string `json:"auth_index"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Type      string `json:"type"`
	Disabled  bool   `json:"disabled"`
}

// managementDisabler talks to CPA Management API to set auth disabled flag.
// Inspired by vrxiaojie/xai-autoban (async worker simplified to sync + auth cooldown).
type managementDisabler struct {
	mu           sync.Mutex
	cfg          PluginConfig
	host         HostClient
	blockedUntil time.Time
	lastError    string
}

func newManagementDisabler(cfg PluginConfig, host HostClient) *managementDisabler {
	return &managementDisabler{cfg: cfg, host: host}
}

func (m *managementDisabler) updateConfig(cfg PluginConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
}

func (m *managementDisabler) status() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]any{
		"disable_via":     m.cfg.DisableVia,
		"management_url":  m.cfg.ManagementURL,
		"has_key":         strings.TrimSpace(m.resolveKeyLocked()) != "",
		"last_error":      m.lastError,
		"blocked_until":   "",
		"blocked":         false,
	}
	if !m.blockedUntil.IsZero() {
		out["blocked_until"] = m.blockedUntil.Format(time.RFC3339)
		out["blocked"] = time.Now().Before(m.blockedUntil)
	}
	return out
}

func (m *managementDisabler) resolveKeyLocked() string {
	if k := strings.TrimSpace(m.cfg.ManagementKey); k != "" {
		return k
	}
	envName := strings.TrimSpace(m.cfg.ManagementKeyEnv)
	if envName == "" {
		envName = defaultManagementKeyEnv
	}
	return strings.TrimSpace(os.Getenv(envName))
}

func (m *managementDisabler) setAuthDisabled(authID, authIndex string, disabled bool) error {
	m.mu.Lock()
	cfg := m.cfg
	blocked := m.blockedUntil
	host := m.host
	m.mu.Unlock()

	if time.Now().Before(blocked) {
		return fmt.Errorf("management api cooling down until %s: %s", blocked.Format(time.RFC3339), m.lastError)
	}
	if host == nil {
		return fmt.Errorf("host unavailable")
	}
	key := m.resolveKey()
	if key == "" {
		return errManagementKeyMissing
	}

	err := m.patchAuthStatus(host, cfg, key, authID, disabled)
	if err == nil {
		m.mu.Lock()
		m.lastError = ""
		m.mu.Unlock()
		return nil
	}

	var httpErr *managementHTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound && strings.TrimSpace(authIndex) != "" {
		if file, found, listErr := m.findAuthFile(host, cfg, key, authID, authIndex); listErr == nil && found {
			name := strings.TrimSpace(file.Name)
			if name == "" {
				name = file.ID
			}
			if name != "" && name != authID {
				if err2 := m.patchAuthStatus(host, cfg, key, name, disabled); err2 == nil {
					m.mu.Lock()
					m.lastError = ""
					m.mu.Unlock()
					return nil
				} else {
					err = err2
				}
			}
		}
	}

	// verify actual state on non-auth errors
	if !isManagementAuthError(err) {
		if file, found, verifyErr := m.findAuthFile(host, cfg, key, authID, authIndex); verifyErr == nil && found && file.Disabled == disabled {
			m.mu.Lock()
			m.lastError = ""
			m.mu.Unlock()
			return nil
		}
	}

	m.mu.Lock()
	m.lastError = err.Error()
	if isManagementAuthError(err) {
		sec := cfg.ManagementAuthFailureCooldownSeconds
		if sec <= 0 {
			sec = defaultMgmtAuthCooldownSec
		}
		m.blockedUntil = time.Now().Add(time.Duration(sec) * time.Second)
	}
	m.mu.Unlock()
	return err
}

func (m *managementDisabler) resolveKey() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resolveKeyLocked()
}

func (m *managementDisabler) baseURL(cfg PluginConfig) string {
	base := strings.TrimRight(strings.TrimSpace(cfg.ManagementURL), "/")
	if base == "" {
		base = defaultManagementURL
	}
	if !strings.HasSuffix(base, "/v0/management") {
		base += "/v0/management"
	}
	return base
}

func (m *managementDisabler) patchAuthStatus(host HostClient, cfg PluginConfig, key, name string, disabled bool) error {
	body, err := json.Marshal(map[string]any{"name": name, "disabled": disabled})
	if err != nil {
		return err
	}
	resp, err := host.HTTPDo(pluginapi.HTTPRequest{
		Method: http.MethodPatch,
		URL:    m.baseURL(cfg) + "/auth-files/status",
		Headers: http.Header{
			"Authorization": {"Bearer " + key},
			"Content-Type":  {"application/json"},
			"Accept":        {"application/json"},
		},
		Body: body,
	})
	if err != nil {
		return fmt.Errorf("management api call failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &managementHTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(resp.Body))}
	}
	return nil
}

func (m *managementDisabler) findAuthFile(host HostClient, cfg PluginConfig, key, authID, authIndex string) (managementAuthFile, bool, error) {
	resp, err := host.HTTPDo(pluginapi.HTTPRequest{
		Method: http.MethodGet,
		URL:    m.baseURL(cfg) + "/auth-files",
		Headers: http.Header{
			"Authorization": {"Bearer " + key},
			"Accept":        {"application/json"},
		},
	})
	if err != nil {
		return managementAuthFile{}, false, fmt.Errorf("list auth-files failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return managementAuthFile{}, false, &managementHTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(resp.Body))}
	}
	var payload struct {
		Files []managementAuthFile `json:"files"`
	}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return managementAuthFile{}, false, fmt.Errorf("decode auth-files: %w", err)
	}
	authID = strings.TrimSpace(authID)
	authIndex = strings.TrimSpace(authIndex)
	for _, file := range payload.Files {
		if strings.TrimSpace(file.ID) == authID || (authIndex != "" && strings.TrimSpace(file.AuthIndex) == authIndex) {
			return file, true, nil
		}
		if strings.TrimSpace(file.Name) == authID {
			return file, true, nil
		}
	}
	return managementAuthFile{}, false, nil
}

func isManagementAuthError(err error) bool {
	if errors.Is(err, errManagementKeyMissing) {
		return true
	}
	var httpErr *managementHTTPError
	return errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode == http.StatusForbidden)
}
