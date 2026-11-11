package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Source is one parsed non-test Go file.
type Source struct {
	// Rel is the slash-separated path relative to the root that was loaded.
	Rel string
	// File is the parsed syntax tree, with comments.
	File *ast.File
	// Fset resolves positions in File.
	Fset *token.FileSet
}

// Line is the line of pos in the source file.
func (s Source) Line(pos token.Pos) int { return s.Fset.Position(pos).Line }

// skipDir reports whether a directory holds no Go source any rule applies to.
func skipDir(name string) bool {
	switch name {
	case ".git", ".github", "testdata", "vendor", "node_modules":
		return true
	}
	return false
}

// Load parses every non-test .go file under root, skipping version-control, vendor and test-data directories and the
// extra directory names given.
func Load(root string, skip ...string) ([]Source, error) {
	extra := map[string]bool{}
	for _, s := range skip {
		extra[s] = true
	}
	fset := token.NewFileSet()
	var out []Source
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && (skipDir(d.Name()) || extra[d.Name()]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", p, err)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return fmt.Errorf("making %s relative: %w", p, err)
		}
		out = append(out, Source{Rel: filepath.ToSlash(rel), File: f, Fset: fset})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("loading sources under %s: %w", root, err)
	}
	return out, nil
}

// InAddons reports whether rel is in a directory named addons, or below one.
func InAddons(rel string) bool { return inAddonsDir(rel) }

// Finding is one rule broken at one place.
type Finding struct {
	// Rule is the rule ID, such as "A5".
	Rule string
	// File and Line say where.
	File string
	Line int
	// Msg says what is wrong; Hint what to do instead.
	Msg, Hint string
	// Reason is set for a finding the code excused with an allow comment.
	Reason string
}

// String formats the finding as "file:line: A5: message (instead: hint)".
func (f Finding) String() string {
	s := fmt.Sprintf("%s:%d: %s: %s", f.File, f.Line, f.Rule, f.Msg)
	if f.Hint != "" {
		s += " (instead: " + f.Hint + ")"
	}
	if f.Reason != "" {
		s += " [allowed: " + f.Reason + "]"
	}
	return s
}

// Report is everything CheckApp found.
type Report struct {
	// Findings are rule breaks that are not excused. Any makes the check fail.
	Findings []Finding
	// Exceptions are rule breaks excused by a //dashygotui:allow comment.
	Exceptions []Finding
	// Gaps are the //dashygotui:gap markers: workarounds for missing library
	// features, each of which must say where the gap is recorded.
	Gaps []Finding
}

// Rules returns the application rules by ID, for the report and for the
// dashygotui rules command.
func Rules() map[string]string {
	return map[string]string{
		"A1":  "addon files are named opt_*.go, widget_*.go, view_*.go, modal_*.go or shortcut_*.go",
		"A2":  "addons never use declarations from a sibling addon file",
		"A3":  "an addon has no package-level mutable variable",
		"A4":  "an addon starts no goroutines, sleeps, exits, prints, or panics",
		"A5":  "an addon draws with theme styles, never raw lipgloss styles or colour literals",
		"A6":  "text written by an addon is plain ASCII; symbols come from the theme",
		"A7":  "an addon ends, prints and runs programs through the Context, never through tea",
		"A8":  "an addon does not wrap or embed a library modal",
		"A9":  "an addon uses Opens and Do instead of a Run that only opens a view or returns nil",
		"A10": "an addon keeps no state in a variable captured from init and changed by a callback; use dashygotui.Local",
		"C1":  "a change that adds an addon edits no other Go file (dashygotui check -changed)",
		"G1":  "a //dashygotui:gap marker says where the gap is recorded",
		"X1":  "a //dashygotui:allow comment names a rule and gives a reason",
	}
}

// CheckApp applies the application rules to the sources of an application. The
// addon rules apply to files in a directory named addons: A1 to A4, A6 and A9 only
// there. A5 (raw styles), A7 and A8 apply to every file that imports the library
// as well, because a workaround is as likely to live in the dashboard package as
// in an addon. The gap markers are collected from every file.
func CheckApp(srcs []Source) Report {
	var rep Report
	byDir := map[string][]Source{}
	for _, s := range srcs {
		rep.add(s, gapMarkers(s))
		addon := InAddons(s.Rel)
		if addon {
			byDir[dirOf(s.Rel)] = append(byDir[dirOf(s.Rel)], s)
		} else if importsLibrary(s.File) {
			rep.add(s, sourceFindings(s, false))
		}
	}
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		files := byDir[d]
		for _, s := range files {
			rep.add(s, sourceFindings(s, true))
		}
		rep.add(Source{}, couplingFindings(files), files...)
	}
	rep.sort()
	return rep
}

