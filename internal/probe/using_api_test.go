package probe

import (
	"testing"

	"xai-autoban/internal/config"
)

func TestShouldAutoUsingAPI(t *testing.T) {
	oauth := authMaterial{AuthKind: "oauth", Token: "t"}
	apiKey := authMaterial{AuthKind: "api_key", Token: "k"}
	already := true
	oauthAPI := authMaterial{AuthKind: "oauth", Token: "t", UsingAPI: &already}

	cases := []struct {
		name   string
		mode   string
		status int
		mat    authMaterial
		tried  bool
		want   bool
	}{
		{"off+403", config.AutoUsingAPIOff, 403, oauth, false, false},
		{"on403+401", config.AutoUsingAPIOn403, 401, oauth, false, false},
		{"on403+402", config.AutoUsingAPIOn403, 402, oauth, false, false},
		{"on403+403", config.AutoUsingAPIOn403, 403, oauth, false, true},
		{"onfail+401", config.AutoUsingAPIOnFail, 401, oauth, false, true},
		{"onfail+403", config.AutoUsingAPIOnFail, 403, oauth, false, true},
		{"api_key", config.AutoUsingAPIOn403, 403, apiKey, false, false},
		{"already_using_api", config.AutoUsingAPIOn403, 403, oauthAPI, false, false},
		{"tried", config.AutoUsingAPIOn403, 403, oauth, true, false},
		{"empty_kind_oauth_tokens", config.AutoUsingAPIOn403, 403, authMaterial{AuthKind: "", Token: "t"}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.AutoUsingAPI = tc.mode
			got := ShouldAutoUsingAPI(cfg, tc.status, tc.mat, tc.tried)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
