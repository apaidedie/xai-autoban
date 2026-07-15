package tokenutil

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestInspectExpiredJWT(t *testing.T) {
	// header.payload.sig with exp in the past
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"exp":1700000000}`))
	tok := "eyJhbGciOiJub25lIn0." + payload + ".x"
	raw, _ := json.Marshal(map[string]any{"access_token": tok, "refresh_token": "rt"})
	now := time.Unix(1_800_000_000, 0)
	got := InspectAuthJSON(raw, now)
	if !got.TokenExpired || !got.NeedsRefresh || !got.HasRefreshToken {
		t.Fatalf("%#v", got)
	}
}

func TestInspectExpiresAtFuture(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	raw, _ := json.Marshal(map[string]any{
		"access_token":  "plain",
		"refresh_token": "rt",
		"expires_at":    now.Add(2 * time.Hour).Format(time.RFC3339),
	})
	got := InspectAuthJSON(raw, now)
	if got.TokenExpired || got.NeedsRefresh {
		t.Fatalf("%#v", got)
	}
}