func dirOf(rel string) string {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return "."
}

// add sorts findings into the report, applying allow comments. Findings for
// gap markers go to Gaps. srcs are the files the findings may belong to; with
// none, s is the only one.
func (r *Report) add(s Source, fs []Finding, srcs ...Source) {
	if len(srcs) == 0 {
		srcs = []Source{s}
	}
	allows := map[string]map[int]allow{}
	for _, src := range srcs {
		allows[src.Rel] = allowComments(src)
	}
	for _, f := range fs {
		switch f.Rule {
		case "gap":
			f.Rule = "G1"
			r.Gaps = append(r.Gaps, f)
		case "G1":
			r.Findings = append(r.Findings, f)
		default:
			if a, ok := lookupAllow(allows[f.File], f.Line, f.Rule); ok {
				if a.reason == "" {
					r.Findings = append(r.Findings, Finding{Rule: "X1", File: f.File, Line: a.line,
						Msg: "//dashygotui:allow " + f.Rule + " needs a reason", Hint: "write the reason after the rule ID"})
					continue
				}
				f.Reason = a.reason
				r.Exceptions = append(r.Exceptions, f)
				continue
			}
			r.Findings = append(r.Findings, f)
		}
	}
}

func (r *Report) sort() {
	less := func(a []Finding) func(i, j int) bool {
		return func(i, j int) bool {
			if a[i].File != a[j].File {
				return a[i].File < a[j].File
			}
			if a[i].Line != a[j].Line {
				return a[i].Line < a[j].Line
			}
			return a[i].Rule < a[j].Rule
		}
	}
	sort.Slice(r.Findings, less(r.Findings))
	sort.Slice(r.Exceptions, less(r.Exceptions))
	sort.Slice(r.Gaps, less(r.Gaps))
}

// --- allow comments ---

var majorVersion = regexp.MustCompile(`^v\d+$`)

var allowRe = regexp.MustCompile(`^//\s*dashygotui:allow\s+([A-Z]\d+)\s*(.*)$`)

type allow struct {
	rule, reason string
	line         int
	own          bool // the comment is alone on its line, so it excuses the next line
}

// allowComments maps a line to the allow comment on it.
func allowComments(s Source) map[int]allow {
	code := map[int]bool{} // lines on which some code starts
	ast.Inspect(s.File, func(n ast.Node) bool {
		switch n.(type) {
		case nil, *ast.Comment, *ast.CommentGroup, *ast.File:
		default:
			code[s.Line(n.Pos())] = true
		}
		return true
	})
	out := map[int]allow{}
	for _, cg := range s.File.Comments {
		for _, c := range cg.List {
			if m := allowRe.FindStringSubmatch(c.Text); m != nil {
				l := s.Line(c.Pos())
				out[l] = allow{rule: m[1], reason: strings.TrimSpace(m[2]), line: l, own: !code[l]}
			}
		}
	}
	return out
}

// lookupAllow finds an allow comment for rule on the line or the line above.
func lookupAllow(m map[int]allow, line int, rule string) (allow, bool) {
	if a, ok := m[line]; ok && a.rule == rule {
		return a, true
	}
	if a, ok := m[line-1]; ok && a.rule == rule && a.own {
		return a, true
	}
	return allow{}, false
}

// --- gap markers ---

