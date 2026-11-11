package dashygotui_test

// Architecture guard tests: they enforce RULES.md sections 3, 5 and 10 by
// parsing the module's source with go/parser. Each rule is a small function over
// *ast.File so it can be verified against in-memory fixtures as well as the
// real tree.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/internal/guard"
)

// source is one parsed non-test Go file of the module.
type source struct {
	rel  string // slash-separated path relative to the module root
	file *ast.File
}

func (s source) inExamples() bool { return strings.HasPrefix(s.rel, "examples/") }

func (s source) inAddons() bool { return inAddonsDir(s.rel) }

// isLibrary reports whether the file belongs to the reusable library, as
// opposed to an example application or the dashygotui command, which prints and
// exits as commands do.
func (s source) isLibrary() bool {
	return !s.inExamples() && !strings.HasPrefix(s.rel, "cmd/")
}

func inAddonsDir(rel string) bool {
	for _, seg := range strings.Split(path.Dir(rel), "/") {
		if seg == "addons" {
			return true
		}
	}
	return false
}

// skippedDirs hold no Go source that the guards apply to.
var skippedDirs = map[string]bool{
	".git": true, ".github": true, "testdata": true, "scripts": true, "vendor": true,
}

// loadSources parses every non-test .go file in the module.
func loadSources(t *testing.T) []source {
	t.Helper()
	fset := token.NewFileSet()
	var out []source
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != "." && skippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", p, err)
		}
		out = append(out, source{rel: filepath.ToSlash(p), file: f})
		return nil
	})
	if err != nil {
		t.Fatalf("load sources: %v", err)
	}
	return out
}

// --- Tests on the real tree ---

func TestLibraryDoesNotImportExamplesOrAddons(t *testing.T) {
	for _, s := range loadSources(t) {
		if !s.isLibrary() {
			continue
		}
		for _, v := range guard.LibraryImportViolations(s.rel, s.file) {
			t.Error(v)
		}
	}
}

func TestCoreIsOSAgnostic(t *testing.T) {
	for _, s := range loadSources(t) {
		if !s.isLibrary() {
			continue
		}
		for _, v := range guard.OSAgnosticViolations(s.rel, s.file) {
			t.Error(v)
		}
	}
}

func TestLibraryHasNoInitOrMutableGlobals(t *testing.T) {
	for _, s := range loadSources(t) {
		if !s.isLibrary() {
			continue
		}
		for _, v := range guard.StateViolations(s.rel, s.file) {
			t.Error(v)
		}
	}
}

func TestAddonFileNames(t *testing.T) {
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != "." && skippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel := filepath.ToSlash(p)
		if strings.HasSuffix(rel, ".go") && inAddonsDir(rel) {
			for _, v := range guard.AddonNameViolations(rel) {
				t.Error(v)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

func TestAddonsDoNotDependOnEachOther(t *testing.T) {
	byDir := map[string]map[string]*ast.File{}
	for _, s := range loadSources(t) {
		if !s.inAddons() {
			continue
		}
		dir := path.Dir(s.rel)
		if byDir[dir] == nil {
			byDir[dir] = map[string]*ast.File{}
		}
		byDir[dir][s.rel] = s.file
	}
	for _, files := range byDir {
		for _, v := range guard.CouplingViolations(files) {
			t.Error(v)
		}
	}
}

// --- Self-tests: the guards themselves, against in-memory fixtures ---

func parseFixture(t *testing.T, name, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return f
}

func TestGuardLibraryImports(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"clean", `package p; import "fmt"; var _ = fmt.Sprint`, 0},
		{"own subpackage ok", `package p; import _ "github.com/CeeSmag/dashygotui/text"`, 0},
		{"examples", `package p; import _ "github.com/CeeSmag/dashygotui/examples/demo"`, 1},
		{"addons", `package p; import _ "github.com/CeeSmag/dashygotui/examples/demo/dashboard/addons"`, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guard.LibraryImportViolations("p.go", parseFixture(t, "p.go", tt.src))
			if len(got) != tt.want {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.want)
			}
		})
	}
}

