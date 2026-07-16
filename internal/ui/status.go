package ui

import (
	"encoding/json"
	"html"
	"strings"
)

// StatusPage renders the ops console.
// serverMgmtKey is accepted for ABI compatibility but never embedded in HTML
// (resource ops use GET /data|/ops; secrets stay server-side).
func StatusPage(pluginName, pluginVersion, serverMgmtKey string) string {
	_ = serverMgmtKey
	name := html.EscapeString(pluginName)
	verJS, err := json.Marshal(pluginVersion)
	if err != nil {
		verJS = []byte(`""`)
	}
	verLit := string(verJS) // quoted JSON string e.g. "1.0.8"
	body := strings.Replace(statusBodyTemplate, "v__PLUGIN_VERSION__", "v"+pluginVersion, 1)
	js := strings.Replace(statusJSTemplate, `"__PLUGIN_VERSION__"`, verLit, 1)
	var b strings.Builder
	b.Grow(len(statusCSS) + len(body) + len(js) + 256)
	b.WriteString("<!doctype html>\n<html lang=\"zh-CN\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">\n")
	b.WriteString("<title>")
	b.WriteString(name)
	b.WriteString("</title>\n<style>\n")
	b.WriteString(statusCSS)
	b.WriteString("\n</style>\n")
	b.WriteString(body)
	b.WriteString("<script>\n")
	b.WriteString(js)
	b.WriteString("\n</script>\n</body>\n</html>\n")
	return b.String()
}