var hexColour = regexp.MustCompile(`^#(?:[0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)

var gapRe = regexp.MustCompile(`^//\s*dashygotui:gap\b\s*(.*)$`)

// gapMarkers returns a finding with Rule "gap" for each marker, and a G1
// finding for a marker that does not say where the gap is recorded.
func gapMarkers(s Source) []Finding {
	var out []Finding
	for _, cg := range s.File.Comments {
		for _, c := range cg.List {
			m := gapRe.FindStringSubmatch(c.Text)
			if m == nil {
				continue
			}
			f := Finding{Rule: "gap", File: s.Rel, Line: s.Line(c.Pos()), Msg: strings.TrimSpace(m[1])}
			if f.Msg == "" {
				out = append(out, Finding{Rule: "G1", File: s.Rel, Line: f.Line,
					Msg: "a //dashygotui:gap marker must say where the gap is recorded", Hint: "add an issue link or a GAPS.md entry"})
				continue
			}
			out = append(out, f)
		}
	}
	return out
}

// --- coupling (A2) ---

func couplingFindings(files []Source) []Finding {
	m := map[string]*ast.File{}
	by := map[string]Source{}
	for _, s := range files {
		m[s.Rel] = s.File
		by[s.Rel] = s
	}
	var out []Finding
	for _, v := range coupling(m, func(rel string, id *ast.Ident) int { return by[rel].Line(id.Pos()) }) {
		out = append(out, Finding{Rule: "A2", File: v.file, Line: v.line,
			Msg:  fmt.Sprintf("uses %s declared in %s", v.name, v.owner),
			Hint: "move shared code to the dashboard package"})
	}
	return out
}

// --- the addon rules A1, A3 to A9 ---

// aliasOf returns the names a file imports a package under, where match tells
// whether an import path is the package. An unnamed import is named by the last
// path element, except for a version suffix such as /v2.
func aliasOf(f *ast.File, match func(path string) bool) map[string]bool {
	out := map[string]bool{}
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil || !match(p) {
			continue
		}
		switch {
		case imp.Name != nil:
			out[imp.Name.Name] = true
		default:
			parts := strings.Split(p, "/")
			name := parts[len(parts)-1]
			if len(parts) > 1 && majorVersion.MatchString(name) {
				name = parts[len(parts)-2]
			}
			out[name] = true
		}
	}
	return out
}

// importsLibrary reports whether a file imports dashygotui or one of its packages.
func importsLibrary(f *ast.File) bool {
	for _, p := range ImportPaths(f) {
		if p == "github.com/CeeSmag/dashygotui" || strings.HasPrefix(p, "github.com/CeeSmag/dashygotui/") {
			return true
		}
	}
	return false
}