func TestGuardOSAgnostic(t *testing.T) {
	tests := []struct {
		name string
		rel  string
		src  string
		want int
	}{
		{"clean", "a.go", `package p; import "os"; var _ = os.Args`, 0},
		{"syscall", "a.go", `package p; import "syscall"; var _ = syscall.Getpid`, 1},
		{"x/sys", "a.go", `package p; import _ "golang.org/x/sys/unix"`, 1},
		{"net/http", "a.go", `package p; import _ "net/http"`, 1},
		{"os/exec elsewhere", "a.go", `package p; import _ "os/exec"`, 1},
		{"os/exec in exec.go", "exec.go", `package p; import _ "os/exec"`, 0},
		{"GOOS", "a.go", `package p; import "runtime"; var _ = runtime.GOOS`, 1},
		{"NumCPU is fine", "a.go", `package p; import "runtime"; var _ = runtime.NumCPU`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guard.OSAgnosticViolations(tt.rel, parseFixture(t, tt.rel, tt.src))
			if len(got) != tt.want {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.want)
			}
		})
	}
}

func TestGuardState(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"clean", `package p; const c = 1; type T struct{}`, 0},
		{"init", `package p; func init() {}`, 1},
		{"method named init is fine", `package p; type T struct{}; func (T) init() {}`, 0},
		{"mutable var", `package p; var counter int`, 1},
		{"map var", `package p; var m = map[string]int{}`, 1},
		{"sentinel error", `package p; import "errors"; var ErrX = errors.New("x")`, 0},
		{"wrapped sentinel", `package p; import "fmt"; var ErrX = fmt.Errorf("x")`, 0},
		{"regexp", `package p; import "regexp"; var re = regexp.MustCompile("a")`, 0},
		{"interface assertion", `package p; type I interface{}; type T struct{}; var _ I = T{}`, 0},
		{"mixed group", `package p; import "errors"; var (ErrX = errors.New("x"); n int)`, 1},
		{"embedded files", "package p\nimport \"embed\"\n//go:embed *.md\nvar FS embed.FS", 0},
		{"embed var without the directive", "package p\nimport \"embed\"\nvar FS embed.FS", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guard.StateViolations("p.go", parseFixture(t, "p.go", tt.src))
			if len(got) != tt.want {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.want)
			}
		})
	}
}

func TestGuardAddonNames(t *testing.T) {
	tests := []struct {
		rel  string
		want int
	}{
		{"examples/demo/dashboard/addons/opt_storage.go", 0},
		{"examples/demo/dashboard/addons/widget_disk_io.go", 0},
		{"examples/demo/dashboard/addons/view_logs.go", 0},
		{"examples/demo/dashboard/addons/modal_confirm.go", 0},
		{"examples/demo/dashboard/addons/shortcut_stop.go", 0},
		{"examples/demo/dashboard/addons/hotkey_x.go", 1},
		{"examples/demo/dashboard/addons/opt_storage_test.go", 0},
		{"examples/demo/dashboard/addons/storage.go", 1},
		{"examples/demo/dashboard/addons/helpers.go", 1},
		{"examples/demo/dashboard/addons/Opt_Storage.go", 1},
	}
	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			if got := guard.AddonNameViolations(tt.rel); len(got) != tt.want {
				t.Errorf("got %v, want %d violations", got, tt.want)
			}
		})
	}
}

func TestGuardCoupling(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "independent addons",
			files: map[string]string{
				"opt_a.go": `package addons; func init() { register(1) }; func register(int) {}`,
				"opt_b.go": `package addons; func init() { _ = 2 }`,
			},
			want: 0,
		},
		{
			name: "calls sibling function",
			files: map[string]string{
				"opt_a.go": `package addons; func helper() int { return 1 }`,
				"opt_b.go": `package addons; func init() { _ = helper() }`,
			},
			want: 1,
		},
		{
			name: "uses sibling type",
			files: map[string]string{
				"opt_a.go": `package addons; type Shared struct{}`,
				"opt_b.go": `package addons; var _ Shared`,
			},
			want: 1,
		},
		{
			name: "local variable shadows sibling name",
			files: map[string]string{
				"opt_a.go": `package addons; func render() {}`,
				"opt_b.go": `package addons; func init() { render := 1; _ = render }`,
			},
			want: 0,
		},
		{
			name: "struct literal key equal to sibling name is a field, not a use",
			files: map[string]string{
				"opt_a.go": `package addons; func Label() {}`,
				"opt_b.go": `package addons; type O struct{ Label string }; var _ = O{Label: "x"}`,
			},
			want: 0,
		},
		{
			name: "imported package name equal to sibling name",
			files: map[string]string{
				"opt_a.go": `package addons; func fmt() {}`,
				"opt_b.go": `package addons; import "fmt"; var _ = fmt.Sprint`,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]*ast.File{}
			for name, src := range tt.files {
				files[name] = parseFixture(t, name, src)
			}
			if got := guard.CouplingViolations(files); len(got) != tt.want {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.want)
			}
		})
	}
}
