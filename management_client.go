package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xai-autoban/cpasdk/pluginapi"
)

const (
	disableViaHostAuth         = "host_auth"
	disableViaManagementAPI    = "management_api"
	defaultManagementURL       = "http://127.0.0.1:8317"
	defaultManagementKeyEnv    = "CPA_MANAGEMENT_KEY"
	defaultMgmtTimeoutSec      = 10
	defaultMgmtAuthCooldownSec = 600
)

var errManagementKeyMissing = errors.New("management key missing (set management_key or management_key_env)")

// directMgmtTransport never uses HTTP(S)_PROXY / CPA global proxy.
// host.HTTPDo routes through NewProxyAwareHTTPClient, which forces cfg.ProxyURL
// and can return proxy 403 "client_connect_invalid_ip" when calling localhost.
var directMgmtTransport = &http.Transport{
	Proxy: nil,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          32,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

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

// managementHTTPDoer performs one Management API HTTP call.
// Tests inject a stub; production uses direct (no-proxy) net/http.
type managementHTTPDoer func(req pluginapi.HTTPRequest, timeoutSec int) (pluginapi.HTTPResponse, error)

// managementDisabler talks to CPA Management API to set auth disabled flag.
// Inspired by vrxiaojie/xai-autoban (async worker simplified to sync + auth cooldown).
type managementDisabler struct {
	mu           sync.Mutex
	cfg          PluginConfig
	host         HostClient
	blockedUntil time.Time
	lastError    string
	// httpDo: nil → directManagementHTTP (bypass CPA proxy). Tests set a stub.
	httpDo managementHTTPDoer
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
		"disable_via":    m.cfg.DisableVia,
		"management_url": m.cfg.ManagementURL,
		"has_key":        strings.TrimSpace(m.resolveKeyLocked()) != "",
		"http_mode":      "direct_no_proxy",
		"last_error":     m.lastError,
		"blocked_until":  "",
		"blocked":        false,
	}
	if m.httpDo != nil {
		out["http_mode"] = "injected"
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
	return m.setAuthDisabledWithKey(authID, authIndex, disabled, m.resolveKey())
}

func (m *managementDisabler) setAuthDisabledWithKey(authID, authIndex string, disabled bool, key string) error {
	m.mu.Lock()
	cfg := m.cfg
	blocked := m.blockedUntil
	host := m.host
	do := m.httpDo
	lastBlockedErr := m.lastError
	m.mu.Unlock()

	if time.Now().Before(blocked) {
		return fmt.Errorf("management api cooling down until %s: %s", blocked.Format(time.RFC3339), lastBlockedErr)
	}
	if host == nil {
		return fmt.Errorf("host unavailable")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return errManagementKeyMissing
	}
	if do == nil {
		do = directManagementHTTP
	}

	// Try common name forms: as-is, with/without .json
	candidates := []string{authID}
	if !strings.HasSuffix(strings.ToLower(authID), ".json") {
		candidates = append(candidates, authID+".json")
	} else {
		candidates = append(candidates, strings.TrimSuffix(authID, filepath.Ext(authID)))
	}

	var err error
	for _, name := range candidates {
		err = m.patchAuthStatus(do, cfg, key, name, disabled)
		if err == nil {
			m.mu.Lock()
			m.lastError = ""
			m.mu.Unlock()
			return nil
		}
	}
	if err == nil {
		m.mu.Lock()
		m.lastError = ""
		m.mu.Unlock()
		return nil
	}

	var httpErr *managementHTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound && strings.TrimSpace(authIndex) != "" {
		if file, found, listErr := m.findAuthFile(do, cfg, key, authID, authIndex); listErr == nil && found {
			name := strings.TrimSpace(file.Name)
			if name == "" {
				name = file.ID
			}
			if name != "" && name != authID {
				if err2 := m.patchAuthStatus(do, cfg, key, name, disabled); err2 == nil {
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
		if file, found, verifyErr := m.findAuthFile(do, cfg, key, authID, authIndex); verifyErr == nil && found && file.Disabled == disabled {
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
	return annotateManagementError(err)
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

func (m *managementDisabler) patchAuthStatus(do managementHTTPDoer, cfg PluginConfig, key, name string, disabled bool) error {
	body, err := json.Marshal(map[string]any{"name": name, "disabled": disabled})
	if err != nil {
		return err
	}
	timeout := cfg.ManagementTimeoutSeconds
	if timeout <= 0 {
		timeout = defaultMgmtTimeoutSec
	}
	resp, err := do(pluginapi.HTTPRequest{
		Method: http.MethodPatch,
		URL:    m.baseURL(cfg) + "/auth-files/status",
		Headers: http.Header{
			"Authorization":    {"Bearer " + key},
			"X-Management-Key": {key},
			"Content-Type":     {"application/json"},
			"Accept":           {"application/json"},
		},
		Body: body,
	}, timeout)
	if err != nil {
		return fmt.Errorf("management api call failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &managementHTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(resp.Body))}
	}
	return nil
}

// patchAuthNoteWithKey writes note via PATCH /auth-files/fields (keeps Auth.Disabled).
// Unlike host.auth.save, fields patch does not rebuild Auth as StatusActive.
func (m *managementDisabler) patchAuthNoteWithKey(authID, authIndex, note, key string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return nil
	}
	m.mu.Lock()
	cfg := m.cfg
	do := m.httpDo
	m.mu.Unlock()
	key = strings.TrimSpace(key)
	if key == "" {
		return errManagementKeyMissing
	}
	if do == nil {
		do = directManagementHTTP
	}
	candidates := []string{authID}
	if !strings.HasSuffix(strings.ToLower(authID), ".json") {
		candidates = append(candidates, authID+".json")
	} else {
		candidates = append(candidates, strings.TrimSuffix(authID, filepath.Ext(authID)))
	}
	var last error
	for _, name := range candidates {
		if err := m.patchAuthFields(do, cfg, key, name, map[string]any{"note": note}); err != nil {
			last = err
			continue
		}
		return nil
	}
	if strings.TrimSpace(authIndex) != "" {
		if file, found, listErr := m.findAuthFile(do, cfg, key, authID, authIndex); listErr == nil && found {
			name := strings.TrimSpace(file.Name)
			if name == "" {
				name = file.ID
			}
			if name != "" {
				if err := m.patchAuthFields(do, cfg, key, name, map[string]any{"note": note}); err == nil {
					return nil
				} else {
					last = err
				}
			}
		}
	}
	if last == nil {
		last = fmt.Errorf("note patch failed")
	}
	return last
}

func (m *managementDisabler) patchAuthFields(do managementHTTPDoer, cfg PluginConfig, key, name string, fields map[string]any) error {
	payload := map[string]any{"name": name}
	for k, v := range fields {
		payload[k] = v
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	timeout := cfg.ManagementTimeoutSeconds
	if timeout <= 0 {
		timeout = defaultMgmtTimeoutSec
	}
	resp, err := do(pluginapi.HTTPRequest{
		Method: http.MethodPatch,
		URL:    m.baseURL(cfg) + "/auth-files/fields",
		Headers: http.Header{
			"Authorization":    {"Bearer " + key},
			"X-Management-Key": {key},
			"Content-Type":     {"application/json"},
			"Accept":           {"application/json"},
		},
		Body: body,
	}, timeout)
	if err != nil {
		return fmt.Errorf("management fields patch failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &managementHTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(resp.Body))}
	}
	return nil
}

func (m *managementDisabler) findAuthFile(do managementHTTPDoer, cfg PluginConfig, key, authID, authIndex string) (managementAuthFile, bool, error) {
	timeout := cfg.ManagementTimeoutSeconds
	if timeout <= 0 {
		timeout = defaultMgmtTimeoutSec
	}
	resp, err := do(pluginapi.HTTPRequest{
		Method: http.MethodGet,
		URL:    m.baseURL(cfg) + "/auth-files",
		Headers: http.Header{
			"Authorization":    {"Bearer " + key},
			"X-Management-Key": {key},
			"Accept":           {"application/json"},
		},
	}, timeout)
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

// directManagementHTTP calls CPA Management API with a no-proxy transport.
// Avoids host.HTTPDo → NewProxyAwareHTTPClient → cfg.ProxyURL (e.g. Webshare)
// which rejects private/localhost targets with client_connect_invalid_ip.
func directManagementHTTP(req pluginapi.HTTPRequest, timeoutSec int) (pluginapi.HTTPResponse, error) {
	if timeoutSec <= 0 {
		timeoutSec = defaultMgmtTimeoutSec
	}
	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = http.MethodGet
	}
	httpReq, err := http.NewRequest(method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return pluginapi.HTTPResponse{}, fmt.Errorf("create request: %w", err)
	}
	if req.Headers != nil {
		httpReq.Header = req.Headers.Clone()
	}
	client := &http.Client{
		Timeout:   time.Duration(timeoutSec) * time.Second,
		Transport: directMgmtTransport,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return pluginapi.HTTPResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return pluginapi.HTTPResponse{}, fmt.Errorf("read response: %w", err)
	}
	return pluginapi.HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       body,
	}, nil
}

func isManagementAuthError(err error) bool {
	if errors.Is(err, errManagementKeyMissing) {
		return true
	}
	var httpErr *managementHTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	if httpErr.StatusCode == http.StatusUnauthorized {
		return true
	}
	if httpErr.StatusCode == http.StatusForbidden {
		body := strings.ToLower(httpErr.Body)
		// Proxy / residential-IP blocks are network errors, not bad management keys.
		if strings.Contains(body, "client_connect_invalid_ip") ||
			strings.Contains(body, "forbidden to connect") ||
			strings.Contains(body, "x-webshare") {
			return false
		}
		return true
	}
	return false
}

func annotateManagementError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	if strings.Contains(low, "client_connect_invalid_ip") || strings.Contains(low, "forbidden to connect") {
		return fmt.Errorf("%w（像是全局代理拦截了本机 Management API；本版本已改为直连 127.0.0.1。若仍出现请确认 management_url 为本机可达地址，且未再经 host.HTTPDo）", err)
	}
	if strings.Contains(low, "remote management disabled") {
		return fmt.Errorf("%w（非本机访问需 remote-management.allow-remote=true，或把 management_url 设为 http://127.0.0.1:<port>）", err)
	}
	if strings.Contains(low, "invalid management key") || strings.Contains(low, "missing management key") {
		return fmt.Errorf("%w（密钥须与 CPA remote-management.secret-key / MANAGEMENT_PASSWORD 一致）", err)
	}
	return err
}

// hostHTTPDoer adapts HostClient.HTTPDo for tests that inject stubHost.httpFn.
func hostHTTPDoer(host HostClient) managementHTTPDoer {
	return func(req pluginapi.HTTPRequest, _ int) (pluginapi.HTTPResponse, error) {
		return host.HTTPDo(req)
	}
}
