package guard

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// apiSkip are directories that are not part of the library's public API.
func apiSkip(name string) bool {
	switch name {
	case ".git", ".github", "internal", "examples", "cmd", "templates", "docs", "scripts", "api", "testdata", "bin", "vendor":
		return true
	}
	return false
}

var spaceRe = regexp.MustCompile(`\s+`)

// APISnapshot lists the exported API of every public package under root, one
// sorted line per identifier, field and method. It is the text kept in
// api/dashygotui.txt: any change to the public API shows up as a diff of it.
func APISnapshot(root string) (string, error) {
	pkgs := map[string][]string{} // directory -> files
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && apiSkip(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			pkgs[filepath.Dir(p)] = append(pkgs[filepath.Dir(p)], p)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking %s: %w", root, err)
	}

	var lines []string
	for dir, files := range pkgs {
		fset := token.NewFileSet()
		label := ""
		for _, fn := range files {
			f, err := parser.ParseFile(fset, fn, nil, 0)
			if err != nil {
				return "", fmt.Errorf("parsing %s: %w", fn, err)
			}
			if label == "" {
				rel, _ := filepath.Rel(root, dir)
				label = f.Name.Name
				if rel != "." {
					label = filepath.ToSlash(rel)
				}
			}
			lines = append(lines, declLines(fset, label, f)...)
		}
	}
	sort.Strings(lines)
	return "# The exported API of dashygotui. Generated: go test -run TestAPISnapshot -update\n" +
		strings.Join(lines, "\n") + "\n", nil
}

// render prints a node on one line.
func render(fset *token.FileSet, n any) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return "?"
	}
	return strings.TrimSpace(spaceRe.ReplaceAllString(buf.String(), " "))
}

func declLines(fset *token.FileSet, pkg string, f *ast.File) []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, pkg+": "+fmt.Sprintf(format, args...)) }
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			if d.Recv != nil && !exportedRecv(d.Recv) {
				continue
			}
			c := *d
			c.Body, c.Doc = nil, nil
			add("%s", render(fset, &c))
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, id := range s.Names {
						if id.IsExported() {
							typ := ""
							if s.Type != nil {
								typ = " " + render(fset, s.Type)
							}
							add("%s %s%s", strings.ToLower(d.Tok.String()), id.Name, typ)
						}
					}
				case *ast.TypeSpec:
					if !s.Name.IsExported() {
						continue
					}
					name := s.Name.Name
					if s.TypeParams != nil {
						var ps []string
						for _, f := range s.TypeParams.List {
							for _, id := range f.Names {
								ps = append(ps, id.Name+" "+render(fset, f.Type))
							}
						}
						name += "[" + strings.Join(ps, ", ") + "]"
					}
					switch t := s.Type.(type) {
					case *ast.StructType:
						add("type %s struct", name)
						for _, fld := range t.Fields.List {
							typ := render(fset, fld.Type)
							if len(fld.Names) == 0 {
								if ast.IsExported(strings.TrimLeft(typ, "*")) {
									add("field %s.%s (embedded)", s.Name.Name, typ)
								}
								continue
							}
							for _, id := range fld.Names {
								if id.IsExported() {
									add("field %s.%s %s", s.Name.Name, id.Name, typ)
								}
							}
						}
					case *ast.InterfaceType:
						add("type %s interface", name)
						for _, m := range t.Methods.List {
							for _, id := range m.Names {
								if id.IsExported() {
									add("method %s.%s %s", s.Name.Name, id.Name, render(fset, m.Type))
								}
							}
						}
					default:
						eq := " "
						if s.Assign.IsValid() {
							eq = " = "
						}
						add("type %s%s%s", name, eq, render(fset, s.Type))
					}
				}
			}
		}
	}
	return out
}

// exportedRecv reports whether the receiver type is exported.
func exportedRecv(r *ast.FieldList) bool {
	if len(r.List) == 0 {
		return false
	}
	t := r.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
		case *ast.IndexExpr:
			t = x.X
		case *ast.IndexListExpr:
			t = x.X
		case *ast.Ident:
			return x.IsExported()
		default:
			return false
		}
	}
}

// ExportedFields returns the exported field names of the named struct types of
// the package in dir.
func ExportedFields(dir string, types ...string) (map[string][]string, error) {
	fset := token.NewFileSet()
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", dir, err)
	}
	want := map[string]bool{}
	for _, t := range types {
		want[t] = true
	}
	out := map[string][]string{}
	for _, fn := range entries {
		if strings.HasSuffix(fn, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, fn, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", fn, err)
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !want[ts.Name.Name] {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, fld := range st.Fields.List {
					for _, id := range fld.Names {
						if id.IsExported() {
							out[ts.Name.Name] = append(out[ts.Name.Name], id.Name)
						}
					}
				}
			}
		}
	}
	return out, nil
}

// ApplicationVocabulary is the words that name a particular application or
// product. The library must not contain them: see docs/DESIGN.md, the generic
// gate. Extend the list when a new consumer teaches a new lesson.
func ApplicationVocabulary() []string {
	return []string{"cloudvault", "tailscale", "journalctl", "systemd", "sudo", "jellyfin", "nextcloud", "plex"}
}

// VocabularyViolations reports each whole-word, case-insensitive use of an
// application word in src.
func VocabularyViolations(rel string, src []byte, words []string) []string {
	var out []string
	re := regexp.MustCompile(`(?i)\b(` + strings.Join(words, "|") + `)\b`)
	for i, line := range strings.Split(string(src), "\n") {
		if m := re.FindString(line); m != "" {
			out = append(out, fmt.Sprintf("%s:%d: mentions %q: the library must stay generic (docs/DESIGN.md)", rel, i+1, m))
		}
	}
	return out
}

// FieldReference renders the exported fields of the named struct types of the
// package in dir, in the order given, as a markdown reference built from their
// doc comments.
func FieldReference(dir string, types ...string) (string, error) {
	fset := token.NewFileSet()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return "", fmt.Errorf("listing %s: %w", dir, err)
	}
	sort.Strings(files)
	type structDoc struct {
		doc string
		st  *ast.StructType
	}
	found := map[string]structDoc{}
	for _, fn := range files {
		if strings.HasSuffix(fn, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, fn, nil, parser.ParseComments)
		if err != nil {
			return "", fmt.Errorf("parsing %s: %w", fn, err)
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				doc := ts.Doc
				if doc == nil {
					doc = gd.Doc
				}
				found[ts.Name.Name] = structDoc{doc: oneLine(doc), st: st}
			}
		}
	}

	var sb strings.Builder
	for _, name := range types {
		sd, ok := found[name]
		if !ok {
			return "", fmt.Errorf("no struct type %s in %s", name, dir)
		}
		fmt.Fprintf(&sb, "### `%s`\n\n", name)
		if sd.doc != "" {
			fmt.Fprintf(&sb, "%s\n\n", sd.doc)
		}
		sb.WriteString("| Field | Type | What it does |\n| --- | --- | --- |\n")
		for _, fld := range sd.st.Fields.List {
			doc := oneLine(fld.Doc)
			for _, id := range fld.Names {
				if !id.IsExported() {
					continue
				}
				fmt.Fprintf(&sb, "| `%s` | `%s` | %s |\n", id.Name, escapePipes(render(fset, fld.Type)), escapePipes(doc))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// oneLine joins a comment's lines into one, dropping blank-line breaks.
func oneLine(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return strings.TrimSpace(spaceRe.ReplaceAllString(cg.Text(), " "))
}

func escapePipes(s string) string { return strings.ReplaceAll(s, "|", `\|`) }