// sourceFindings applies the rules to one file. addon is true for a file in an
// addons directory, which every rule applies to; elsewhere only the app-wide
// rules (A5 styles, A7, A8) apply.
func sourceFindings(s Source, addon bool) []Finding {
	var out []Finding
	at := func(rule string, n ast.Node, msg, hint string) {
		out = append(out, Finding{Rule: rule, File: s.Rel, Line: s.Line(n.Pos()), Msg: msg, Hint: hint})
	}
	f := s.File

	if addon {
		// A1: file name.
		if len(AddonNameViolations(s.Rel)) > 0 {
			out = append(out, Finding{Rule: "A1", File: s.Rel, Line: 1,
				Msg:  "addon file names start with opt_, widget_, view_, modal_ or shortcut_",
				Hint: "rename the file, for example opt_" + strings.TrimSuffix(baseName(s.Rel), ".go") + ".go"})
		}
		// A3: package-level mutable variables.
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				if allReadOnly(vs) {
					continue
				}
				at("A3", vs, "package-level variable "+joinNames(vs.Names), "use dashygotui.Local for per-dashboard state, or the Store for saved settings")
			}
		}
	}

	if addon {
		// A10: state captured from init.
		for _, m := range capturedMutations(f) {
			at("A10", m.node, fmt.Sprintf("%s is declared in init and changed by a callback: it is shared by every dashboard in the process", m.name),
				"keep it with dashygotui.Local (per dashboard), or in the Store if it must be saved")
		}
	}

	lip := aliasOf(f, func(p string) bool { return strings.Contains(p, "lipgloss") })
	tea := aliasOf(f, func(p string) bool { return strings.Contains(p, "bubbletea") })
	mod := aliasOf(f, func(p string) bool { return strings.HasSuffix(p, "/dashygotui/modal") })
	osPkg := aliasOf(f, func(p string) bool { return p == "os" })
	styleNames := []string{"NewStyle", "Color"}
	teaNames := []string{"Quit", "Println", "Printf", "ExecProcess", "Exec"}

	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GoStmt:
			if addon {
				at("A4", n, "go statement", "use Context.RunTask for background work")
			}
		case *ast.CallExpr:
			if addon {
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok {
						call := id.Name + "." + sel.Sel.Name
						switch {
						case call == "time.Sleep":
							at("A4", n, "time.Sleep", "use RunTask, or wait on the context")
						case call == "os.Exit":
							at("A4", n, "os.Exit", "use Context.Quit")
						case id.Name == "fmt" && strings.HasPrefix(sel.Sel.Name, "Print"):
							at("A4", n, call, "use Context.Flash, or the logger")
						case id.Name == "log" && (strings.HasPrefix(sel.Sel.Name, "Fatal") || strings.HasPrefix(sel.Sel.Name, "Panic")):
							at("A4", n, call, "return an error or use Context.FlashError")
						}
					}
				}
				if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "panic" {
					at("A4", n, "panic", "return an error or use Context.FlashError")
				}
			}
		case *ast.Ident:
			// A dot-import brings the names in unqualified; an unresolved identifier is
			// one the file does not declare itself.
			if n.Obj != nil {
				break
			}
			switch {
			case lip["."] && isOneOf(n.Name, styleNames...):
				at("A5", n, "lipgloss."+n.Name+" (dot-imported)", "use the theme's styles (Context.Theme) or a Tone")
			case tea["."] && isOneOf(n.Name, teaNames...):
				at("A7", n, "tea."+n.Name+" (dot-imported)", "use Context.Quit or Context.Exec")
			}
		case *ast.CompositeLit:
			if isPkgType(n.Type, lip, "Style") || (lip["."] && isIdent(n.Type, "Style")) {
				at("A5", n, "a lipgloss.Style literal", "use the theme's styles (Context.Theme) or a Tone")
			}
		case *ast.SelectorExpr:
			id, ok := n.X.(*ast.Ident)
			if !ok {
				break
			}
			switch {
			case lip[id.Name] && isOneOf(n.Sel.Name, styleNames...):
				at("A5", n, id.Name+"."+n.Sel.Name, "use the theme's styles (Context.Theme) or a Tone")
			case tea[id.Name] && isOneOf(n.Sel.Name, teaNames...):
				at("A7", n, id.Name+"."+n.Sel.Name, "use Context.Quit or Context.Exec")
			case addon && osPkg[id.Name] && isOneOf(n.Sel.Name, "Stdout", "Stderr", "Stdin"):
				at("A4", n, "os."+n.Sel.Name, "the terminal belongs to the UI; use Context.Flash, the logger or RunTask")
			}
		case *ast.BasicLit:
			if n.Kind != token.STRING || !addon {
				break
			}
			v, err := strconv.Unquote(n.Value)
			if err != nil {
				break
			}
			if hexColour.MatchString(v) {
				at("A5", n, "colour literal "+n.Value, "use the theme's styles (Context.Theme)")
			}
			if r, ok := firstNonASCII(v); ok {
				at("A6", n, fmt.Sprintf("non-ASCII %q in a string", r), "write plain ASCII, or draw symbols with Context.Theme().G")
			} else if r, ok := firstControl(v); ok {
				at("A6", n, fmt.Sprintf("control character %q in a string", r), "styling is the theme's job; do not write escape sequences")
			}
		case *ast.StructType:
			for _, fld := range n.Fields.List {
				if len(fld.Names) == 0 && isFromPkg(fld.Type, mod) {
					at("A8", fld, "struct embeds a library modal", "extend the library (record a gap)")
				}
			}
		case *ast.TypeSpec:
			if isFromPkg(n.Type, mod) {
				at("A8", n, "type "+n.Name.Name+" is defined from a library modal", "extend the library (record a gap)")
			}
		case *ast.KeyValueExpr:
			if !addon {
				break
			}
			if k, ok := n.Key.(*ast.Ident); ok && k.Name == "Run" {
				if fl, ok := n.Value.(*ast.FuncLit); ok {
					if msg, hint := trivialRun(fl); msg != "" {
						at("A9", n, msg, hint)
					}
				}
			}
		}
		return true
	})
	return out
}

// isPkgType reports whether t is name from one of the packages in pkgs.
func isPkgType(t ast.Expr, pkgs map[string]bool, name string) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && pkgs[id.Name]
}

