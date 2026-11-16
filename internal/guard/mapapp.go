package guard

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The rules for an addons package that is laid out around a map (docs/DESIGN.md,
// "the map"): the package's addons.go places every addon, and each other file is
// one self-contained feature named <name>_<kind>.go.

// MapFile is the hand-written map of an addons package: the only file that
// places anything.
const MapFile = "addons.go"

// GenFile and DocFile are the generated addon list and its doc.go of the v0.4
// layout, which the map replaced. Left in an addons package, they are A1.
const (
	GenFile = "addons_gen.go"
	DocFile = "doc.go"
)

// libraryPath is the import path of the library.
const libraryPath = "github.com/CeeSmag/dashygotui"

// MapKinds are the kinds of feature file, the last part of its name. A subpage
// (<name>_subpage.go) is a window opened from a row; what it holds is named after
// it, <subpage>_<name>_sub<kind>.go.
func MapKinds() []string {
	return []string{"opt", "widget", "view", "page", "modal", "shortcut", "hook",
		"subpage", "subopt", "subwidget", "subview", "submodal"}
}

var mapFileName = regexp.MustCompile(`^[a-z][a-z0-9_]*_(opt|widget|view|page|modal|shortcut|hook|subpage|subopt|subwidget|subview|submodal)(_test)?\.go$`)

// featureKind splits a feature file's base name into its name and kind:
// "diag_logs_subopt.go" is ("diag_logs", "subopt").
func featureKind(base string) (name, kind string) {
	stem := strings.TrimSuffix(base, ".go")
	i := strings.LastIndex(stem, "_")
	if i < 0 {
		return stem, ""
	}
	return stem[:i], stem[i+1:]
}

var mapPartName = regexp.MustCompile(`^[a-z][a-z0-9_]*_map\.go$`)

// IsMapFile reports whether rel is part of an addons package's map: addons.go,
// the root, or a <name>_map.go file that maps one page or window.
func IsMapFile(rel string) bool {
	b := path.Base(rel)
	return b == MapFile || mapPartName.MatchString(b)
}

// IsFeatureFile reports whether rel is a feature file of a map-laid-out addons
// package by its name, such as backup_opt.go.
func IsFeatureFile(rel string) bool {
	return mapFileName.MatchString(path.Base(rel)) && !strings.HasSuffix(rel, "_test.go")
}

// mapFindings applies the package rules of a map-laid-out addons package: A1
// (file names), A2 (a feature file uses nothing of the application but the
// dashboard package's types) and A13 (every addon is placed in the map). all
// are every source loaded, from which the dashboard package's types are read.
func mapFindings(dir string, files, all []Source) []Finding {
	var out []Finding
	var features, maps []Source
	var mapFile Source
	for _, s := range files {
		base := path.Base(s.Rel)
		switch {
		case IsMapFile(s.Rel):
			maps = append(maps, s)
			if base == MapFile {
				mapFile = s
			}
		case IsFeatureFile(s.Rel):
			features = append(features, s)
		case base == GenFile || base == DocFile:
			out = append(out, Finding{Rule: "A1", File: s.Rel, Line: 1,
				Msg:  base + " belongs to the v0.4 layout: the map in " + MapFile + " lists the addons now",
				Hint: "delete it; put the package comment in " + MapFile})
		default:
			out = append(out, Finding{Rule: "A1", File: s.Rel, Line: 1,
				Msg:  "a feature file is named <name>_<kind>.go (kinds: " + strings.Join(MapKinds(), ", ") + "); the map is " + MapFile + " and <name>_map.go files",
				Hint: "rename the file, for example " + strings.TrimSuffix(base, ".go") + "_opt.go"})
		}
	}

	// A2: features do not use each other.
	if len(features) > 0 {
		out = append(out, couplingFindingsHint(features,
			"a feature is self-contained: make what it shares a method of State or a library feature, or repeat it")...)
	}

	// A2: from the application, a feature uses only the dashboard package's types.
	dashDir := path.Dir(dir)
	types := map[string]bool{}
	dashPkg := ""
	for _, s := range all {
		if dirOf(s.Rel) != dashDir {
			continue
		}
		if s.Pkg != "" {
			dashPkg = s.Pkg
		}
		for _, d := range s.File.Decls {
			if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					types[spec.(*ast.TypeSpec).Name.Name] = true
				}
			}
		}
	}
	if dashPkg == "" && len(features) > 0 && features[0].Pkg != "" {
		dashPkg = path.Dir(features[0].Pkg)
	}
	for _, s := range features {
		out = append(out, featureImportFindings(s, dashPkg, types)...)
	}

	out = append(out, subpageFindings(features, maps)...)

	// A13: every addon of a feature file is placed in the map.
	if mapFile.File == nil {
		out = append(out, Finding{Rule: "A13", File: dir + "/" + MapFile, Line: 1,
			Msg:  "the addons package has no map, " + MapFile + ", so nothing in it is placed",
			Hint: "write " + MapFile + " with func Map() dashboard.Map (see dashygotui rules design)"})
	} else {
		used := map[string]int{}
		for _, m := range maps {
			ast.Inspect(m.File, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					used[id.Name]++
				}
				return true
			})
		}
		for _, s := range features {
			for _, d := range s.File.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					for _, id := range spec.(*ast.ValueSpec).Names {
						switch {
						case id.Name == "_":
						case used[id.Name] == 0:
							out = append(out, Finding{Rule: "A13", File: s.Rel, Line: s.Line(id.Pos()),
								Msg:  id.Name + " is not placed in the map, so it never appears",
								Hint: "add it to a panel, Shortcuts, Hooks or Views in the map, or delete the file"})
						case used[id.Name] > 1:
							out = append(out, Finding{Rule: "A13", File: s.Rel, Line: s.Line(id.Pos()),
								Msg:  id.Name + " is placed in the map more than once",
								Hint: "place each feature once; a second way into a view is an option with Opens"})
						}
					}
				}
			}
		}
	}
	return out
}

