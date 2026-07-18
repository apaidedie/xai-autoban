package probe

import (
	"testing"

	"xai-autoban/internal/config"
)

func TestShouldAutoUsingAPIAlwaysOff(t *testing.T) {
	cfg := config.Default()
	cfg.AutoUsingAPI = config.AutoUsingAPIOn403
	oauth := authMaterial{AuthKind: "oauth", Token: "t"}
	if ShouldAutoUsingAPI(cfg, 403, oauth, false) {
		t.Fatal("auto using_api must stay off after feature removal")
	}
	cfg.AutoUsingAPI = config.AutoUsingAPIOnFail
	if ShouldAutoUsingAPI(cfg, 401, oauth, false) {
		t.Fatal("auto using_api on_fail must not enable")
	}
}
