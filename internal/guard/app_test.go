package guard_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/internal/guard"
)

// src builds a Source from text.
func src(t *testing.T, rel, text string) guard.Source {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, text, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	return guard.Source{Rel: rel, File: f, Fset: fset}
}

// rules lists the rule IDs of the findings.
func rules(fs []guard.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Rule)
	}
	return out
}

const addonDir = "app/dashboard/addons/"

func TestAppRules(t *testing.T) {
	tests := []struct {
		name string
		rel  string
		code string
		want string // space-separated rule IDs, in order; "" for none
	}{
		{"clean", "opt_a.go", `package addons
import "fmt"
func init() { _ = fmt.Sprint("hi") }`, ""},

		// A1
		{"A1 bad file name", "helpers.go", `package addons`, "A1"},
		{"A1 good file names", "view_x.go", `package addons`, ""},

		// A3
		{"A3 mutable var", "opt_a.go", `package addons
var counter int`, "A3"},
		{"A3 map var", "opt_a.go", `package addons
var m = map[string]int{}`, "A3"},
		{"A3 sentinel error is fine", "opt_a.go", `package addons
import "errors"
var errX = errors.New("x")`, ""},
		{"A3 const is fine", "opt_a.go", `package addons
const k = 1`, ""},

		// A4
		{"A4 go statement", "opt_a.go", `package addons
func init() { go func() {}() }`, "A4"},
		{"A4 time.Sleep", "opt_a.go", `package addons
import "time"
func init() { time.Sleep(1) }`, "A4"},
		{"A4 os.Exit", "opt_a.go", `package addons
import "os"
func init() { os.Exit(1) }`, "A4"},
		{"A4 fmt.Println", "opt_a.go", `package addons
import "fmt"
func init() { fmt.Println("x") }`, "A4"},
		{"A4 log.Fatal", "opt_a.go", `package addons
import "log"
func init() { log.Fatal("x") }`, "A4"},
		{"A4 panic", "opt_a.go", `package addons
func init() { panic("x") }`, "A4"},
		{"A4 log.Printf is fine", "opt_a.go", `package addons
import "log"
func init() { log.Printf("x") }`, ""},
		{"A4 fmt.Sprintf is fine", "opt_a.go", `package addons
import "fmt"
func init() { _ = fmt.Sprintf("x") }`, ""},

		// A5
		{"A5 NewStyle", "opt_a.go", `package addons
import "charm.land/lipgloss/v2"
func init() { _ = lipgloss.NewStyle() }`, "A5"},
		{"A5 Color", "opt_a.go", `package addons
import lg "github.com/charmbracelet/lipgloss"
func init() { _ = lg.Color("1") }`, "A5"},
		{"A5 hex literal", "opt_a.go", `package addons
var _ = "#38BDF8"`, "A5"},
		{"A5 short hex literal", "opt_a.go", `package addons
var _ = "#fff"`, "A5"},
		{"A5 a # that is not a colour", "opt_a.go", `package addons
var _ = "issue #42"`, ""},

		{"A5 dot-imported NewStyle", "opt_a.go", `package addons
import . "charm.land/lipgloss/v2"
func init() { _ = NewStyle() }`, "A5"},
		{"A5 a lipgloss.Style literal", "opt_a.go", `package addons
import "charm.land/lipgloss/v2"
var _ = lipgloss.Style{}`, "A5"},
		{"A5 a Style literal in a function", "opt_a.go", `package addons
import "charm.land/lipgloss/v2"
func init() { _ = lipgloss.Style{} }`, "A5"},
		{"A5 a declared Style identifier is not a dot-import", "opt_a.go", `package addons
import . "charm.land/lipgloss/v2"
type Style struct{}
func init() { _ = Style{} }`, ""},
		{"A5 passing a theme style type is fine", "opt_a.go", `package addons
import "charm.land/lipgloss/v2"
func f(s lipgloss.Style) string { return s.Render("x") }`, ""},

		// A4 (terminal)
		{"A4 os.Stdout", "opt_a.go", `package addons
import ("fmt";"os")
func init() { _, _ = fmt.Fprintln(os.Stdout, "x") }`, "A4"},
		{"A4 os.Stderr aliased", "opt_a.go", `package addons
import o "os"
var _ = o.Stderr`, "A4"},
		{"A4 os.Args is fine", "opt_a.go", `package addons
import "os"
func init() { _ = os.Args }`, ""},

		// A6
		{"A6 ANSI escape", "opt_a.go", "package addons\nvar _ = \"\\x1b[31mred\"", "A6"},
		{"A6 newline and tab are fine", "opt_a.go", "package addons\nvar _ = \"a\\n\\tb\"", ""},
		{"A6 non-ASCII", "opt_a.go", `package addons
var _ = "done ✓"`, "A6"},
		{"A6 ASCII is fine", "opt_a.go", `package addons
var _ = "done..."`, ""},

		// A7
		{"A7 tea.Quit", "opt_a.go", `package addons
import tea "charm.land/bubbletea/v2"
var _ = tea.Quit`, "A7"},
		{"A7 tea.Println", "opt_a.go", `package addons
import tea "charm.land/bubbletea/v2"
func init() { _ = tea.Println("x") }`, "A7"},
		{"A7 tea.ExecProcess", "opt_a.go", `package addons
import tea "charm.land/bubbletea/v2"
func init() { _ = tea.ExecProcess(nil, nil) }`, "A7"},
		{"A7 tea.Cmd is fine", "opt_a.go", `package addons
import tea "charm.land/bubbletea/v2"
var _ tea.Cmd`, ""},

		// A8
		{"A8 embeds a modal", "modal_a.go", `package addons
import "github.com/CeeSmag/dashygotui/modal"
type mine struct{ modal.Edit[int] }`, "A8"},
		{"A8 embeds a pointer to a modal", "modal_a.go", `package addons
import "github.com/CeeSmag/dashygotui/modal"
type mine struct{ *modal.Confirm[int] }`, "A8"},
		{"A8 defines a type from a modal", "modal_a.go", `package addons
import "github.com/CeeSmag/dashygotui/modal"
type mine modal.Edit[int]`, "A8"},
		{"A8 using a modal in a literal is fine", "modal_a.go", `package addons
import "github.com/CeeSmag/dashygotui/modal"
var _ = &modal.Edit[int]{}`, ""},
		{"A8 a modal as a named field is fine", "modal_a.go", `package addons
import "github.com/CeeSmag/dashygotui/modal"
type mine struct{ edit modal.Edit[int] }`, ""},

		// A9
		{"A9 Run only opens a view", "opt_a.go", `package addons
type O struct{ Run func(c int) int }
func init() { _ = O{Run: func(c int) int { return openView(c, "x") }} }
func openView(int, string) int { return 0 }`, ""}, // not a recognised OpenView call: no finding
		{"A9 Run returns OpenView", "opt_a.go", `package addons
type C struct{}
func (C) OpenView(string) error { return nil }
type O struct{ Run func(c C) error }
func init() { _ = O{Run: func(c C) error { return c.OpenView("x") }} }`, "A9"},
		{"A9 Run opens a view then returns nil", "opt_a.go", `package addons
type C struct{}
func (C) OpenView(string) error { return nil }
type O struct{ Run func(c C) any }
func init() { _ = O{Run: func(c C) any {
	if err := c.OpenView("x"); err != nil { return nil }
	return nil
}} }`, "A9"},
		{"A9 Run opens a pop-up", "opt_a.go", `package addons
type C struct{}
func (C) OpenModal(any) {}
type O struct{ Run func(c C) any }
func init() { _ = O{Run: func(c C) any { c.OpenModal(1); return nil }} }`, "A9"},
		{"A9 Run returns only nil", "opt_a.go", `package addons
type O struct{ Run func(c int) any }
func init() { _ = O{Run: func(c int) any { println(); return nil }} }`, "A9"},
		{"A9 Run returning a value is fine", "opt_a.go", `package addons
type O struct{ Run func(c int) any }
func init() { _ = O{Run: func(c int) any { return c }} }`, ""},
		{"A9 Run returning nil on one path and a value on another is fine", "opt_a.go", `package addons
type O struct{ Run func(c int) any }
func init() { _ = O{Run: func(c int) any { if c > 0 { return nil }; return c }} }`, ""},
		{"A9 nested func literal returns do not count", "opt_a.go", `package addons
type O struct{ Run func(c int) any }
func init() { _ = O{Run: func(c int) any { f := func() any { return nil }; return f }} }`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := guard.CheckApp([]guard.Source{src(t, addonDir+tt.rel, tt.code)})
			if got := strings.Join(rules(rep.Findings), " "); got != tt.want {
				t.Errorf("rules = %q, want %q\n%v", got, tt.want, rep.Findings)
			}
		})
	}
}

