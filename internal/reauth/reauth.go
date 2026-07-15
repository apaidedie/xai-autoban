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

const defaultTokenURL = "https://accounts.x.ai/oauth/token"
const defaultProbeBase = "https://api.x.ai/v1"

// Direct no-proxy transport (same idea as management client — avoid CPA global proxy).
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
// Token endpoint uses direct HTTP (no host proxy). Optional verifyProbe hits /models.
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
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", local.RefreshToken)
	var obj map[string]any
	_ = json.Unmarshal(got.JSON, &obj)
	if cid := stringField(obj, "client_id", "clientId"); cid != "" {
		form.Set("client_id", cid)
	}

	resp, err := directHTTP(pluginapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    tokenURL,
		Headers: http.Header{
			"Content-Type": {"application/x-www-form-urlencoded"},
			"Accept":       {"application/json"},
		},
		Body: []byte(form.Encode()),
	}, 30)
	if err != nil {
		return Result{AuthID: id, Message: err.Error()}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(resp.Body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return Result{AuthID: id, Status: resp.StatusCode, Message: msg}, fmt.Errorf("token endpoint HTTP %d", resp.StatusCode)
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
	}
	if tok.ExpiresAt != "" {
		obj["expires_at"] = tok.ExpiresAt
	} else if tok.ExpiresIn > 0 {
		obj["expires_at"] = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).Format(time.RFC3339)
	}
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

func probeModels(accessToken string) (int, error) {
	resp, err := directHTTP(pluginapi.HTTPRequest{
		Method: http.MethodGet,
		URL:    defaultProbeBase + "/models",
		Headers: http.Header{
			"Authorization": {"Bearer " + accessToken},
			"Accept":        {"application/json"},
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
