package dashygotui_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

type (
	mapT   = dashygotui.Map[state]
	pageT  = dashygotui.Page[state]
	addonT = dashygotui.Addon[state]
)

// row is an option that records its label in log when opened.
func row(label string, log *[]string) option {
	return option{Label: label, Do: func(*ctx) { *log = append(*log, label) }}
}

func textWidget(title, body string) widget {
	return widget{Title: title, Render: func(*ctx, dashygotui.Size) []string { return []string{body} }}
}

// mapConfig is a dashboard of two tabs and a sub-page, built from a Map.
func mapConfig(log *[]string, sessions *[]string) cfg {
	diag := pageT{Title: "Diagnostics", Entry: &dashygotui.Entry[state]{Hotkey: "d", HotkeyHint: "Diag"},
		Session: func(*ctx) func() {
			*sessions = append(*sessions, "diag+")
			return func() { *sessions = append(*sessions, "diag-") }
		},
		Panels: []panelT{
			{Widget: textWidget("Health", "all green")},
			{Title: "Tools", Items: []addonT{row("Self-Test", log), row("Export", log),
				option{Label: "Probe", Hotkey: "p", Do: func(*ctx) { *log = append(*log, "probe") }}}},
		}}
	return cfg{
		Title: "MapApp",
		Map: mapT{
			Pages: []pageT{
				{ID: "monitor", Title: "Monitor", Panels: []panelT{
					{Widget: textWidget("Stats", "count is here")},
					{Title: "Quick Actions", Items: []addonT{row("Backup", log), row("Restart", log), diag}},
				}},
				{Title: "Settings", Panels: []panelT{
					{Title: "Saved", Number: 10, Items: []addonT{row("Storage", log), row("Region", log)}},
					{Title: "More", Items: []addonT{row("Volume", log)}},
				}},
			},
		},
	}
}

var spaces = regexp.MustCompile(`\s+`)

// hasRow reports whether the screen shows row "[n] label", however spaced.
func hasRow(d *dashtest.Driver, n, label string) bool {
	return regexp.MustCompile(`\[` + n + `\]\s+` + regexp.QuoteMeta(label)).MatchString(d.Screen())
}

func TestMapNumbersRowsByPosition(t *testing.T) {
	var log, sessions []string
	d := dashtest.New(t, mapConfig(&log, &sessions), 120, 40)
	for _, r := range [][2]string{{"1", "Backup"}, {"2", "Restart"}, {"3", "Diagnostics (d)"}} {
		if !hasRow(d, r[0], r[1]) {
			t.Errorf("no row [%s] %s on the first page:\n%s", r[0], r[1], d.Screen())
		}
	}
	d.RequireContains("count is here") // the widget panel
	d.Press("tab")
	for _, r := range [][2]string{{"10", "Storage"}, {"11", "Region"}, {"12", "Volume"}} {
		if !hasRow(d, r[0], r[1]) {
			t.Errorf("no row [%s] %s on the second page:\n%s", r[0], r[1], d.Screen())
		}
	}
	d.Press("shift+tab")
	d.Type("12") // a number on another page moves there
	d.Press("enter")
	if strings.Join(log, ",") != "Volume" {
		t.Errorf("log = %q, want Volume", log)
	}
}

func TestMapSubPage(t *testing.T) {
	var log, sessions []string
	d := dashtest.New(t, mapConfig(&log, &sessions), 120, 40)
	d.Press("d") // the sub-page's hotkey works from the tab page
	d.RequireContains("all green")
	if got := spaces.ReplaceAllString(d.Screen(), " "); !strings.Contains(got, "[1] Self-Test") || !strings.Contains(got, "[3] Probe (p)") {
		t.Fatalf("the sub-page numbers its own rows from 1:\n%s", d.Screen())
	}
	if strings.Join(sessions, ",") != "diag+" {
		t.Errorf("sessions = %q, want the sub-page's session started", sessions)
	}
	d.Press("p") // a row hotkey inside the sub-page
	d.Type("2")
	d.Press("enter")
	if strings.Join(log, ",") != "probe,Export" {
		t.Errorf("log = %q", log)
	}
	d.Press("esc")
	d.RequireAbsent("all green")
	d.RequireContains("count is here")
	if strings.Join(sessions, ",") != "diag+,diag-" {
		t.Errorf("sessions = %q, want it ended on Back", sessions)
	}
	d.Press("p") // the sub-page's hotkeys do not work from the tab page
	if strings.Join(log, ",") != "probe,Export" {
		t.Errorf("log = %q: a sub-page hotkey ran outside it", log)
	}
}

