package guard_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/internal/guard"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadParsesNonTestSourcesAndSkipsTheRest(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"a.go":          "package a\n",
		"a_test.go":     "package a\n",
		"sub/b.go":      "package sub\n",
		"vendor/v.go":   "package v\n",
		".git/g.go":     "package g\n",
		"testdata/t.go": "package t\n",
		"skipme/s.go":   "package s\n",
		"notes.txt":     "not go",
	})
	srcs, err := guard.Load(dir, "skipme")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range srcs {
		got = append(got, s.Rel)
	}
	if strings.Join(got, ",") != "a.go,sub/b.go" {
		t.Errorf("loaded %v, want only a.go and sub/b.go", got)
	}
	if srcs[0].Line(srcs[0].File.Pos()) != 1 {
		t.Error("Line should resolve positions")
	}
}

func TestLoadReportsASyntaxError(t *testing.T) {
	dir := writeTree(t, map[string]string{"bad.go": "package a\nfunc ("})
	if _, err := guard.Load(dir); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("err = %v", err)
	}
}

func TestAPISnapshotListsOnlyTheExportedSurface(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"go.mod": "module example.com/m\n",
		"m.go": `package m

const Version = "1"
var ErrX error
type Config[S any] struct {
	Title string
	hidden int
	Poll func(S) error
}
type Shower interface{ Show() string; hide() }
type ID = string
type Count int
func New[S any](c Config[S]) *Config[S] { return nil }
func (c *Config[S]) Run() error { return nil }
func (c *Config[S]) stop() {}
func helper() {}
type unexported struct{ Field int }
func (unexported) Exported() {}
`,
		"sub/s.go":        "package sub\nfunc Hello() {}\n",
		"internal/i/i.go": "package i\nfunc Hidden() {}\n",
		"examples/e/e.go": "package e\nfunc Example() {}\n",
		"m_test.go":       "package m\nfunc TestOnly() {}\n",
	})
	got, err := guard.APISnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"m: const Version", "m: var ErrX error", "m: type Config[S any] struct", "m: field Config.Title string",
		"m: field Config.Poll func(S) error", "m: type Shower interface", "m: method Shower.Show func() string",
		"m: type ID = string", "m: type Count int", "m: func New[S any](c Config[S]) *Config[S]",
		"m: func (c *Config[S]) Run() error", "sub: func Hello()",
	} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("snapshot lacks %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"hidden", "stop", "helper", "unexported", "Hidden", "Example", "TestOnly", "hide"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("snapshot must not mention %q:\n%s", unwanted, got)
		}
	}
	again, _ := guard.APISnapshot(dir)
	if again != got {
		t.Error("the snapshot must be deterministic")
	}
}

func TestFieldReferenceAndExportedFields(t *testing.T) {
	dir := writeTree(t, map[string]string{"m.go": `package m

// Thing is a thing.
// It has two lines.
type Thing struct {
	// Name says | what it is.
	// Required.
	Name string
	// Size is how big.
	Size int
	hidden bool
}
`})
	ref, err := guard.FieldReference(dir, "Thing")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"### `Thing`", "Thing is a thing. It has two lines.", "| `Name` | `string` | Name says \\| what it is. Required. |", "| `Size` | `int` | Size is how big. |"} {
		if !strings.Contains(ref, want) {
			t.Errorf("reference lacks %q:\n%s", want, ref)
		}
	}
	if strings.Contains(ref, "hidden") {
		t.Error("unexported fields must not appear")
	}
	if _, err := guard.FieldReference(dir, "Nope"); err == nil {
		t.Error("an unknown type is an error")
	}
	fields, err := guard.ExportedFields(dir, "Thing")
	if err != nil || strings.Join(fields["Thing"], ",") != "Name,Size" {
		t.Errorf("fields = %v, err %v", fields, err)
	}
}

func TestVocabularyViolations(t *testing.T) {
	got := guard.VocabularyViolations("f.go", []byte("ok\n// for CloudVault only\nx := \"sudo\"\n"), guard.ApplicationVocabulary())
	if len(got) != 2 || !strings.HasPrefix(got[0], "f.go:2:") || !strings.HasPrefix(got[1], "f.go:3:") {
		t.Errorf("violations = %v", got)
	}
}

func parseFile(t *testing.T, name, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestLibraryRulesOnFixtures(t *testing.T) {
	f := parseFile(t, "a.go", `package p
import (
	"os/exec"
	"runtime"
	_ "github.com/CeeSmag/dashygotui/examples/demo/dashboard/addons"
)
var counter int
var _ = runtime.GOOS
func init() {}
`)
	if got := guard.ImportPaths(f); len(got) != 3 {
		t.Errorf("ImportPaths = %v", got)
	}
	if got := guard.LibraryImportViolations("a.go", f); len(got) != 2 { // examples and addons
		t.Errorf("LibraryImportViolations = %v", got)
	}
	if got := guard.OSAgnosticViolations("a.go", f); len(got) != 2 { // os/exec outside exec.go, runtime.GOOS
		t.Errorf("OSAgnosticViolations = %v", got)
	}
	if got := guard.OSAgnosticViolations(guard.ExecFile, parseFile(t, "exec.go", "package p\nimport _ \"os/exec\"")); len(got) != 0 {
		t.Errorf("exec.go may import os/exec: %v", got)
	}
	if got := guard.StateViolations("a.go", f); len(got) != 2 { // counter, init
		t.Errorf("StateViolations = %v", got)
	}
	if got := guard.AddonNameViolations("x/addons/helpers.go"); len(got) != 1 {
		t.Errorf("AddonNameViolations = %v", got)
	}
	files := map[string]*ast.File{
		"opt_a.go": parseFile(t, "opt_a.go", "package a\nfunc helper() {}"),
		"opt_b.go": parseFile(t, "opt_b.go", "package a\nfunc init() { helper() }"),
	}
	if got := guard.CouplingViolations(files); len(got) != 1 {
		t.Errorf("CouplingViolations = %v", got)
	}
}