func isIdent(t ast.Expr, name string) bool {
	id, ok := t.(*ast.Ident)
	return ok && id.Name == name && id.Obj == nil
}

// firstControl returns the first control character other than newline and tab.
func firstControl(s string) (rune, bool) {
	for _, r := range s {
		if r < 0x20 && r != '\n' && r != '\t' {
			return r, true
		}
	}
	return 0, false
}

func baseName(rel string) string {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[i+1:]
	}
	return rel
}

func isOneOf(s string, in ...string) bool {
	for _, x := range in {
		if s == x {
			return true
		}
	}
	return false
}

func firstNonASCII(s string) (rune, bool) {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return r, true
		}
	}
	return 0, false
}

// isFromPkg reports whether t is (a pointer to, or an instantiation of) a type
// of one of the packages named by names.
func isFromPkg(t ast.Expr, names map[string]bool) bool {
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
		case *ast.IndexExpr:
			t = x.X
		case *ast.IndexListExpr:
			t = x.X
		case *ast.SelectorExpr:
			id, ok := x.X.(*ast.Ident)
			return ok && names[id.Name]
		default:
			return false
		}
	}
}

// trivialRun reports a Run func literal that only opens a view, or that returns
// nothing but nil, with what to write instead.
func trivialRun(fl *ast.FuncLit) (msg, hint string) {
	returns, allNil, opens, modal := 0, true, false, false
	ast.Inspect(fl.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false // a nested function has its own returns
		case *ast.ReturnStmt:
			returns++
			for _, r := range n.Results {
				if id, ok := r.(*ast.Ident); !ok || id.Name != "nil" {
					allNil = false
				}
			}
			for _, r := range n.Results {
				if call, ok := r.(*ast.CallExpr); ok && calleeIs(call, "OpenView") {
					opens = true
				}
			}
		case *ast.CallExpr:
			if calleeIs(n, "OpenView") {
				opens = true
			}
			if calleeIs(n, "OpenModal") {
				modal = true
			}
		}
		return true
	})
	switch {
	case opens && (allNil || returns == 1):
		return "Run only opens a view", "use Opens: \"<view id>\""
	case modal && allNil && returns > 0:
		return "Run only opens a pop-up", "use Modal: func(c) Modal { return &modal.X{...} }"
	case returns > 0 && allNil:
		return "Run only returns nil", "use Do: func(c) { ... }"
	}
	return "", ""
}

// calleeIs reports whether call is a call of a function or method named name.
func calleeIs(call *ast.CallExpr, name string) bool {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name == name
	case *ast.SelectorExpr:
		return f.Sel.Name == name
	}
	return false
}

// mutation is a change, inside a callback, to a variable captured from init.
type mutation struct {
	name string
	node ast.Node
}

// capturedMutations finds variables declared at the top of an init function (other
// than functions) that a function literal in it assigns to, increments, or changes
// the elements or fields of: state shared by every dashboard in the process. It
// reports the first change of each variable. Changing an object through a method
// (series.Add) is not seen; the check is a net for the common cases.
func capturedMutations(f *ast.File) []mutation {
	var out []mutation
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Name.Name != "init" || fd.Body == nil {
			continue
		}
		declared := map[string]bool{}
		for _, st := range fd.Body.List {
			switch st := st.(type) {
			case *ast.DeclStmt:
				if gd, ok := st.Decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
					for _, spec := range gd.Specs {
						vs := spec.(*ast.ValueSpec)
						if len(vs.Values) == 1 {
							if _, isFunc := vs.Values[0].(*ast.FuncLit); isFunc {
								continue
							}
						}
						for _, id := range vs.Names {
							declared[id.Name] = true
						}
					}
				}
			case *ast.AssignStmt:
				if st.Tok != token.DEFINE {
					continue
				}
				for i, l := range st.Lhs {
					if id, ok := l.(*ast.Ident); ok {
						if len(st.Rhs) == len(st.Lhs) {
							if _, isFunc := st.Rhs[i].(*ast.FuncLit); isFunc {
								continue
							}
						}
						declared[id.Name] = true
					}
				}
			}
		}
		if len(declared) == 0 {
			continue
		}
		seen := map[string]bool{}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			fl, ok := n.(*ast.FuncLit)
			if !ok {
				return true
			}
			own := ownNames(fl)
			report := func(e ast.Expr, at ast.Node) {
				if id := rootIdent(e); id != nil && declared[id.Name] && !own[id.Name] && !seen[id.Name] {
					seen[id.Name] = true
					out = append(out, mutation{name: id.Name, node: at})
				}
			}
			ast.Inspect(fl.Body, func(m ast.Node) bool {
				switch m := m.(type) {
				case *ast.AssignStmt:
					if m.Tok != token.DEFINE {
						for _, l := range m.Lhs {
							report(l, m)
						}
					}
				case *ast.IncDecStmt:
					report(m.X, m)
				}
				return true
			})
			return true
		})
	}
	return out
}