func TestA9HintsMatchWhatTheRunDoes(t *testing.T) {
	cases := map[string]string{
		`c.OpenModal(1); return nil`:  "Modal:",
		`println(); return nil`:       "Do:",
		`c.OpenView("x"); return nil`: "Opens:",
		`return c.OpenView("x")`:      "Opens:",
	}
	for body, want := range cases {
		code := "package addons\ntype C struct{}\nfunc (C) OpenModal(any) {}\nfunc (C) OpenView(string) error { return nil }\n" +
			"type O struct{ Run func(c C) any }\nfunc init() { _ = O{Run: func(c C) any { " + body + " }} }"
		rep := guard.CheckApp([]guard.Source{src(t, addonDir+"opt_a.go", code)})
		if len(rep.Findings) != 1 || !strings.Contains(rep.Findings[0].Hint, want) {
			t.Errorf("%s: findings = %v, want a hint mentioning %s", body, rep.Findings, want)
		}
	}
}

// A workaround is as likely to live in the dashboard package as in an addon, so
// the rules about styles, tea and library modals apply to every file that imports
// the library; the rest apply to addons only.
func TestAppWideRulesReachTheDashboardPackage(t *testing.T) {
	tests := []struct {
		name, code, want string
	}{
		{"modal wrapper", `package dashboard
import "github.com/CeeSmag/dashygotui/modal"
type SettingEdit struct{ *modal.Edit[int] }`, "A8"},
		{"raw style", `package dashboard
import (_ "github.com/CeeSmag/dashygotui"; "charm.land/lipgloss/v2")
var _ = lipgloss.NewStyle()`, "A5"},
		{"tea.Quit", `package dashboard
import (_ "github.com/CeeSmag/dashygotui"; tea "charm.land/bubbletea/v2")
var _ = tea.Quit`, "A7"},
		{"a palette's hex colours are fine here", `package dashboard
import _ "github.com/CeeSmag/dashygotui"
var palette = map[string]string{"bg": "#002B36"}`, ""},
		{"goroutines are the foundation's business", `package dashboard
import _ "github.com/CeeSmag/dashygotui"
func run() { go func() {}() }`, ""},
		{"non-ASCII in the foundation is its business", `package dashboard
import _ "github.com/CeeSmag/dashygotui"
var title = "caf\u00e9"`, ""},
		{"a package that does not use the library is not checked", `package util
import "github.com/CeeSmag/dashygotui/modal"
type X struct{ modal.Edit[int] }`, "A8"}, // it does import the library's modal package
		{"a package unrelated to the library is not checked", `package other
import "charm.land/lipgloss/v2"
var _ = lipgloss.NewStyle()`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := guard.CheckApp([]guard.Source{src(t, "app/dashboard/actions.go", tt.code)})
			if got := strings.Join(rules(rep.Findings), " "); got != tt.want {
				t.Errorf("rules = %q, want %q\n%v", got, tt.want, rep.Findings)
			}
		})
	}
}