func TestMapViewsShortcutsHooksAndReadyMadeRows(t *testing.T) {
	var ticks int
	buf := logs.NewBuffer(10)
	c := cfg{Map: mapT{
		Shortcuts: []dashygotui.Shortcut[state]{{Key: "h", Opens: "help"}},
		Hooks:     []dashygotui.Hook[state]{{Tick: func(*ctx, time.Time) tea.Cmd { ticks++; return nil }}},
		Views: []dashygotui.View[state]{{ID: "help", Title: "Help",
			Body: func(*ctx, dashygotui.Size) []string { return []string{"help text"} }}},
		Pages: []pageT{{Title: "Main", Panels: []panelT{
			{Widget: widget{Title: "Ticker", Tick: func(*ctx, time.Time) tea.Cmd { ticks += 10; return nil },
				Render: func(*ctx, dashygotui.Size) []string { return nil }}},
			{Title: "Tools", Items: []addonT{
				dashygotui.View[state]{Title: "Notes", Entry: &dashygotui.Entry[state]{Note: "read only"},
					Body: func(*ctx, dashygotui.Size) []string { return []string{"the notes"} }},
				stdaddons.ThemePicker[state]{},
				stdaddons.LogView[state]{Buffer: buf},
			}},
		}}},
	}}
	d := dashtest.New(t, c, 120, 40)
	for _, r := range [][2]string{{"1", "Notes"}, {"2", "Color Theme"}, {"3", "Logs"}} {
		if !hasRow(d, r[0], r[1]) {
			t.Errorf("no row [%s] %s:\n%s", r[0], r[1], d.Screen())
		}
	}
	d.Press("enter")
	d.RequireContains("the notes")
	d.Press("esc", "h")
	d.RequireContains("help text")
	d.Press("esc")
	d.Type("3")
	d.Press("enter")
	d.RequireContains("Logs")
	ticks = 0
	d.Tick()
	if ticks != 11 {
		t.Errorf("ticks = %d, want the map's hook and the widget's Tick once each", ticks)
	}
}

func TestMapErrorsNameThePlace(t *testing.T) {
	ok := func() option { return option{Label: "Fine", Do: func(*ctx) {}} }
	tests := []struct {
		name string
		c    cfg
		want string
	}{
		{"no pages", cfg{Map: mapT{Hooks: []dashygotui.Hook[state]{{Tick: func(*ctx, time.Time) tea.Cmd { return nil }}}}},
			"the Map has no Pages"},
		{"no action", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Panels: []panelT{{Title: "Tools", Items: []addonT{
			option{Label: "Idle"}}}}}}}},
			`Main > Tools > row 1: option "Idle" needs exactly one of Run, Do, Opens and Modal`},
		{"widget as row", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Panels: []panelT{{Title: "Tools", Items: []addonT{
			textWidget("Chart", "x")}}}}}}},
			`Main > Tools > row 1: widget "Chart" is a panel of its own`},
		{"shortcut as row", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Panels: []panelT{{Title: "Tools", Items: []addonT{
			dashygotui.Shortcut[state]{Key: "x", Do: func(*ctx) {}}}}}}}}},
			`shortcut "x" goes in Map.Shortcuts`},
		{"widget and items", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Panels: []panelT{
			{Widget: textWidget("Chart", "x"), Items: []addonT{ok()}}}}}}},
			"Main > Chart: panel has both a Widget and Items"},
		{"untitled panel", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Panels: []panelT{{Items: []addonT{ok()}}}}}}},
			"Main > panel 1: panel has neither a Widget nor a Title"},
		{"numbers collide", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Panels: []panelT{
			{Title: "A", Items: []addonT{ok(), ok()}},
			{Title: "B", Number: 2, Items: []addonT{option{Label: "Late", Do: func(*ctx) {}}}}}}}}},
			`Main > B > row 1: option "Late" gets number 2`},
		{"view without ID", cfg{Map: mapT{Pages: []pageT{{Title: "Main"}}, Views: []dashygotui.View[state]{{Title: "Lost",
			Body: func(*ctx, dashygotui.Size) []string { return nil }}}}},
			`Map.Views[0]: view "Lost" has no ID`},
		{"sub-page opens nothing", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Panels: []panelT{{Title: "Tools", Items: []addonT{
			pageT{Title: "Sub", Panels: []panelT{{Title: "In", Items: []addonT{option{Label: "Go", Opens: "nowhere"}}}}}}}}}}}},
			`Main > Tools > Sub > In > row 1: row "Go" opens unknown view "nowhere"`},
		{"tab with entry", cfg{Map: mapT{Pages: []pageT{{Title: "Main", Entry: &dashygotui.Entry[state]{}}}}},
			"Main: a tab has no row"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dashygotui.New(tt.c)
			if err == nil {
				t.Fatalf("New succeeded, want an error containing %q", tt.want)
			}
			if !errors.Is(err, dashygotui.ErrInvalidConfig) {
				t.Errorf("error %v does not wrap ErrInvalidConfig", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v\nwant it to contain %q", err, tt.want)
			}
		})
	}
}

func TestMapDashboardPassesTheAudit(t *testing.T) {
	var log, sessions []string
	dashtest.Audit(t, mapConfig(&log, &sessions), dashtest.WalkOptions())
}

func TestMapOutlineListsSubPagesAsViews(t *testing.T) {
	var log, sessions []string
	d, err := dashygotui.New(mapConfig(&log, &sessions))
	if err != nil {
		t.Fatal(err)
	}
	out := d.Outline()
	if strings.Join(out.Pages, ",") != "monitor,Settings" {
		t.Errorf("Pages = %q, want the ID or else the Title", out.Pages)
	}
	if strings.Join(out.Views, ",") != "Monitor > Quick Actions > Diagnostics" {
		t.Errorf("Views = %q, want the sub-page named by its place", out.Views)
	}
}
