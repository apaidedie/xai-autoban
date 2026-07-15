package probe

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
)

func TestProbeOneModelsSuccess(t *testing.T) {
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			t.Fatalf("unexpected url %s", req.URL)
			return pluginapi.HTTPResponse{}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "models"
	st, body, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 || !strings.Contains(body, "grok-4.5") {
		t.Fatalf("st=%d body=%q err=%v", st, body, err)
	}
}

func TestProbeOneBare429RetriesOnce(t *testing.T) {
	var responsesHits int32
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			if strings.Contains(req.URL, "/responses") {
				n := atomic.AddInt32(&responsesHits, 1)
				if n == 1 {
					return pluginapi.HTTPResponse{StatusCode: 429, Body: []byte(`{"error":{"message":"rate limited"}}`)}, nil
				}
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"id":"ok"}`)}, nil
			}
			t.Fatalf("unexpected url %s", req.URL)
			return pluginapi.HTTPResponse{}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, _, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 {
		t.Fatalf("st=%d err=%v hits=%d", st, err, responsesHits)
	}
	if responsesHits < 2 {
		t.Fatalf("expected responses retry, hits=%d", responsesHits)
	}
}

func TestProbeOneFreeUsageNoRetry(t *testing.T) {
	var responsesHits int32
	body429 := `{"error":{"code":"free-usage-exhausted","message":"used all the included free usage"}}`
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			if strings.Contains(req.URL, "/responses") {
				atomic.AddInt32(&responsesHits, 1)
				return pluginapi.HTTPResponse{StatusCode: 429, Body: []byte(body429)}, nil
			}
			// completions fallback may also see free-usage; do not count as responses retry
			return pluginapi.HTTPResponse{StatusCode: 429, Body: []byte(body429)}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, body, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if st != 429 || err == nil {
		t.Fatalf("st=%d err=%v body=%q", st, err, body)
	}
	if responsesHits != 1 {
		t.Fatalf("free-usage must not 429-retry responses, responsesHits=%d", responsesHits)
	}
}

func TestProbeOneResponsesFallbackCompletions(t *testing.T) {
	var sawResponses, sawCompletions bool
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			if strings.Contains(req.URL, "/responses") {
				sawResponses = true
				return pluginapi.HTTPResponse{StatusCode: 403, Body: []byte(`{"error":{"message":"denied"}}`)}, nil
			}
			if strings.Contains(req.URL, "/chat/completions") {
				sawCompletions = true
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"choices":[]}`)}, nil
			}
			t.Fatalf("url %s", req.URL)
			return pluginapi.HTTPResponse{}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, _, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 || !sawResponses || !sawCompletions {
		t.Fatalf("st=%d err=%v sawResponses=%v sawCompletions=%v", st, err, sawResponses, sawCompletions)
	}
}

func TestProbeOneResponsesRealBody(t *testing.T) {
	var gotBody, gotURL string
	var hdr http.Header
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok","refresh_token":"rt","auth_kind":"oauth","type":"xai"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			if strings.Contains(req.URL, "/responses") {
				gotURL = req.URL
				gotBody = string(req.Body)
				hdr = req.Headers
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"id":"resp_1","output":[]}`)}, nil
			}
			t.Fatalf("url %s", req.URL)
			return pluginapi.HTTPResponse{}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, _, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 {
		t.Fatalf("st=%d err=%v", st, err)
	}
	if !strings.Contains(gotURL, "cli-chat-proxy.grok.com") {
		t.Fatalf("oauth probe must use chat-proxy, url=%s", gotURL)
	}
	if hdr.Get("X-XAI-Token-Auth") != "xai-grok-cli" {
		t.Fatalf("missing grok cli headers: %v", hdr)
	}
	if !strings.Contains(gotBody, `"model"`) || !strings.Contains(gotBody, "OK") {
		t.Fatalf("expected real responses body, got %s", gotBody)
	}
}

func TestProbeOneAPIKeyUsesOfficialAPI(t *testing.T) {
	var gotURL string
	stub := &host.Stub{
		Files:  []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"api_key":"xai-key-1"}`)},
		HTTPFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			if strings.Contains(req.URL, "/responses") {
				gotURL = req.URL
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"id":"ok"}`)}, nil
			}
			return pluginapi.HTTPResponse{}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, _, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 {
		t.Fatalf("st=%d err=%v", st, err)
	}
	if !strings.Contains(gotURL, "api.x.ai") {
		t.Fatalf("api_key probe must use api.x.ai, url=%s", gotURL)
	}
}
