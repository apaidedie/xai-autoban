package persist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStatePathAbsoluteUnchanged(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	got := ResolveStatePath(p)
	if got != filepath.Clean(p) {
		t.Fatalf("got %q want %q", got, p)
	}
}

func TestResolveStatePathFindsExistingRelative(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	name := "xai-autoban-state.json"
	if err := os.WriteFile(name, []byte(`{"version":2,"bans":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := ResolveStatePath(name)
	want, _ := filepath.Abs(name)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveStatePathUsesDataEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XAI_AUTOBAN_DATA_DIR", dir)
	// no existing file — should place under env dir
	got := ResolveStatePath("xai-autoban-state.json")
	want := filepath.Join(dir, "xai-autoban-state.json")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
