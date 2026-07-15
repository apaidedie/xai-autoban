package reauth

import "testing"

func TestIsRetiredAccountsEndpoint(t *testing.T) {
	if !isRetiredAccountsEndpoint("https://accounts.x.ai/oauth/token") {
		t.Fatal("expected accounts.x.ai retired")
	}
	if isRetiredAccountsEndpoint("https://auth.x.ai/oauth/token") {
		t.Fatal("auth.x.ai must not be treated as retired")
	}
}

func TestFormatTokenErrorJSON(t *testing.T) {
	msg := formatTokenError(403, []byte(`{"error":"access_denied","error_description":"blocked"}`), defaultTokenURL)
	if msg == "" || !containsAll(msg, "403", "access_denied") {
		t.Fatalf("msg=%q", msg)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