func TestA10StateCapturedFromInit(t *testing.T) {
	const pre = "package addons\ntype T struct{ f int }\nfunc register(func()) {}\nfunc init() {\n"
	tests := []struct {
		name, body string
		want       int
	}{
		{"counter incremented in a callback", "n := 0\nregister(func() { n++ })", 1},
		{"slice appended in a callback", "var events []string\nregister(func() { events = append(events, \"x\") })", 1},
		{"map written in a callback", "killed := map[int]bool{}\nregister(func() { killed[3] = true })", 1},
		{"struct field written in a callback", "s := &T{}\nregister(func() { s.f = 2 })", 1},
		{"star assignment", "p := new(int)\nregister(func() { *p = 2 })", 1},
		{"compound assignment", "total := 0\nregister(func() { total += 2 })", 1},
		{"reported once per variable", "n := 0\nregister(func() { n++; n++ })\nregister(func() { n-- })", 1},
		{"two variables, two findings", "a, b := 0, 0\nregister(func() { a++; b++ })", 2},
		{"static data that is only read", "regions := []string{\"a\"}\nregister(func() { _ = regions[0] })", 0},
		{"a callback's own variable", "register(func() { n := 0; n++ })", 0},
		{"a parameter with the same name", "n := 0\n_ = n\nregister(func() {})\nvar f = func(n int) { n++ }\n_ = f", 0},
		{"a name redeclared inside the callback", "n := 0\n_ = n\nregister(func() { n := 1; n++ })", 0},
		{"a range variable of the same name", "i := 0\n_ = i\nregister(func() { for i := range 3 { _ = i } })", 0},
		{"a helper function", "rows := func() int { return 1 }\nregister(func() { _ = rows() })", 0},
		{"assigning in init itself is fine", "n := 0\nn++\n_ = n", 0},
		{"constants", "const keep = 3\nregister(func() { _ = keep })", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := guard.CheckApp([]guard.Source{src(t, addonDir+"opt_a.go", pre+tt.body+"\n}")})
			n := 0
			for _, f := range rep.Findings {
				if f.Rule == "A10" {
					n++
				}
			}
			if n != tt.want {
				t.Errorf("A10 findings = %d, want %d\n%v", n, tt.want, rep.Findings)
			}
		})
	}
}

