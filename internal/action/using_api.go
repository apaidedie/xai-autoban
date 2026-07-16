package action

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/ban"
	"xai-autoban/internal/host"
	"xai-autoban/internal/xai"
)

// SetUsingAPI flips CPA xAI "使用 API 模式" (using_api) on the auth file.
// Prefer Management PATCH /auth-files/fields; fallback host.auth.save.
func (e *Engine) SetUsingAPI(authID string, enabled bool) error {
	e.mu.Lock()
	hostClient := e.host
	mgmt := e.mgmt
	e.mu.Unlock()
	if hostClient == nil {
		return fmt.Errorf("host callbacks unavailable")
	}
	files, err := hostClient.AuthList()
	if err != nil {
		return err
	}
	var target *pluginapi.HostAuthFileEntry
	for i := range files {
		f := files[i]
		if f.ID == authID || f.AuthIndex == authID || f.Name == authID || ban.AuthIDsEqual(xai.AuthKey(f), authID) {
			target = &f
			break
		}
	}
	if target == nil {
		return fmt.Errorf("credential not found: %s", authID)
	}
	index := target.AuthIndex
	if index == "" {
		index = target.Name
	}
	fileName := strings.TrimSpace(target.Name)
	if fileName == "" {
		fileName = strings.TrimSpace(target.ID)
	}
	if fileName == "" {
		fileName = authID
	}

	reqKey := e.RequestManagementKey()
	cfgKey := ""
	if mgmt != nil {
		cfgKey = mgmt.resolveKey()
	}
	key := reqKey
	if key == "" {
		key = cfgKey
	}

	if mgmt != nil && key != "" {
		if err := mgmt.setUsingAPIWithKey(fileName, index, enabled, key); err != nil {
			slog.Warn("xai-autoban: management using_api patch failed, trying host save",
				"auth_id", authID, "name", fileName, "enabled", enabled, "error", err)
		} else if vErr := e.verifyUsingAPI(hostClient, index, enabled); vErr == nil {
			if e.onUsingAPI != nil {
				e.onUsingAPI(authID, fileName, index, enabled)
			}
			slog.Info("xai-autoban: set using_api via management",
				"auth_id", authID, "name", fileName, "using_api", enabled)
			return nil
		} else {
			slog.Warn("xai-autoban: management using_api not reflected, trying host save",
				"auth_id", authID, "name", fileName, "error", vErr)
		}
	}

	if err := e.patchHostUsingAPI(hostClient, index, fileName, enabled); err != nil {
		return err
	}
	if vErr := e.verifyUsingAPI(hostClient, index, enabled); vErr != nil {
		return vErr
	}
	if e.onUsingAPI != nil {
		e.onUsingAPI(authID, fileName, index, enabled)
	}
	slog.Info("xai-autoban: set using_api via host.auth.save",
		"auth_id", authID, "name", fileName, "using_api", enabled)
	return nil
}

func (e *Engine) verifyUsingAPI(hostClient host.Client, index string, want bool) error {
	got, err := hostClient.AuthGet(index)
	if err != nil {
		return fmt.Errorf("using_api verify AuthGet: %w", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(got.JSON, &obj); err != nil {
		return fmt.Errorf("using_api verify parse: %w", err)
	}
	have, ok := readUsingAPIFlag(obj)
	if !ok {
		return fmt.Errorf("using_api write not reflected (want=%v got=<missing>)", want)
	}
	if have != want {
		return fmt.Errorf("using_api write not reflected (want=%v got=%v)", want, have)
	}
	return nil
}

func readUsingAPIFlag(obj map[string]any) (bool, bool) {
	if v, ok := obj["using_api"].(bool); ok {
		return v, true
	}
	if s, ok := obj["using_api"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		}
	}
	if meta, ok := obj["metadata"].(map[string]any); ok {
		if v, ok := meta["using_api"].(bool); ok {
			return v, true
		}
		if s, ok := meta["using_api"].(string); ok {
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "true", "1", "yes":
				return true, true
			case "false", "0", "no":
				return false, true
			}
		}
	}
	return false, false
}

func (e *Engine) patchHostUsingAPI(hostClient host.Client, index, fileName string, enabled bool) error {
	got, err := hostClient.AuthGet(index)
	if err != nil {
		return err
	}
	var obj map[string]any
	if err := json.Unmarshal(got.JSON, &obj); err != nil {
		return err
	}
	obj["using_api"] = enabled
	if meta, ok := obj["metadata"].(map[string]any); ok {
		meta["using_api"] = enabled
		obj["metadata"] = meta
	} else {
		obj["metadata"] = map[string]any{"using_api": enabled}
	}
	if attrs, ok := obj["attributes"].(map[string]any); ok {
		if enabled {
			attrs["using_api"] = "true"
		} else {
			attrs["using_api"] = "false"
		}
		obj["attributes"] = attrs
	} else {
		obj["attributes"] = map[string]any{"using_api": map[bool]string{true: "true", false: "false"}[enabled]}
	}
	if enabled {
		if _, ok := obj["base_url"]; !ok {
			obj["base_url"] = "https://api.x.ai/v1"
		}
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	name := fileName
	if name == "" {
		name = got.Name
	}
	_, err = hostClient.AuthSave(name, raw)
	return err
}
