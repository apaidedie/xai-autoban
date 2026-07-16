package probe

import (
	"xai-autoban/internal/config"
)

// ShouldAutoUsingAPI reports whether probe/recheck may auto-enable using_api.
// Manual apply-action using_api is never gated by this.
func ShouldAutoUsingAPI(cfg config.PluginConfig, status int, mat authMaterial, alreadyTriedThisRun bool) bool {
	if alreadyTriedThisRun {
		return false
	}
	mode := cfg.AutoUsingAPI
	if mode == "" {
		mode = config.AutoUsingAPIOn403
	}
	if mode == config.AutoUsingAPIOff {
		return false
	}
	if mat.UsingAPI != nil && *mat.UsingAPI {
		return false
	}
	if mat.AuthKind == "api_key" {
		return false
	}
	// OAuth/web only (empty kind treated as eligible when not api_key).
	if mat.AuthKind != "" && mat.AuthKind != "oauth" {
		return false
	}
	switch mode {
	case config.AutoUsingAPIOn403:
		return status == 403
	case config.AutoUsingAPIOnFail:
		return status == 401 || status == 402 || status == 403
	default:
		return status == 403
	}
}
