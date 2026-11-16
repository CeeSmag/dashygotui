package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const genState = "example.com/app/dashboard.State"

func valueApp(t *testing.T) (root, addons string) {
	t.Helper()
	root = t.TempDir()
	addons = filepath.Join(root, "dashboard", "addons")
	write(t, root, "go.mod", "module example.com/app\n\ngo 1.27\n")
	write(t, root, "dashboard/addons/opt_backup.go", "package addons\n\nimport \"example.com/app/dashboard\"\n\nvar backup = dashboard.Option{Key: \"backup\"}\n")
	return root, addons
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGenWritesTheListAndADocFile(t *testing.T) {
	_, addons := valueApp(t)
	code, out, errOut := runCLI(t, "gen", "-state", genState, addons)
	if code != exitOK {
		t.Fatalf("gen: %d %s %s", code, out, errOut)
	}
	list := read(t, filepath.Join(addons, "addons_gen.go"))
	if !strings.Contains(list, `dashygotui.At[dashboard.State]("opt_backup.go:5", backup)`) {
		t.Fatalf("list:\n%s", list)
	}
	doc := read(t, filepath.Join(addons, "doc.go"))
	if !strings.Contains(doc, "//go:generate go tool dashygotui gen -state "+genState) || !strings.Contains(doc, "package addons") {
		t.Fatalf("doc.go:\n%s", doc)
	}

	// Run again with no -state: it is read from the list, and nothing changes.
	if code, out, _ := runCLI(t, "gen", "-q", addons); code != exitOK || out != "" {
		t.Fatalf("second gen: %d %q", code, out)
	}
	// The list it wrote passes the check.
	if code, out, _ := runCLI(t, "check", "-q", filepath.Dir(filepath.Dir(addons))); code != exitOK {
		t.Fatalf("check after gen: %d\n%s", code, out)
	}
}

func TestGenAllRegeneratesEveryList(t *testing.T) {
	root, addons := valueApp(t)
	if code, _, errOut := runCLI(t, "gen", "-state", genState, addons); code != exitOK {
		t.Fatal(errOut)
	}
	write(t, root, "dashboard/addons/opt_restart.go", "package addons\n\nimport \"example.com/app/dashboard\"\n\nvar restart = dashboard.Option{Key: \"restart\"}\n")
	if code, out, _ := runCLI(t, "check", "-q", root); code != exitFindings || !strings.Contains(out, "A11") {
		t.Fatalf("a stale list passed the check: %d\n%s", code, out)
	}
	if code, out, errOut := runCLI(t, "gen", "-all", root); code != exitOK || !strings.Contains(out, "wrote") {
		t.Fatalf("gen -all: %d %q %q", code, out, errOut)
	}
	if !strings.Contains(read(t, filepath.Join(addons, "addons_gen.go")), "restart") {
		t.Fatal("gen -all did not pick up the new addon")
	}
	if err := os.Remove(filepath.Join(addons, "opt_backup.go")); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI(t, "gen", "-all", "-q", root); code != exitOK {
		t.Fatal("gen -all after a deletion")
	}
	if strings.Contains(read(t, filepath.Join(addons, "addons_gen.go")), "backup") {
		t.Fatal("a deleted addon is still listed")
	}
}

func TestGenRejectsBadInput(t *testing.T) {
	_, addons := valueApp(t)
	for _, args := range [][]string{
		{"gen", addons},                              // no -state and no list to read it from
		{"gen", "-state", "nonsense", addons},        // not <path>.<Type>
		{"gen", "-all", "-state", genState, addons},  // -all reads each list's own state
		{"gen", "-state", genState, addons, "extra"}, // one directory
	} {
		if code, _, _ := runCLI(t, args...); code == exitOK {
			t.Errorf("%v succeeded", args)
		}
	}
}