// featureImportFindings reports what a feature file uses of the application
// beyond the dashboard package's types: another application package, or a
// function, variable or constant of the dashboard package. dashPkg is the
// dashboard package's import path ("" when unknown); types are the names it
// declares as types (nil when its files were not loaded).
func featureImportFindings(s Source, dashPkg string, types map[string]bool) []Finding {
	var out []Finding
	dashNames := map[string]bool{}
	app := s.Module
	if app == libraryPath && dashPkg != "" {
		app = path.Dir(dashPkg) // an example inside the library's own module
	}
	for _, imp := range s.File.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		switch {
		case dashPkg != "" && p == dashPkg:
			name := path.Base(p)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			dashNames[name] = true
		case app != "" && (p == app || strings.HasPrefix(p, app+"/")):
			out = append(out, Finding{Rule: "A2", File: s.Rel, Line: s.Line(imp.Pos()),
				Msg:  "imports " + p + ": a feature uses nothing of the application but the dashboard package's types",
				Hint: "reach the application through State (its fields and methods); place shared settings in the map"})
		}
	}
	if len(dashNames) == 0 || len(types) == 0 {
		return out
	}
	ast.Inspect(s.File, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Obj == nil && dashNames[id.Name] && !types[sel.Sel.Name] {
			out = append(out, Finding{Rule: "A2", File: s.Rel, Line: s.Line(sel.Pos()),
				Msg:  fmt.Sprintf("uses %s.%s: from the dashboard package a feature uses only types", id.Name, sel.Sel.Name),
				Hint: "make it a field or method of State, or move what looks generic into the library (record a gap)"})
		}
		return true
	})
	return out
}

var moduleLine = regexp.MustCompile(`(?m)^module\s+(\S+)`)

// moduleOf finds the go.mod at or above dir and returns its module path and
// directory, or "" when there is none.
func moduleOf(dir string) (module, modDir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", ""
	}
	for d := abs; ; d = filepath.Dir(d) {
		if b, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			if m := moduleLine.FindSubmatch(b); m != nil {
				return string(m[1]), d
			}
			return "", ""
		}
		if filepath.Dir(d) == d {
			return "", ""
		}
	}
}

// importPathOf is the import path of the package in directory dir, a file of
// which is at rel under root, given the module.
func importPathOf(module, modDir, root, rel string) string {
	if module == "" {
		return ""
	}
	abs, err := filepath.Abs(filepath.Join(root, filepath.Dir(rel)))
	if err != nil {
		return ""
	}
	r, err := filepath.Rel(modDir, abs)
	if err != nil || strings.HasPrefix(r, "..") {
		return ""
	}
	if r == "." {
		return module
	}
	return module + "/" + filepath.ToSlash(r)
}

