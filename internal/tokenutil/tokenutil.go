package tokenutil

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// LocalInspect summarizes offline token health without calling upstream.
type LocalInspect struct {
	AccessToken     string
	RefreshToken    string
	ExpiresAt       time.Time
	TokenExpired    bool
	NeedsRefresh    bool
	HasRefreshToken bool
}

// InspectAuthJSON reads common xAI/CPA credential shapes.
func InspectAuthJSON(raw json.RawMessage, now time.Time) LocalInspect {
	out := LocalInspect{}
	if len(raw) == 0 {
		return out
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return out
	}
	out.AccessToken = firstString(obj, "access_token", "accessToken", "token", "api_key", "apiKey")
	if out.AccessToken == "" {
		if nested, ok := obj["token"].(map[string]any); ok {
			out.AccessToken = firstString(nested, "access_token", "accessToken")
			out.RefreshToken = firstString(nested, "refresh_token", "refreshToken")
		}
		if nested, ok := obj["tokens"].(map[string]any); ok {
			if out.AccessToken == "" {
				out.AccessToken = firstString(nested, "access_token", "accessToken")
			}
			if out.RefreshToken == "" {
				out.RefreshToken = firstString(nested, "refresh_token", "refreshToken")
			}
		}
	}
	if out.RefreshToken == "" {
		out.RefreshToken = firstString(obj, "refresh_token", "refreshToken")
	}
	out.HasRefreshToken = strings.TrimSpace(out.RefreshToken) != ""

	if exp := parseTimeAny(obj["expires_at"]); !exp.IsZero() {
		out.ExpiresAt = exp
	} else if exp := parseTimeAny(obj["expired"]); !exp.IsZero() {
		out.ExpiresAt = exp
	} else if exp := parseTimeAny(obj["expiry"]); !exp.IsZero() {
		out.ExpiresAt = exp
	} else if nested, ok := obj["token"].(map[string]any); ok {
		if exp := parseTimeAny(nested["expires_at"]); !exp.IsZero() {
			out.ExpiresAt = exp
		}
	}
	if out.ExpiresAt.IsZero() && out.AccessToken != "" {
		if exp, ok := jwtExp(out.AccessToken); ok {
			out.ExpiresAt = exp
		}
	}
	if !out.ExpiresAt.IsZero() {
		// refresh window: expired or within 5 minutes
		if !out.ExpiresAt.After(now) {
			out.TokenExpired = true
			out.NeedsRefresh = true
		} else if out.ExpiresAt.Sub(now) <= 5*time.Minute {
			out.NeedsRefresh = true
		}
	}
	return out
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseTimeAny(v any) time.Time {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return time.Time{}
		}
		if ts, err := time.Parse(time.RFC3339, s); err == nil {
			return ts
		}
		if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return ts
		}
	case float64:
		// unix seconds or ms
		n := int64(t)
		if n > 1_000_000_000_000 {
			n /= 1000
		}
		if n > 0 {
			return time.Unix(n, 0)
		}
	case json.Number:
		if n, err := t.Int64(); err == nil {
			if n > 1_000_000_000_000 {
				n /= 1000
			}
			return time.Unix(n, 0)
		}
	}
	return time.Time{}
}

func jwtExp(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// try padded std encoding
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, false
		}
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, false
	}
	switch exp := claims["exp"].(type) {
	case float64:
		return time.Unix(int64(exp), 0), true
	case json.Number:
		n, err := exp.Int64()
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(n, 0), true
	default:
		return time.Time{}, false
	}
}
