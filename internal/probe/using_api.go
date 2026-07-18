package probe

import (
	"xai-autoban/internal/config"
)

// ShouldAutoUsingAPI reports whether probe/recheck may auto-enable using_api.
// Feature removed from ops UI: always false (never auto-write using_api).
func ShouldAutoUsingAPI(cfg config.PluginConfig, status int, mat authMaterial, alreadyTriedThisRun bool) bool {
	_ = cfg
	_ = status
	_ = mat
	_ = alreadyTriedThisRun
	return false
}
