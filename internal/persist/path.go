package persist

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultStateFileName = "xai-autoban-state.json"

// ResolveStatePath turns a configured state_file into a stable absolute path.
// Relative names (default xai-autoban-state.json) are resolved under a durable
// data directory so CPA container rebuild / CWD change does not drop ops settings.
//
// Order:
//  1. Absolute configured path as-is
//  2. Existing file under CWD / env data dirs / exe dir (migrate old installs)
//  3. First writable data dir (env → exe/data → user config → cwd/data)
//  4. Absolute path of basename under CWD
func ResolveStatePath(configured string) string {
	name := strings.TrimSpace(configured)
	if name == "" {
		name = defaultStateFileName
	}
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	baseName := filepath.Base(name)
	if baseName == "" || baseName == "." {
		baseName = defaultStateFileName
	}

	// Preserve existing relative file (upgrade path from pre-1.1.2 installs).
	if abs, err := filepath.Abs(name); err == nil {
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			return abs
		}
	}

	var bases []string
	for _, e := range []string{
		"XAI_AUTOBAN_DATA_DIR",
		"CPA_DATA_DIR",
		"CLIPROXYAPI_DATA_DIR",
		"DATA_DIR",
		"CPA_HOME",
	} {
		if v := strings.TrimSpace(os.Getenv(e)); v != "" {
			bases = append(bases, v)
		}
	}
	if ex, err := os.Executable(); err == nil {
		dir := filepath.Dir(ex)
		bases = append(bases,
			filepath.Join(dir, "data"),
			dir,
			filepath.Clean(filepath.Join(dir, "..", "data")),
			filepath.Clean(filepath.Join(dir, "..", "config")),
		)
	}
	if h, err := os.UserConfigDir(); err == nil {
		bases = append(bases, filepath.Join(h, "cliproxyapi"), filepath.Join(h, "xai-autoban"))
	}
	if h, err := os.UserHomeDir(); err == nil {
		bases = append(bases, filepath.Join(h, ".cliproxyapi"), filepath.Join(h, ".cpa"))
	}
	if wd, err := os.Getwd(); err == nil {
		bases = append(bases, filepath.Join(wd, "data"), filepath.Join(wd, "config"), wd)
	}

	// Prefer an already-existing state file.
	for _, base := range uniqueNonEmpty(bases) {
		p := filepath.Join(base, baseName)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	// Prefer a writable durable directory.
	for _, base := range uniqueNonEmpty(bases) {
		if !dirWritable(base) {
			continue
		}
		return filepath.Join(base, baseName)
	}
	abs, err := filepath.Abs(baseName)
	if err != nil {
		return baseName
	}
	return abs
}

func uniqueNonEmpty(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = filepath.Clean(strings.TrimSpace(s))
		if s == "" || s == "." {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func dirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	test := filepath.Join(dir, ".xai-autoban-writetest")
	if err := os.WriteFile(test, []byte("1"), 0o600); err != nil {
		return false
	}
	_ = os.Remove(test)
	return true
}
