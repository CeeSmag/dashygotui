package browse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/theme"
)

func TestNavigatorListsDirsAndStats(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Alpha"))
	mustWrite(t, filepath.Join(root, "Alpha", "a.txt"), "hello")
	mustMkdir(t, filepath.Join(root, "Beta"))
	mustWrite(t, filepath.Join(root, "file.txt"), "skip")

	n := New(root)
	vis := n.Visible()
	if len(vis) != 2 {
		t.Fatalf("entries = %d, want 2 dirs (got %#v)", len(vis), names(vis))
	}
	if vis[0].Name != "Alpha" || vis[1].Name != "Beta" {
		t.Fatalf("order = %v", names(vis))
	}
	if vis[0].Files != 1 || vis[0].Bytes != 5 {
		t.Fatalf("Alpha stats files=%d bytes=%d", vis[0].Files, vis[0].Bytes)
	}
	if vis[1].Files != 0 || vis[1].Bytes != 0 {
		t.Fatalf("Beta stats files=%d bytes=%d", vis[1].Files, vis[1].Bytes)
	}
}

func TestLetterJumpCycles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Docs"))
	mustMkdir(t, filepath.Join(root, "Downloads"))
	mustMkdir(t, filepath.Join(root, "Music"))
	n := New(root)

	act, ok := n.HandleKey("d", 10)
	if !ok || act != ActionNone {
		t.Fatalf("first d: act=%v ok=%v", act, ok)
	}
	if e, _ := n.Selected(); e.Name != "Docs" {
		t.Fatalf("first d -> %q", e.Name)
	}
	_, _ = n.HandleKey("d", 10)
	if e, _ := n.Selected(); e.Name != "Downloads" {
		t.Fatalf("second d -> %q", e.Name)
	}
	_, _ = n.HandleKey("d", 10)
	if e, _ := n.Selected(); e.Name != "Docs" {
		t.Fatalf("third d wraps -> %q", e.Name)
	}
}

func TestSearchFilter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "VaultStorage"))
	mustMkdir(t, filepath.Join(root, "Videos"))
	n := New(root)
	_, _ = n.HandleKey("ctrl+s", 10)
	if !n.Searching() {
		t.Fatal("expected search mode")
	}
	_, _ = n.HandleKey("v", 10)
	_, _ = n.HandleKey("a", 10)
	_, _ = n.HandleKey("u", 10)
	vis := n.Visible()
	if len(vis) != 1 || vis[0].Name != "VaultStorage" {
		t.Fatalf("filter %q -> %#v", n.Filter(), names(vis))
	}
}

func TestHiddenDirsDefaultAndToggle(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Visible"))
	mustMkdir(t, filepath.Join(root, ".config"))
	n := New(root)
	if n.ShowHidden() {
		t.Fatal("hidden should be off by default")
	}
	if got := names(n.Visible()); len(got) != 1 || got[0] != "Visible" {
		t.Fatalf("visible = %v", got)
	}
	_, _ = n.HandleKey("ctrl+h", 10)
	if !n.ShowHidden() {
		t.Fatal("ctrl+h should show hidden")
	}
	if got := names(n.Visible()); len(got) != 2 {
		t.Fatalf("with hidden = %v", got)
	}
}

func TestParentStopsAtRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	child := filepath.Join(root, "child")
	mustMkdir(t, child)
	n := New(child)
	n.Parent()
	if n.Dir() != root {
		t.Fatalf("parent = %q, want %q", n.Dir(), root)
	}
	// filepath.Dir of a cleaned abs path stops changing at the volume root.
	n2 := &Navigator{home: "", cache: map[string]dirStats{}, dirsOnly: true}
	n2.open("/", "")
	before := n2.Dir()
	n2.Parent()
	if n2.Dir() != before {
		t.Fatalf("parent of / became %q", n2.Dir())
	}
}

func TestOpenAndSelect(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Vault"))
	n := New(root)
	n.Focus("Vault")
	_, _ = n.HandleKey("enter", 10)
	if n.Dir() != filepath.Join(root, "Vault") {
		t.Fatalf("cwd = %q", n.Dir())
	}
	n.Parent()
	n.Focus("Vault")
	act, ok := n.HandleKey("ctrl+enter", 10)
	if !ok || act != ActionSelect {
		t.Fatalf("select: act=%v ok=%v", act, ok)
	}
	path, ok := n.SelectedPath()
	if !ok || path != filepath.Join(root, "Vault") {
		t.Fatalf("SelectedPath = %q ok=%v", path, ok)
	}
	// Empty listing: Ctrl+Enter is handled but does not select.
	n.Open("Vault")
	act, ok = n.HandleKey("ctrl+enter", 10)
	if !ok || act != ActionNone {
		t.Fatalf("empty select: act=%v ok=%v", act, ok)
	}
}

func TestValidateNameConflict(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "Taken"))
	n := New(root)
	if err := ValidateName(n.Dir(), "Taken"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("conflict err = %v", err)
	}
	if err := ValidateName(n.Dir(), "a/b"); err == nil {
		t.Fatal("expected separator error")
	}
	if err := n.Mkdir("Fresh"); err != nil {
		t.Fatal(err)
	}
	n.Reload()
	if e, ok := n.Selected(); !ok || e.Name != "Fresh" && !n.Exists("Fresh") {
		n.Focus("Fresh")
	}
	if !n.Exists("Fresh") {
		t.Fatal("Fresh missing after mkdir")
	}
}

func TestStatsCache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "D")
	mustMkdir(t, dir)
	mustWrite(t, filepath.Join(dir, "f"), "x")
	n := New(root)
	first := n.Visible()[0].Bytes
	n.Reload()
	if n.Visible()[0].Bytes != first {
		t.Fatal("cache/reload changed bytes")
	}
}

func TestRowsFit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "A"))
	n := New(root)
	th := theme.New(theme.Default(), false)
	rows := n.Rows(th, 80, 5)
	if len(rows) != 5 {
		t.Fatalf("rows = %d", len(rows))
	}
	if !strings.Contains(rows[0], "NAME") {
		t.Fatalf("header = %q", rows[0])
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(ents []Entry) []string {
	out := make([]string, len(ents))
	for i, e := range ents {
		out[i] = e.Name
	}
	return out
}
