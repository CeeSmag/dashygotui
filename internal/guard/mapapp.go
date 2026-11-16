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

// MapKinds are the kinds of feature file, the last part of its name.
func MapKinds() []string {
	return []string{"opt", "widget", "view", "page", "modal", "shortcut", "hook"}
}

var mapFileName = regexp.MustCompile(`^[a-z][a-z0-9_]*_(opt|widget|view|page|modal|shortcut|hook)(_test)?\.go$`)

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
	var features []Source
	var mapFile Source
	for _, s := range files {
		base := path.Base(s.Rel)
		switch {
		case base == MapFile:
			mapFile = s
		case IsFeatureFile(s.Rel):
			features = append(features, s)
		case base == GenFile || base == DocFile:
			out = append(out, Finding{Rule: "A1", File: s.Rel, Line: 1,
				Msg:  base + " belongs to the v0.4 layout: the map in " + MapFile + " lists the addons now",
				Hint: "delete it; put the package comment in " + MapFile})
		default:
			out = append(out, Finding{Rule: "A1", File: s.Rel, Line: 1,
				Msg:  "a feature file is named <name>_<kind>.go (kinds: " + strings.Join(MapKinds(), ", ") + "); the map is " + MapFile,
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

	// A13: every addon of a feature file is placed in the map.
	if mapFile.File == nil {
		out = append(out, Finding{Rule: "A13", File: dir + "/" + MapFile, Line: 1,
			Msg:  "the addons package has no map, " + MapFile + ", so nothing in it is placed",
			Hint: "write " + MapFile + " with func Map() dashboard.Map (see dashygotui rules design)"})
	} else {
		used := map[string]int{}
		ast.Inspect(mapFile.File, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				used[id.Name]++
			}
			return true
		})
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
								Hint: "add it to a panel, Shortcuts, Hooks or Views in " + MapFile + ", or delete the file"})
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
