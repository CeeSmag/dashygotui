package dashygotui_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/internal/collect"
)

type state struct{ Count int }

type (
	reg    = collect.List[state]
	ctx    = dashygotui.Context[state]
	option = dashygotui.Option[state]
	widget = dashygotui.Widget[state]
	cfg    = dashygotui.Config[state]
	panelT = dashygotui.Panel[state]
)

func noop(*ctx) tea.Cmd { return nil }

// validate builds a dashboard from r on the base pages and returns the error,
// which reports every mistake in r.
func validate(r *reg) error {
	_, err := dashygotui.New(baseConfig(r))
	return err
}

// baseConfig is a small valid dashboard of three pages. The second has two
// panels of r's rows, "actions" numbered from 1 and "settings" from 12; the
// third has "appearance", from 20. r's views, shortcuts and hooks come with
// "actions". r is read when the dashboard is built, so a test may add to it
// after calling baseConfig.
func baseConfig(r *reg) cfg {
	if r == nil {
		r = collect.New[state]()
	}
	return cfg{
		Title: "TestApp",
		Map: dashygotui.Map[state]{Pages: []pageT{
			{ID: "monitor", Title: "Monitor"},
			{ID: "setup", Title: "Setup", Panels: []panelT{
				{Title: "Quick Actions", Items: []dashygotui.Addon[state]{r.All("actions")}},
				{Title: "Settings", Number: 12, Items: []dashygotui.Addon[state]{r.Rows("settings")}},
			}},
			{ID: "theme", Title: "Theme", Panels: []panelT{
				{Title: "Appearance", Number: 20, Items: []dashygotui.Addon[state]{r.Rows("appearance")}},
			}},
		}},
	}
}

// withWidget adds w as a panel of the configuration's first page.
func withWidget(c *cfg, w widget) {
	c.Map.Pages[0].Panels = append(c.Map.Pages[0].Panels, panelT{Widget: w})
}

func validOption(label string) option {
	return option{Label: "Label " + label, Run: noop}
}

// errText builds the dashboard and returns the text of its error, or "".
func errText(t *testing.T, c cfg) string {
	t.Helper()
	_, err := dashygotui.New(c)
	if err == nil {
		return ""
	}
	if !errors.Is(err, dashygotui.ErrInvalidConfig) {
		t.Errorf("error %v does not wrap ErrInvalidConfig", err)
	}
	return err.Error()
}

func TestNewAcceptsAValidConfig(t *testing.T) {
	r := collect.New[state]()
	r.Row("actions", validOption("a"))
	c := baseConfig(r)
	withWidget(&c, widget{Title: "W", Render: func(*ctx, dashygotui.Size) []string { return nil }})
	if msg := errText(t, c); msg != "" {
		t.Fatalf("unexpected error: %s", msg)
	}
}

