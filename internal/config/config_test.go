package config

import (
	"testing"
)

func TestNormalizeAutoUsingAPI(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", AutoUsingAPIOn403},
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
		{"nope", AutoUsingAPIOn403},
	}
	for _, tc := range cases {
		cfg, _ := Normalize(PluginConfig{AutoUsingAPI: tc.in})
		if cfg.AutoUsingAPI != tc.want {
			t.Fatalf("in=%q got=%q want=%q", tc.in, cfg.AutoUsingAPI, tc.want)
		}
	}
}

func TestDefaultAutoUsingAPI(t *testing.T) {
	if Default().AutoUsingAPI != AutoUsingAPIOn403 {
		t.Fatalf("default=%q", Default().AutoUsingAPI)
	}
}

func TestPublicViewIncludesAutoUsingAPI(t *testing.T) {
	v := Default().PublicView()
	if v["auto_using_api"] != AutoUsingAPIOn403 {
		t.Fatalf("%#v", v["auto_using_api"])
	}
	ops := Default().OpsSettingsView()
	if ops["auto_using_api"] != AutoUsingAPIOn403 {
		t.Fatalf("ops missing auto_using_api: %#v", ops["auto_using_api"])
	}
}
