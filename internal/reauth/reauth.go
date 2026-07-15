package reauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/host"
	"xai-autoban/internal/tokenutil"
	"xai-autoban/internal/xai"
)

// Grok CLI / shared public OAuth client (same as openclaw / official device-code flows).
const defaultClientID = "b1a00492-073a-47ea-816f-4c329264a828"

// auth.x.ai is the OIDC issuer; accounts.x.ai is the browser sign-in host (returns 403 on /oauth/token).
const defaultTokenURL = "https://auth.x.ai/oauth/token"
const discoveryURL = "https://auth.x.ai/.well-known/openid-configuration"
const defaultProbeBase = "https://api.x.ai/v1"
const defaultUserAgent = "xai-autoban (oauth-refresh)"

// Direct no-proxy transport (avoid CPA global proxy for OAuth).
var directTransport = &http.Transport{
	Proxy: nil,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          16,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// Result of a refresh attempt.
type Result struct {
	OK          bool   `json:"ok"`
	AuthID      string `json:"auth_id"`
	Message     string `json:"message,omitempty"`
	Status      int    `json:"status,omitempty"`
	ProbeStatus int    `json:"probe_status,omitempty"`
	ProbeOK     bool   `json:"probe_ok,omitempty"`
}

// RefreshOne uses refresh_token to obtain a new access_token and AuthSave it.
// Token endpoint uses direct HTTP to auth.x.ai (not accounts.x.ai).
// Does not launch browser OAuth (use cpa-auth-inspect for Chromium reauth).
func RefreshOne(h host.Client, f pluginapi.HostAuthFileEntry, tokenURL string) (Result, error) {
	return RefreshOneOpts(h, f, tokenURL, true)
}

func RefreshOneOpts(h host.Client, f pluginapi.HostAuthFileEntry, tokenURL string, verifyProbe bool) (Result, error) {
	id := xai.AuthKey(f)
	index := f.AuthIndex
	if index == "" {
		index = f.Name
	}
	got, err := h.AuthGet(index)
	if err != nil {
		return Result{AuthID: id, Message: err.Error()}, err
	}
	local := tokenutil.InspectAuthJSON(got.JSON, time.Now())
	if !local.HasRefreshToken {
		return Result{AuthID: id, Message: "no refresh_token in credential"}, fmt.Errorf("no refresh_token")
	}
	var obj map[string]any
	_ = json.Unmarshal(got.JSON, &obj)

	endpoint := strings.TrimSpace(tokenURL)
	if endpoint == "" {
		endpoint = stringField(obj, "token_endpoint", "tokenEndpoint", "oidc_token_url")
	}
	if endpoint == "" || isRetiredAccountsEndpoint(endpoint) {
		if discovered := discoverTokenEndpoint(); discovered != "" {
			endpoint = discovered
		} else {
			endpoint = defaultTokenURL
		}
	}
	if isRetiredAccountsEndpoint(endpoint) {
		endpoint = defaultTokenURL
	}

	clientID := stringField(obj, "client_id", "clientId")
	if clientID == "" {
		clientID = defaultClientID
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", local.RefreshToken)
	form.Set("client_id", clientID)

	resp, err := directHTTP(pluginapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    endpoint,
		Headers: http.Header{
			"Content-Type": {"application/x-www-form-urlencoded"},
			"Accept":       {"application/json"},
			"User-Agent":   {defaultUserAgent},
		},
		Body: []byte(form.Encode()),
	}, 30)
	if err != nil {
		return Result{AuthID: id, Message: err.Error()}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := formatTokenError(resp.StatusCode, resp.Body, endpoint)
		return Result{AuthID: id, Status: resp.StatusCode, Message: msg}, fmt.Errorf("%s", msg)
	}
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		ExpiresAt    string `json:"expires_at"`
	}
	if err := json.Unmarshal(resp.Body, &tok); err != nil {
		return Result{AuthID: id, Message: "decode token response: " + err.Error()}, err
	}
	if strings.TrimSpace(tok.AccessToken) == "" {
		return Result{AuthID: id, Message: "empty access_token in response"}, fmt.Errorf("empty access_token")
	}

	if obj == nil {
		obj = map[string]any{}
	}
	obj["access_token"] = tok.AccessToken
	if tok.RefreshToken != "" {
		obj["refresh_token"] = tok.RefreshToken
	} else {
		// Keep old refresh token when server does not rotate.
		obj["refresh_token"] = local.RefreshToken
	}
	if tok.ExpiresAt != "" {
		obj["expires_at"] = tok.ExpiresAt
	} else if tok.ExpiresIn > 0 {
		obj["expires_at"] = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).Format(time.RFC3339)
	}
	obj["token_endpoint"] = endpoint
	obj["client_id"] = clientID
	if nested, ok := obj["token"].(map[string]any); ok {
		nested["access_token"] = tok.AccessToken
		if tok.RefreshToken != "" {
			nested["refresh_token"] = tok.RefreshToken
		}
		obj["token"] = nested
	}
	if nested, ok := obj["tokens"].(map[string]any); ok {
		nested["access_token"] = tok.AccessToken
		if tok.RefreshToken != "" {
			nested["refresh_token"] = tok.RefreshToken
		}
		obj["tokens"] = nested
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		return Result{AuthID: id, Message: err.Error()}, err
	}
	name := strings.TrimSpace(f.Name)
	if name == "" {
		name = strings.TrimSpace(f.ID)
	}
	if name == "" {
		name = id
	}
	if _, err := h.AuthSave(name, raw); err != nil {
		return Result{AuthID: id, Message: "auth save: " + err.Error()}, err
	}

	out := Result{OK: true, AuthID: id, Message: "refreshed access_token"}
	if verifyProbe {
		st, perr := probeModels(tok.AccessToken)
		out.ProbeStatus = st
		out.ProbeOK = perr == nil && st >= 200 && st < 300
		if !out.ProbeOK {
			if perr != nil {
				out.Message = "refreshed but probe failed: " + perr.Error()
			} else {
				out.Message = fmt.Sprintf("refreshed but probe status %d", st)
			}
		} else {
			out.Message = "refreshed access_token; probe ok"
		}
	}
	return out, nil
}

