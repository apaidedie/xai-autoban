package classify

import (
	"net/http"
	"testing"
)

func TestIsFreeUsageExhausted(t *testing.T) {
	if !IsFreeUsageExhausted("subscription:free-usage-exhausted", "You've used all the included free usage") {
		t.Fatal("expected free usage match")
	}
	if IsFreeUsageExhausted("", "rate limit exceeded") {
		t.Fatal("generic rate limit must not match free usage")
	}
}

func TestProbeReauth(t *testing.T) {
	r := Probe(Input{Status: 401, Body: `{"error":"token is expired"}`})
	if r.Classification != Reauth || !r.Isolate || r.RecommendedAction != ActionDelete {
		t.Fatalf("%+v", r)
	}
}

func TestProbeQuotaOn429(t *testing.T) {
	body := `{"code":"subscription:free-usage-exhausted","error":"You've used all the included free usage for model x"}`
	r := Probe(Input{Status: 429, Body: body})
	if r.Classification != QuotaExhausted || !r.Isolate {
		t.Fatalf("%+v", r)
	}
	if r.RecommendedAction != ActionDisable {
		t.Fatalf("want disable, got %s", r.RecommendedAction)
	}
}

func TestProbeBare429(t *testing.T) {
	r := Probe(Input{Status: 429, Body: `{"error":"rate limit"}`})
	if r.Classification != RateLimited || !r.Isolate {
		t.Fatalf("%+v", r)
	}
	if r.RecommendedAction != ActionBan {
		t.Fatalf("bare 429 should recommend ban not disable: %s", r.RecommendedAction)
	}
}

func TestProbePermission(t *testing.T) {
	r := Probe(Input{Status: 403, Body: `{"code":"permission-denied","error":"chat endpoint is denied"}`})
	if r.Classification != PermissionDenied || !r.Isolate {
		t.Fatalf("%+v", r)
	}
}

func TestProbeModelUnavailableNoIsolate(t *testing.T) {
	r := Probe(Input{Status: 404, Body: `{"error":"model does not exist"}`})
	if r.Classification != ModelUnavailable || r.Isolate {
		t.Fatalf("%+v", r)
	}
}

func TestProbeHealthy(t *testing.T) {
	r := Probe(Input{Status: http.StatusOK})
	if r.Classification != Healthy || r.Isolate {
		t.Fatalf("%+v", r)
	}
	r = Probe(Input{Status: 200, Disabled: true})
	if r.RecommendedAction != ActionEnable {
		t.Fatalf("%+v", r)
	}
}

func TestExtractErrorNested(t *testing.T) {
	p := ExtractError(`{"error":{"code":"invalid_grant","message":"bad token"}}`)
	if p.Code != "invalid_grant" || p.Message != "bad token" {
		t.Fatalf("%+v", p)
	}
}
