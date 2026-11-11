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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// source is one parsed non-test Go file of the module.
type source struct {
	rel  string // slash-separated path relative to the module root
	file *ast.File
}

func (s source) inExamples() bool { return strings.HasPrefix(s.rel, "examples/") }

func (s source) inAddons() bool { return inAddonsDir(s.rel) }

// isLibrary reports whether the file belongs to the reusable library, as
// opposed to an example application.
func (s source) isLibrary() bool { return !s.inExamples() }

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
	".git": true, ".github": true, "testdata": true, "templates": true,
	"docs": true, "scripts": true, "vendor": true,
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

func importPaths(f *ast.File) []string {
	var out []string
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err == nil {
			out = append(out, p)
		}
	}
	return out
}

// --- Rule: the library never imports examples or addons (RULES.md section 3) ---

func libraryImportViolations(rel string, f *ast.File) []string {
	var out []string
	for _, p := range importPaths(f) {
		if !strings.HasPrefix(p, "github.com/CeeSmag/dashygotui") {
			continue
		}
		sub := strings.TrimPrefix(p, "github.com/CeeSmag/dashygotui")
		if strings.HasPrefix(sub, "/examples") {
			out = append(out, fmt.Sprintf("%s imports %q: the library must not import examples", rel, p))
		}
		for _, seg := range strings.Split(sub, "/") {
			if seg == "addons" {
				out = append(out, fmt.Sprintf("%s imports %q: the library must not import addons", rel, p))
			}
		}
	}
	return out
}

// --- Rule: the core is OS-agnostic (RULES.md section 3) ---

// execFile is the single library file allowed to import os/exec.
const execFile = "exec.go"

func osAgnosticViolations(rel string, f *ast.File) []string {
	var out []string
	for _, p := range importPaths(f) {
		switch {
		case p == "syscall", strings.HasPrefix(p, "golang.org/x/sys"), p == "net/http":
			out = append(out, fmt.Sprintf("%s imports %q: the core must be OS-agnostic", rel, p))
		case p == "os/exec" && rel != execFile:
			out = append(out, fmt.Sprintf("%s imports os/exec: only %s may", rel, execFile))
		}
	}
	runtimeName := "runtime"
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == "runtime" && imp.Name != nil {
			runtimeName = imp.Name.Name
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if ok && id.Name == runtimeName && (sel.Sel.Name == "GOOS" || sel.Sel.Name == "GOARCH") {
			out = append(out, fmt.Sprintf("%s uses runtime.%s: the core must be OS-agnostic", rel, sel.Sel.Name))
		}
		return true
	})
	return out
}

// --- Rule: no init() and no mutable package state in the library (section 5) ---

// readOnlyCtors are the only calls a package-level var in the library may be
// initialised with: sentinel errors and compiled regexps.
var readOnlyCtors = map[string]bool{
	"errors.New":         true,
	"fmt.Errorf":         true,
	"regexp.MustCompile": true,
}

func calleeName(e ast.Expr) string {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name + "." + sel.Sel.Name
}

func stateViolations(rel string, f *ast.File) []string {
	var out []string
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "init" {
				out = append(out, fmt.Sprintf("%s declares init(): the library must not use init()", rel))
			}
		case *ast.GenDecl:
			if d.Tok != token.VAR {
				continue
			}
			for _, spec := range d.Specs {
				vs := spec.(*ast.ValueSpec)
				if allReadOnly(vs) {
					continue
				}
				out = append(out, fmt.Sprintf("%s declares package-level var %s: the library must not hold mutable package state",
					rel, joinNames(vs.Names)))
			}
		}
	}
	return out
}

func joinNames(ids []*ast.Ident) string {
	names := make([]string, len(ids))
	for i, id := range ids {
		names[i] = id.Name
	}
	return strings.Join(names, ", ")
}

// allReadOnly reports whether a package-level var spec is a blank interface
// assertion or is initialised only from read-only constructors.
func allReadOnly(vs *ast.ValueSpec) bool {
	blank := true
	for _, id := range vs.Names {
		if id.Name != "_" {
			blank = false
		}
	}
	if blank {
		return true
	}
	if len(vs.Values) == 0 {
		return false
	}
	for _, v := range vs.Values {
		if !readOnlyCtors[calleeName(v)] {
			return false
		}
	}
	return true
}

// --- Rule: addon file names (section 10) ---

var addonFileName = regexp.MustCompile(`^(opt|widget|view|modal|shortcut)_[a-z0-9_]+(_test)?\.go$`)

func addonNameViolations(rel string) []string {
	base := path.Base(rel)
	if addonFileName.MatchString(base) {
		return nil
	}
	return []string{fmt.Sprintf("%s: addon files must be named opt_*.go, widget_*.go, view_*.go, modal_*.go or shortcut_*.go", rel)}
}

// --- Rule: addons never depend on each other (sections 3 and 10) ---

// couplingViolations reports every use, in one addon file, of a package-level
// identifier that is declared in a different addon file of the same package.
func couplingViolations(files map[string]*ast.File) []string {
	owner := map[string]string{}
	for rel, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.Name != "init" && d.Name.Name != "_" {
					owner[d.Name.Name] = rel
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						owner[s.Name.Name] = rel
					case *ast.ValueSpec:
						for _, id := range s.Names {
							if id.Name != "_" {
								owner[id.Name] = rel
							}
						}
					}
				}
			}
		}
	}

	var out []string
	for rel, f := range files {
		imported := map[string]bool{}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			name := path.Base(p)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			imported[name] = true
		}
		// Unresolved lists identifiers the parser could not bind inside this
		// file: exactly the references that leave the file.
		for _, id := range f.Unresolved {
			if imported[id.Name] {
				continue
			}
			if o, ok := owner[id.Name]; ok && o != rel {
				out = append(out, fmt.Sprintf("%s uses %s declared in %s: addons must not depend on each other", rel, id.Name, o))
			}
		}
	}
	sort.Strings(out)
	return out
}

// --- Tests on the real tree ---

func TestLibraryDoesNotImportExamplesOrAddons(t *testing.T) {
	for _, s := range loadSources(t) {
		if !s.isLibrary() {
			continue
		}
		for _, v := range libraryImportViolations(s.rel, s.file) {
			t.Error(v)
		}
	}
}

func TestCoreIsOSAgnostic(t *testing.T) {
	for _, s := range loadSources(t) {
		if !s.isLibrary() {
			continue
		}
		for _, v := range osAgnosticViolations(s.rel, s.file) {
			t.Error(v)
		}
	}
}

func TestLibraryHasNoInitOrMutableGlobals(t *testing.T) {
	for _, s := range loadSources(t) {
		if !s.isLibrary() {
			continue
		}
		for _, v := range stateViolations(s.rel, s.file) {
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
			for _, v := range addonNameViolations(rel) {
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
		for _, v := range couplingViolations(files) {
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
			got := libraryImportViolations("p.go", parseFixture(t, "p.go", tt.src))
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
			got := osAgnosticViolations(tt.rel, parseFixture(t, tt.rel, tt.src))
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stateViolations("p.go", parseFixture(t, "p.go", tt.src))
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
			if got := addonNameViolations(tt.rel); len(got) != tt.want {
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
			if got := couplingViolations(files); len(got) != tt.want {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.want)
			}
		})
	}
}