func TestA10IsForAddonsOnly(t *testing.T) {
	rep := guard.CheckApp([]guard.Source{src(t, "app/dashboard/dashboard.go", `package dashboard
import _ "github.com/CeeSmag/dashygotui"
func register(func()) {}
func init() { n := 0; register(func() { n++ }) }`)})
	if len(rep.Findings) != 0 {
		t.Errorf("the foundation file is not an addon: %v", rep.Findings)
	}
}

func TestRulesApplyOnlyInsideAddons(t *testing.T) {
	rep := guard.CheckApp([]guard.Source{src(t, "app/dashboard/dashboard.go", `package dashboard
import "os"
var Registry int
func init() { go func() {}(); os.Exit(1) }`)})
	if len(rep.Findings) != 0 {
		t.Errorf("the foundation file is not an addon: %v", rep.Findings)
	}
}

func TestA2Coupling(t *testing.T) {
	rep := guard.CheckApp([]guard.Source{
		src(t, addonDir+"opt_a.go", `package addons
func helper() int { return 1 }`),
		src(t, addonDir+"opt_b.go", `package addons
func init() { _ = helper() }`),
	})
	if got := strings.Join(rules(rep.Findings), " "); got != "A2" {
		t.Fatalf("rules = %q, want A2\n%v", got, rep.Findings)
	}
	if f := rep.Findings[0]; f.File != addonDir+"opt_b.go" || f.Line != 2 {
		t.Errorf("finding at %s:%d, want opt_b.go:2", f.File, f.Line)
	}
}

func TestFindingsCarryFileAndLine(t *testing.T) {
	rep := guard.CheckApp([]guard.Source{src(t, addonDir+"opt_a.go", `package addons
import "time"

func init() {
	time.Sleep(1)
}`)})
	if len(rep.Findings) != 1 {
		t.Fatalf("findings = %v", rep.Findings)
	}
	got := rep.Findings[0].String()
	if !strings.HasPrefix(got, addonDir+"opt_a.go:5: A4: time.Sleep") || !strings.Contains(got, "instead:") {
		t.Errorf("String = %q", got)
	}
}

func TestAllowComments(t *testing.T) {
	code := `package addons
import "time"

func init() {
	time.Sleep(1) //dashygotui:allow A4 a demo that must pause
	// dashygotui:allow A4 the line below waits for hardware
	time.Sleep(2)
	time.Sleep(3) //dashygotui:allow A4
	time.Sleep(4) //dashygotui:allow A5 wrong rule
}`
	rep := guard.CheckApp([]guard.Source{src(t, addonDir+"opt_a.go", code)})
	if len(rep.Exceptions) != 2 {
		t.Errorf("exceptions = %v, want the two excused lines", rep.Exceptions)
	}
	// line 8: the allow has no reason (X1); line 9: the allow names another rule (A4)
	if got := strings.Join(rules(rep.Findings), " "); got != "X1 A4" {
		t.Errorf("findings = %q, want an X1 then an A4: %v", got, rep.Findings)
	}
	for _, e := range rep.Exceptions {
		if e.Reason == "" {
			t.Errorf("an exception must carry its reason: %v", e)
		}
	}
}