func isRetiredAccountsEndpoint(endpoint string) bool {
	u := strings.ToLower(strings.TrimSpace(endpoint))
	return strings.Contains(u, "accounts.x.ai")
}

func discoverTokenEndpoint() string {
	resp, err := directHTTP(pluginapi.HTTPRequest{
		Method: http.MethodGet,
		URL:    discoveryURL,
		Headers: http.Header{
			"Accept":     {"application/json"},
			"User-Agent": {defaultUserAgent},
		},
	}, 15)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	var doc struct {
		TokenEndpoint string `json:"token_endpoint"`
	}
	if json.Unmarshal(resp.Body, &doc) != nil {
		return ""
	}
	ep := strings.TrimSpace(doc.TokenEndpoint)
	if ep == "" || !strings.Contains(strings.ToLower(ep), "x.ai") {
		return ""
	}
	return ep
}

func formatTokenError(status int, body []byte, endpoint string) string {
	text := strings.TrimSpace(string(body))
	// Cloudflare HTML challenge
	low := strings.ToLower(text)
	if strings.Contains(low, "<html") || strings.Contains(low, "cloudflare") || strings.Contains(low, "just a moment") {
		return fmt.Sprintf("token endpoint HTTP %d (Cloudflare/HTML block on %s); retry later or use cpa-auth-inspect browser reauth", status, endpoint)
	}
	// JSON error
	var errObj map[string]any
	if json.Unmarshal(body, &errObj) == nil {
		code := stringField(errObj, "error")
		desc := stringField(errObj, "error_description", "message")
		if code != "" && desc != "" {
			return fmt.Sprintf("token endpoint HTTP %d: %s (%s)", status, code, desc)
		}
		if code != "" {
			return fmt.Sprintf("token endpoint HTTP %d: %s", status, code)
		}
	}
	if text == "" {
		return fmt.Sprintf("token endpoint HTTP %d (%s)", status, endpoint)
	}
	if len(text) > 180 {
		text = text[:180]
	}
	return fmt.Sprintf("token endpoint HTTP %d: %s", status, text)
}

func probeModels(accessToken string) (int, error) {
	resp, err := directHTTP(pluginapi.HTTPRequest{
		Method: http.MethodGet,
		URL:    defaultProbeBase + "/models",
		Headers: http.Header{
			"Authorization": {"Bearer " + accessToken},
			"Accept":        {"application/json"},
			"User-Agent":    {defaultUserAgent},
		},
	}, 20)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("probe status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func directHTTP(req pluginapi.HTTPRequest, timeoutSec int) (pluginapi.HTTPResponse, error) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = http.MethodGet
	}
	httpReq, err := http.NewRequest(method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return pluginapi.HTTPResponse{}, err
	}
	if req.Headers != nil {
		httpReq.Header = req.Headers.Clone()
	}
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second, Transport: directTransport}
	resp, err := client.Do(httpReq)
	if err != nil {
		return pluginapi.HTTPResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return pluginapi.HTTPResponse{}, err
	}
	return pluginapi.HTTPResponse{StatusCode: resp.StatusCode, Headers: resp.Header.Clone(), Body: body}, nil
}

func stringField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
