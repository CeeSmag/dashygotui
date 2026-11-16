package guard_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/internal/guard"
)

const (
	mapModule = "example.com/app"
	mapDash   = "app/dashboard/"
	mapAddons = "app/dashboard/addons/"
)

// mapSrc is a Source of the application example.com/app, whose files are under
// app/.
func mapSrc(t *testing.T, rel, text string) guard.Source {
	t.Helper()
	s := src(t, rel, text)
	s.Module = mapModule
	s.Pkg = mapModule + "/" + strings.TrimPrefix(filepath.ToSlash(filepath.Dir(rel)), "app/")
	return s
}

// mapApp is a clean map-laid-out application: the dashboard package, the map
// and one feature, plus the files given.
func mapApp(t *testing.T, extra ...guard.Source) []guard.Source {
	t.Helper()
	files := []guard.Source{
		mapSrc(t, mapDash+"dashboard.go", `package dashboard
type State struct{ N int }
type (
	Ctx    = int
	Option = int
)
func Helper() int { return 1 }
var Shared = 1
const PageMain = "main"`),
		mapSrc(t, mapAddons+guard.MapFile, `package addons
import "example.com/app/dashboard"
func Map() dashboard.Map { return dashboard.Map{Items: []any{backupOpt}} }`),
		mapSrc(t, mapAddons+"backup_opt.go", `package addons
import "example.com/app/dashboard"
var backupOpt = dashboard.Option{Label: "Backup", Do: func(c *dashboard.Ctx) { _ = c.State().N }}`),
	}
	return append(files, extra...)
}

func TestMapLayoutRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		rel   string
		code  string
		want  string // space-separated rule IDs; "" for none
		inMsg string // a part of the first finding's message
	}{
		{name: "clean"},
		{name: "a feature that is not placed", rel: mapAddons + "restart_opt.go", code: `package addons
import "example.com/app/dashboard"
var restartOpt = dashboard.Option{Label: "Restart"}`, want: "A13", inMsg: "restartOpt is not placed in the map"},
		{name: "a helper file", rel: mapAddons + "helpers.go", code: `package addons`, want: "A1", inMsg: "<name>_<kind>.go"},
		{name: "a v0.4 name", rel: mapAddons + "opt_backup2.go", code: `package addons`, want: "A1"},
		{name: "a leftover generated list", rel: mapAddons + guard.GenFile, code: `package addons`, want: "A1", inMsg: "belongs to the v0.4 layout"},
		{name: "doc.go", rel: mapAddons + guard.DocFile, code: `package addons`, want: "A1"},
		{name: "a function of the dashboard package", rel: mapAddons + "x_widget.go", code: `package addons
import "example.com/app/dashboard"
func xRows() int { return dashboard.Helper() }`, want: "A2", inMsg: "uses dashboard.Helper"},
		{name: "a variable and a constant of the dashboard package", rel: mapAddons + "x_widget.go", code: `package addons
import d "example.com/app/dashboard"
func xRows() (int, string) { return d.Shared, d.PageMain }`, want: "A2 A2"},
		{name: "another application package", rel: mapAddons + "x_widget.go", code: `package addons
import "example.com/app/internal/backend"
func xRows() int { return backend.N }`, want: "A2", inMsg: "imports example.com/app/internal/backend"},
		{name: "a library or other module is fine", rel: mapAddons + "x_widget.go", code: `package addons
import (
	"strings"
	"github.com/CeeSmag/dashygotui/text"
)
func xRows() string { return text.Pad(strings.ToUpper("x"), 3) }`},
		{name: "a sibling feature", rel: mapAddons + "x_widget.go", code: `package addons
var xWidget = backupOpt`, want: "A13 A2 A3"},
		{name: "init in a feature", rel: mapAddons + "x_hook.go", code: `package addons
func init() {}`, want: "A12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var extra []guard.Source
			if tt.rel != "" {
				extra = append(extra, mapSrc(t, tt.rel, tt.code))
			}
			rep := guard.CheckApp(mapApp(t, extra...))
			if got := strings.Join(rules(rep.Findings), " "); got != tt.want {
				t.Fatalf("rules = %q, want %q: %v", got, tt.want, rep.Findings)
			}
			if tt.inMsg != "" && !strings.Contains(rep.Findings[0].Msg, tt.inMsg) {
				t.Errorf("message = %q, want it to contain %q", rep.Findings[0].Msg, tt.inMsg)
			}
		})
	}
}

func TestMapFileHoldsNoVariables(t *testing.T) {
	t.Parallel()
	files := mapApp(t)
	files[1] = mapSrc(t, mapAddons+guard.MapFile, `package addons
import "example.com/app/dashboard"
var extra = dashboard.Option{}
func Map() dashboard.Map { return dashboard.Map{Items: []any{backupOpt, extra}} }`)
	rep := guard.CheckApp(files)
	if got := strings.Join(rules(rep.Findings), " "); got != "A3" || !strings.Contains(rep.Findings[0].Msg, "in the map") {
		t.Fatalf("findings = %v, want one A3 about a variable in the map", rep.Findings)
	}
}

func TestOneFileRuleLetsTheMapChange(t *testing.T) {
	t.Parallel()
	got := guard.ChangeFindings([]guard.ChangedFile{
		{Path: "dashboard/addons/restart_opt.go", Status: 'A'},
		{Path: "dashboard/addons/" + guard.MapFile, Status: 'M'},
		{Path: "dashboard/dashboard.go", Status: 'M'},
	})
	if len(got) != 1 || got[0].File != "dashboard/dashboard.go" {
		t.Fatalf("findings = %v, want C1 for dashboard.go only", got)
	}
}

func TestLoadKnowsEachFilesPackage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(rel, text string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/app\n")
	write("internal/dashboard/addons/x_opt.go", "package addons\n")
	srcs, err := guard.Load(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(srcs) != 1 || srcs[0].Pkg != "example.com/app/internal/dashboard/addons" || srcs[0].Module != "example.com/app" {
		t.Fatalf("sources = %+v", srcs)
	}
}
