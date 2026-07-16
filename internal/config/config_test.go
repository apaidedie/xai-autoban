package config

import (
	"testing"
)

func TestNormalizeAutoUsingAPI(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", AutoUsingAPIOff},
		{"on_403", AutoUsingAPIOn403},
		{"ON_403", AutoUsingAPIOn403},
		{"true", AutoUsingAPIOn403},
		{"1", AutoUsingAPIOn403},
		{"403", AutoUsingAPIOn403},
		{"off", AutoUsingAPIOff},
		{"false", AutoUsingAPIOff},
		{"0", AutoUsingAPIOff},
		{"on_fail", AutoUsingAPIOnFail},
		{"all", AutoUsingAPIOnFail},
		{"fail", AutoUsingAPIOnFail},
		{"nope", AutoUsingAPIOff},
	}
	for _, tc := range cases {
		cfg, _ := Normalize(PluginConfig{AutoUsingAPI: tc.in})
		if cfg.AutoUsingAPI != tc.want {
			t.Fatalf("in=%q got=%q want=%q", tc.in, cfg.AutoUsingAPI, tc.want)
		}
	}
}

func TestDefaultAutoUsingAPI(t *testing.T) {
	if Default().AutoUsingAPI != AutoUsingAPIOff {
		t.Fatalf("default=%q want off", Default().AutoUsingAPI)
	}
}

func TestPublicViewIncludesAutoUsingAPI(t *testing.T) {
	v := Default().PublicView()
	if v["auto_using_api"] != AutoUsingAPIOff {
		t.Fatalf("%#v", v["auto_using_api"])
	}
	ops := Default().OpsSettingsView()
	if ops["auto_using_api"] != AutoUsingAPIOff {
		t.Fatalf("ops missing auto_using_api: %#v", ops["auto_using_api"])
	}
}

// TestFrozenOpsKeysInPublicView guards STABILITY.md §3 freeze: every ops key must appear in PublicView.
func TestFrozenOpsKeysInPublicView(t *testing.T) {
	view := Default().PublicView()
	for _, k := range OpsSettingsKeys {
		if _, ok := view[k]; !ok {
			t.Errorf("frozen ops key %q missing from PublicView", k)
		}
	}
	ops := Default().OpsSettingsView()
	if len(ops) != len(OpsSettingsKeys) {
		t.Fatalf("OpsSettingsView size %d want %d", len(ops), len(OpsSettingsKeys))
	}
	for _, k := range OpsSettingsKeys {
		if _, ok := ops[k]; !ok {
			t.Errorf("frozen ops key %q missing from OpsSettingsView", k)
		}
	}
}

func TestInstallConfigKeysDocumented(t *testing.T) {
	// Install keys are not all in PublicView (secrets); ensure list is non-empty and stable count.
	if len(InstallConfigKeys) < 5 {
		t.Fatalf("InstallConfigKeys too short: %v", InstallConfigKeys)
	}
	seen := map[string]struct{}{}
	for _, k := range InstallConfigKeys {
		if k == "" {
			t.Fatal("empty install key")
		}
		if _, ok := seen[k]; ok {
			t.Fatalf("duplicate install key %q", k)
		}
		seen[k] = struct{}{}
	}
}