func TestNewRejectsMistakesAndNamesThePlace(t *testing.T) {
	tests := []struct {
		name     string
		register func(r *reg)
		want     []string
	}{
		{
			name:     "missing label",
			register: func(r *reg) { r.Row("actions", option{Run: noop}) },
			want:     []string{"Setup > Quick Actions > row 1: option has no Label"},
		},
		{
			name:     "missing run",
			register: func(r *reg) { r.Row("actions", option{Label: "L"}) },
			want:     []string{`option "L" needs exactly one of Run, Do, Opens and Modal, has 0`},
		},
		{
			name: "several mistakes are all reported",
			register: func(r *reg) {
				r.Row("actions", option{Run: noop})
				r.Row("settings", option{Label: "L"})
			},
			want: []string{"has no Label", "needs exactly one"},
		},
		{
			name: "duplicate hotkey",
			register: func(r *reg) {
				r.Row("actions", option{Label: "A", Hotkey: "x", Run: noop})
				r.Row("settings", option{Label: "B", Hotkey: "x", Run: noop})
			},
			want: []string{`option "B" reuses hotkey "x"`},
		},
		{
			name: "hotkey that is a shortcut",
			register: func(r *reg) {
				r.Shortcut(dashygotui.Shortcut[state]{Key: "x", Run: noop})
				r.Row("settings", option{Label: "B", Hotkey: "x", Run: noop})
			},
			want: []string{`"x"`, "shortcut"},
		},
		{
			name:     "view that draws nothing",
			register: func(r *reg) { r.View(dashygotui.View[state]{ID: "v", Title: "V"}) },
			want:     []string{`view "v" has no Body to draw`},
		},
		{
			name: "duplicate view",
			register: func(r *reg) {
				v := dashygotui.View[state]{ID: "v", Title: "V", Body: func(*ctx, dashygotui.Size) []string { return nil }}
				r.View(v)
				r.View(v)
			},
			want: []string{`duplicate view ID "v"`},
		},
		{
			name:     "reserved shortcut key",
			register: func(r *reg) { r.Shortcut(dashygotui.Shortcut[state]{Key: "tab", Run: noop}) },
			want:     []string{`shortcut "tab" is reserved`},
		},
		{
			name:     "digit shortcut",
			register: func(r *reg) { r.Shortcut(dashygotui.Shortcut[state]{Key: "5", Run: noop}) },
			want:     []string{`shortcut "5" is reserved`},
		},
		{
			name: "duplicate shortcut",
			register: func(r *reg) {
				r.Shortcut(dashygotui.Shortcut[state]{Key: "d", Run: noop})
				r.Shortcut(dashygotui.Shortcut[state]{Key: "d", Run: noop})
			},
			want: []string{`duplicate shortcut key "d"`},
		},
		{
			name:     "empty hook",
			register: func(r *reg) { r.Add(dashygotui.Hook[state]{}) },
			want:     []string{"hook sets none of Tick, Msg and Result"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := collect.New[state]()
			tt.register(r)
			got := errText(t, baseConfig(r))
			if got == "" {
				t.Fatal("want an error, got none")
			}
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("error does not mention %q:\n%s", w, got)
				}
			}
		})
	}
}

func TestMistakesUnwrapToAnAddonError(t *testing.T) {
	r := collect.New[state]()
	if err := validate(r); err != nil {
		t.Fatalf("an empty map is valid: %v", err)
	}
	r.Row("settings", option{Label: "L"})
	err := validate(r)
	var re *dashygotui.AddonError
	if !errors.As(err, &re) {
		t.Fatalf("error %v does not unwrap to *AddonError", err)
	}
	if re.Site != "Setup > Settings > row 1" {
		t.Errorf("site = %q, want the place in the map", re.Site)
	}
}

func TestNewReadsTheMapOnce(t *testing.T) {
	r := collect.New[state]()
	r.Row("actions", validOption("a"))
	d, err := dashygotui.New(baseConfig(r))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r.Row("actions", validOption("late")) // the list is the caller's: changing it later changes no dashboard
	if n := len(d.Outline().Options); n != 1 {
		t.Fatalf("the dashboard has %d options, want the 1 it was built with", n)
	}
}

func TestNewRejectsBrokenMaps(t *testing.T) {
	r := collect.New[state]()
	tests := []struct {
		name   string
		mutate func(c *cfg)
		want   string
	}{
		{"no pages", func(c *cfg) { c.Map.Pages = nil }, "the Map has no Pages"},
		{"duplicate page", func(c *cfg) { c.Map.Pages[1].ID = "monitor" }, `duplicate page ID "monitor"`},
		{"page without title", func(c *cfg) { c.Map.Pages[0].Title = "" }, "page has no Title"},
		{"panel without title", func(c *cfg) { c.Map.Pages[1].Panels[0].Title = "" }, "panel has neither a Widget nor a Title"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := baseConfig(r)
			tt.mutate(&c)
			got := errText(t, c)
			if !strings.Contains(got, tt.want) {
				t.Errorf("error = %q, want it to mention %q", got, tt.want)
			}
		})
	}
}