// ownNames are the names a function literal declares itself: its parameters,
// results, and anything defined inside it. An assignment to one of those is not a
// change to a captured variable.
func ownNames(fl *ast.FuncLit) map[string]bool {
	own := map[string]bool{}
	for _, list := range []*ast.FieldList{fl.Type.Params, fl.Type.Results} {
		if list == nil {
			continue
		}
		for _, f := range list.List {
			for _, id := range f.Names {
				own[id.Name] = true
			}
		}
	}
	ast.Inspect(fl.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				for _, l := range n.Lhs {
					if id, ok := l.(*ast.Ident); ok {
						own[id.Name] = true
					}
				}
			}
		case *ast.ValueSpec:
			for _, id := range n.Names {
				own[id.Name] = true
			}
		case *ast.RangeStmt:
			if n.Tok == token.DEFINE {
				for _, e := range []ast.Expr{n.Key, n.Value} {
					if id, ok := e.(*ast.Ident); ok {
						own[id.Name] = true
					}
				}
			}
		}
		return true
	})
	return own
}

// rootIdent is the variable an assignment target is rooted in: x for x, x[i],
// x.f, *x and (x).
func rootIdent(e ast.Expr) *ast.Ident {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x
		case *ast.IndexExpr:
			e = x.X
		case *ast.SelectorExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		default:
			return nil
		}
	}
}

// ChangedFile is one file of a change set: its path relative to the application
// root, and how it changed: 'A' added, 'M' modified, 'D' deleted, 'R' renamed.
type ChangedFile struct {
	Path   string
	Status byte
}

// ChangeFindings applies the one-file rule to a change set: a change that adds an
// addon must edit no other file, not the dashboard file, not another addon, not a
// registry table (docs/APP-RULES.md). Tests, go.mod and go.sum are exempt, and so
// are files that are not Go source or documentation of the application's own. The
// findings have rule C1 and are advice: an edit can be unavoidable (a new section
// in the dashboard file, say), in which case the change description says why.
func ChangeFindings(files []ChangedFile) []Finding {
	var added []string
	for _, f := range files {
		if f.Status == 'A' && InAddons(f.Path) && strings.HasSuffix(f.Path, ".go") && !strings.HasSuffix(f.Path, "_test.go") {
			added = append(added, f.Path)
		}
	}
	if len(added) == 0 {
		return nil
	}
	var out []Finding
	for _, f := range files {
		if f.Status == 'A' || exemptFromOneFile(f.Path) {
			continue
		}
		verb := map[byte]string{'M': "edited", 'D': "deleted", 'R': "renamed"}[f.Status]
		if verb == "" {
			verb = "changed"
		}
		out = append(out, Finding{Rule: "C1", File: f.Path, Line: 1,
			Msg:  fmt.Sprintf("%s while adding %s: adding an addon should edit no other file", verb, strings.Join(added, ", ")),
			Hint: "if the edit is unavoidable, say why in the change description"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out
}

// exemptFromOneFile reports whether a changed file may accompany a new addon.
func exemptFromOneFile(p string) bool {
	switch {
	case strings.HasSuffix(p, "_test.go"), p == "go.mod", p == "go.sum", strings.HasSuffix(p, "/go.mod"), strings.HasSuffix(p, "/go.sum"):
		return true
	case strings.Contains(p, "/testdata/"), strings.HasPrefix(p, "testdata/"):
		return true
	case !strings.HasSuffix(p, ".go"):
		return true // documentation and other files are not registries
	}
	return false
}