// subpageFindings applies A14 to the features and map files of an addons
// package. A subpage is <name>_subpage.go: the option that opens a window.
// Every element of the window is named after it, <name>_<element>_sub<kind>.go,
// and the map places the elements of a subpage inside its window and nowhere
// else, and nothing else there; a window opens from a subpage's option.
func subpageFindings(features, maps []Source) []Finding {
	var out []Finding
	subpages := map[string]bool{} // subpage names
	for _, s := range features {
		if name, kind := featureKind(path.Base(s.Rel)); kind == "subpage" {
			subpages[name] = true
		}
	}
	// parentOf is the subpage a name belongs to: the longest subpage name it
	// starts with, followed by "_".
	parentOf := func(name string) string {
		best := ""
		for sp := range subpages {
			if strings.HasPrefix(name, sp+"_") && len(sp) > len(best) {
				best = sp
			}
		}
		return best
	}
	owner := map[string]string{}    // feature var -> the window it belongs in ("" for a tab)
	openerOf := map[string]string{} // a subpage's var -> the subpage it opens
	for _, s := range features {
		name, kind := featureKind(path.Base(s.Rel))
		place := ""
		switch {
		case kind == "subpage":
			place = parentOf(name)
		case strings.HasPrefix(kind, "sub"):
			place = parentOf(name)
			if place == "" {
				out = append(out, Finding{Rule: "A14", File: s.Rel, Line: 1,
					Msg:  path.Base(s.Rel) + " is an element of a subpage, but no <subpage>_subpage.go starts its name",
					Hint: "name it <subpage>_<name>_" + kind + ".go after its subpage, or make it a " + strings.TrimPrefix(kind, "sub")})
				continue
			}
		}
		for _, d := range s.File.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				for _, id := range spec.(*ast.ValueSpec).Names {
					owner[id.Name] = place
					if kind == "subpage" {
						openerOf[id.Name] = name
					}
				}
			}
		}
	}

	for _, m := range maps {
		at := func(n ast.Node, msg, hint string) {
			out = append(out, Finding{Rule: "A14", File: m.Rel, Line: m.Line(n.Pos()), Msg: msg, Hint: hint})
		}
		check := func(id *ast.Ident, window string) {
			want, ok := owner[id.Name]
			if !ok || want == window {
				return
			}
			switch {
			case want == "":
				at(id, id.Name+" is placed inside the "+window+" subpage, but it is not one of its elements",
					"name its file "+window+"_<name>_sub<kind>.go, or place it outside the subpage")
			case window == "":
				at(id, id.Name+" is an element of the "+want+" subpage, placed outside it",
					"place it inside the window whose Row is the "+want+" subpage")
			default:
				at(id, id.Name+" is an element of the "+want+" subpage, placed inside "+window,
					"place it inside the "+want+" subpage, or rename its file")
			}
		}
		var walk func(n ast.Node, window string, nested bool)
		walk = func(n ast.Node, window string, nested bool) {
			ast.Inspect(n, func(x ast.Node) bool {
				switch x := x.(type) {
				case *ast.CompositeLit:
					if !isPageType(x.Type) {
						return true
					}
					inner := window
					var row *ast.Ident
					for _, e := range x.Elts {
						if kv, ok := e.(*ast.KeyValueExpr); ok && isIdent(kv.Key, "Row") {
							row, _ = kv.Value.(*ast.Ident)
							if row == nil || openerOf[row.Name] == "" {
								at(kv, "a window opens from a subpage: its Row is the option of a <name>_subpage.go",
									"declare the option in <name>_subpage.go")
							} else {
								inner = openerOf[row.Name]
							}
						}
					}
					if row != nil && nested {
						check(row, window) // the row lives where the window is placed
					}
					for _, e := range x.Elts {
						if kv, ok := e.(*ast.KeyValueExpr); ok && !isIdent(kv.Key, "Row") {
							walk(kv.Value, inner, true)
						}
					}
					return false
				case *ast.Ident:
					check(x, window)
				}
				return true
			})
		}
		for _, d := range m.File.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Body != nil {
				walk(fd.Body, "", false)
			}
		}
	}
	return out
}

// isPageType reports whether a composite literal's type is a Page: Page,
// dashboard.Page or dashygotui.Page[S].
func isPageType(t ast.Expr) bool {
	switch t := t.(type) {
	case *ast.Ident:
		return t.Name == "Page"
	case *ast.SelectorExpr:
		return t.Sel.Name == "Page"
	case *ast.IndexExpr:
		return isPageType(t.X)
	}
	return false
}
