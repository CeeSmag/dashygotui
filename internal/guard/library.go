// Package guard holds the source-level rules of dashygotui: the architecture
// rules the library obeys (RULES.md) and the rules an application's addons obey
// (docs/APP-RULES.md). Every rule is a small function over parsed Go source, so
// it can be tested against in-memory fixtures and run both by the library's own
// guard tests and by the dashygotui command on an application.
package guard

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func ImportPaths(f *ast.File) []string {
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

func LibraryImportViolations(rel string, f *ast.File) []string {
	var out []string
	for _, p := range ImportPaths(f) {
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

// ExecFile is the single library file allowed to import os/exec.
const ExecFile = "exec.go"

func OSAgnosticViolations(rel string, f *ast.File) []string {
	var out []string
	for _, p := range ImportPaths(f) {
		switch {
		case p == "syscall", strings.HasPrefix(p, "golang.org/x/sys"), p == "net/http":
			out = append(out, fmt.Sprintf("%s imports %q: the core must be OS-agnostic", rel, p))
		case p == "os/exec" && rel != ExecFile:
			out = append(out, fmt.Sprintf("%s imports os/exec: only %s may", rel, ExecFile))
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
func readOnlyCtor(call string) bool {
	switch call {
	case "errors.New", "fmt.Errorf", "regexp.MustCompile":
		return true
	}
	return false
}

func CalleeName(e ast.Expr) string {
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

func StateViolations(rel string, f *ast.File) []string {
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
				if allReadOnly(vs) || embedsFiles(d, vs) {
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
		if !readOnlyCtor(CalleeName(v)) {
			return false
		}
	}
	return true
}

// --- Rule: addon file names (section 10) ---

var addonFileName = regexp.MustCompile(`^(opt|widget|view|modal|shortcut)_[a-z0-9_]+(_test)?\.go$`)

func AddonNameViolations(rel string) []string {
	base := path.Base(rel)
	if addonFileName.MatchString(base) {
		return nil
	}
	return []string{fmt.Sprintf("%s: addon files must be named opt_*.go, widget_*.go, view_*.go, modal_*.go or shortcut_*.go", rel)}
}

// --- Rule: addons never depend on each other (sections 3 and 10) ---

// CouplingViolations reports every use, in one addon file, of a package-level
// identifier that is declared in a different addon file of the same package.
func CouplingViolations(files map[string]*ast.File) []string {
	var out []string
	for _, v := range coupling(files, func(string, *ast.Ident) int { return 0 }) {
		out = append(out, fmt.Sprintf("%s uses %s declared in %s: addons must not depend on each other", v.file, v.name, v.owner))
	}
	sort.Strings(out)
	return out
}

// couple is one use of a sibling addon's declaration.
type couple struct {
	file, name, owner string
	line              int
}

// coupling finds every use, in one file, of a package-level identifier that is
// declared in a different file of the set. lineOf gives the line of a use.
func coupling(files map[string]*ast.File, lineOf func(rel string, id *ast.Ident) int) []couple {
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

	var out []couple
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
				out = append(out, couple{file: rel, name: id.Name, owner: o, line: lineOf(rel, id)})
			}
		}
	}
	return out
}

// inAddonsDir reports whether rel is in a directory named addons, or below one.
func inAddonsDir(rel string) bool {
	for _, seg := range strings.Split(path.Dir(rel), "/") {
		if seg == "addons" {
			return true
		}
	}
	return false
}

// embedsFiles reports whether a package-level var is filled by a //go:embed
// directive, which makes it read-only.
func embedsFiles(d *ast.GenDecl, vs *ast.ValueSpec) bool {
	for _, cg := range []*ast.CommentGroup{vs.Doc, d.Doc} {
		if cg == nil {
			continue
		}
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "//go:embed ") {
				return true
			}
		}
	}
	return false
}