func TestGapMarkers(t *testing.T) {
	rep := guard.CheckApp([]guard.Source{
		src(t, "app/dashboard/actions.go", `package dashboard
// dashygotui:gap GAPS.md#settingedit wraps modal.Edit until OnSaved exists
func f() {}
//dashygotui:gap
func g() {}`),
	})
	if len(rep.Gaps) != 1 || !strings.Contains(rep.Gaps[0].Msg, "GAPS.md#settingedit") {
		t.Errorf("gaps = %v", rep.Gaps)
	}
	if got := strings.Join(rules(rep.Findings), " "); got != "G1" {
		t.Errorf("a marker with no reference must fail: findings = %q", got)
	}
}

func TestReportIsSorted(t *testing.T) {
	rep := guard.CheckApp([]guard.Source{
		src(t, addonDir+"opt_b.go", "package addons\nvar b int"),
		src(t, addonDir+"opt_a.go", "package addons\nvar a int"),
	})
	if len(rep.Findings) != 2 || rep.Findings[0].File > rep.Findings[1].File {
		t.Errorf("findings are not sorted: %v", rep.Findings)
	}
}

func TestEveryRuleHasADescription(t *testing.T) {
	for _, id := range []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10", "C1", "G1", "X1"} {
		if guard.Rules()[id] == "" {
			t.Errorf("rule %s has no description", id)
		}
	}
}

func TestChangeFindingsOneFileRule(t *testing.T) {
	add := func(p string) guard.ChangedFile { return guard.ChangedFile{Path: p, Status: 'A'} }
	mod := func(p string) guard.ChangedFile { return guard.ChangedFile{Path: p, Status: 'M'} }
	tests := []struct {
		name  string
		files []guard.ChangedFile
		want  []string // files reported
	}{
		{"nothing changed", nil, nil},
		{"one new addon, nothing else", []guard.ChangedFile{add("app/dashboard/addons/opt_a.go")}, nil},
		{"a new addon and its test", []guard.ChangedFile{add("app/dashboard/addons/opt_a.go"), add("app/dashboard/addons/opt_a_test.go")}, nil},
		{"several new addons", []guard.ChangedFile{add("a/addons/opt_a.go"), add("a/addons/view_b.go")}, nil},
		{"a new addon that edits the dashboard file", []guard.ChangedFile{add("app/dashboard/addons/opt_a.go"), mod("app/dashboard/dashboard.go")}, []string{"app/dashboard/dashboard.go"}},
		{"a new addon that edits a sibling addon", []guard.ChangedFile{add("app/dashboard/addons/opt_a.go"), mod("app/dashboard/addons/opt_b.go")}, []string{"app/dashboard/addons/opt_b.go"}},
		{"deleting a file while adding one", []guard.ChangedFile{add("a/addons/opt_a.go"), {Path: "a/main.go", Status: 'D'}}, []string{"a/main.go"}},
		{"go.mod, go.sum, tests and docs are exempt", []guard.ChangedFile{add("a/addons/opt_a.go"), mod("go.mod"), mod("go.sum"), mod("a/dashboard/dashboard_test.go"), mod("README.md"), mod("a/testdata/x.json")}, nil},
		{"editing alone, without adding, is not this rule", []guard.ChangedFile{mod("app/dashboard/dashboard.go"), mod("app/dashboard/addons/opt_b.go")}, nil},
		{"a new file outside addons is not an addon", []guard.ChangedFile{add("app/dashboard/helpers.go"), mod("app/dashboard/dashboard.go")}, nil},
		{"results are sorted", []guard.ChangedFile{add("a/addons/opt_a.go"), mod("z.go"), mod("b.go")}, []string{"b.go", "z.go"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, f := range guard.ChangeFindings(tt.files) {
				if f.Rule != "C1" {
					t.Errorf("rule = %s, want C1", f.Rule)
				}
				got = append(got, f.File)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("reported %v, want %v", got, tt.want)
			}
		})
	}
}
