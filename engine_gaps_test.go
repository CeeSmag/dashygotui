package dashygotui_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// styleBefore returns the last escape sequence in raw ahead of the first
// occurrence of word: the style the word is drawn in.
func styleBefore(t *testing.T, raw, word string) string {
	t.Helper()
	i := strings.Index(raw, word)
	if i < 0 {
		t.Fatalf("%q is not on the screen", word)
	}
	j := strings.LastIndex(raw[:i], "\x1b[")
	if j < 0 {
		return ""
	}
	k := strings.IndexByte(raw[j:], 'm')
	return raw[j : j+k+1]
}

func TestShortcutDisplayAndTone(t *testing.T) {
	f := newFixture(t)
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "ctrl+x", Display: "Ctrl+X", Label: "Halt", Tone: dashygotui.ToneDanger,
		Scope: dashygotui.ScopeEverywhere, Run: func(*ctx) tea.Cmd { f.log = append(f.log, "halt"); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "y", Label: "Plain", Run: func(*ctx) tea.Cmd { return nil }})
	d := f.driver(140, 40)
	d.RequireContains("[Ctrl+X] Halt", "[y] Plain")
	d.RequireAbsent("[ctrl+x]")

	raw := d.Raw()
	key, label := styleBefore(t, raw, "[Ctrl+X]"), styleBefore(t, raw, "Halt")
	if key != label {
		t.Errorf("a toned hint draws its label in the key's style: key %q, label %q", key, label)
	}
	if plain := styleBefore(t, raw, "Plain"); plain == label {
		t.Errorf("a toned label looks like an untoned one (%q)", plain)
	}
	if styleBefore(t, raw, "[y]") == key {
		t.Error("the danger tone looks like the plain key style")
	}
	d.Press("ctrl+x")
	if got := strings.Join(f.log, ","); got != "halt" {
		t.Errorf("log = %q: the shortcut must still answer to its real key", got)
	}
}

func TestEveryToneIsDistinct(t *testing.T) {
	f := newFixture(t)
	tones := []dashygotui.Tone{dashygotui.ToneDefault, dashygotui.ToneDanger, dashygotui.ToneWarn,
		dashygotui.ToneInfo, dashygotui.ToneGood, dashygotui.ToneMuted}
	keys := "xyzwuo"
	for i, tone := range tones {
		f.reg.Shortcut(dashygotui.Shortcut[state]{Key: string(keys[i]), Label: fmt.Sprintf("Tone%d", i), Tone: tone,
			Run: func(*ctx) tea.Cmd { return nil }})
	}
	d := f.driver(200, 40)
	seen := map[string]int{}
	for i := range tones {
		s := styleBefore(t, d.Raw(), fmt.Sprintf("[%c]", keys[i]))
		if j, dup := seen[s]; dup {
			t.Errorf("tone %d and tone %d share the style %q", j, i, s)
		}
		seen[s] = i
	}
}

func TestOptionHotkeyDisplay(t *testing.T) {
	f := newFixture(t)
	f.reg.Option(option{Key: "logs", Section: "actions", Number: 9, Label: "Open Logs", Hotkey: "l", HotkeyDisplay: "Ctrl+L",
		HotkeyTone: dashygotui.ToneInfo, Run: func(*ctx) tea.Cmd { f.log = append(f.log, "logs"); return nil }})
	f.reg.Option(option{Key: "plain", Section: "actions", Number: 10, Label: "Open Plain", Hotkey: "p",
		Run: func(*ctx) tea.Cmd { return nil }})
	d := f.driver(120, 40)
	d.Press("tab")
	d.RequireContains("Open Logs (Ctrl+L)", "Open Plain (p)")
	raw := d.Raw()
	if styleBefore(t, raw, "(Ctrl+L)") == styleBefore(t, raw, "(p)") {
		t.Error("HotkeyTone had no effect on the hotkey")
	}
	d.Press("l")
	if got := strings.Join(f.log, ","); got != "logs" {
		t.Errorf("log = %q: the hotkey still answers to Hotkey", got)
	}
}

func TestPageFallbackWhen(t *testing.T) {
	f := newFixture(t)
	f.cfg.Pages[0].Fallback = func(*ctx, dashygotui.Size) []string { return []string{"summary"} }
	// A rule an application asked for: (narrow and short) or very short.
	f.cfg.Pages[0].FallbackWhen = func(s dashygotui.Size) bool {
		return (s.Cols < 80 && s.Rows < 22) || s.Rows < 8
	}
	d := f.driver(120, 40)
	d.RequireAbsent("summary")
	d.Resize(70, 40) // narrow but tall
	d.RequireAbsent("summary")
	d.Resize(120, 20) // short but wide
	d.RequireAbsent("summary")
	d.Resize(70, 20) // narrow and short
	d.RequireContains("summary")
	d.Resize(120, 10) // very short: the second branch (body is 7 rows)
	d.RequireContains("summary")
}

func TestPageFallbackWhenSeesTheBodySize(t *testing.T) {
	f := newFixture(t)
	var got dashygotui.Size
	f.cfg.Pages[0].Fallback = func(*ctx, dashygotui.Size) []string { return []string{"summary"} }
	f.cfg.Pages[0].FallbackWhen = func(s dashygotui.Size) bool { got = s; return false }
	d := f.driver(120, 40)
	d.Screen()
	if got.Cols != 120 || got.Rows != 37 { // the terminal less the header, tab bar and footer
		t.Errorf("FallbackWhen got %+v, want the page body: 120x37", got)
	}
}

func TestPageFallbackRuleValidation(t *testing.T) {
	fb := func(*ctx, dashygotui.Size) []string { return nil }
	cases := map[string]struct {
		edit func(*cfg)
		want string
	}{
		"both rules": {func(c *cfg) {
			c.Pages[0].Fallback = fb
			c.Pages[0].FallbackBelow = dashygotui.Size{Cols: 9}
			c.Pages[0].FallbackWhen = func(dashygotui.Size) bool { return true }
		}, "use one"},
		"when, no draw": {func(c *cfg) { c.Pages[0].FallbackWhen = func(dashygotui.Size) bool { return true } }, "no Fallback"},
		"draw, no rule": {func(c *cfg) { c.Pages[0].Fallback = fb }, "FallbackWhen"},
		"when is a rule": {func(c *cfg) {
			c.Pages[0].Fallback = fb
			c.Pages[0].FallbackWhen = func(dashygotui.Size) bool { return true }
		}, ""},
		"below is a rule": {func(c *cfg) { c.Pages[0].Fallback = fb; c.Pages[0].FallbackBelow = dashygotui.Size{Rows: 9} }, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := baseConfig(dashygotui.NewRegistry[state]())
			tc.edit(&c)
			msg := errText(t, c)
			if tc.want == "" && msg != "" {
				t.Fatalf("unexpected error: %v", msg)
			}
			if !strings.Contains(msg, tc.want) {
				t.Errorf("error = %q, want it to mention %q", msg, tc.want)
			}
		})
	}
}

func TestViewOptionsPanelChrome(t *testing.T) {
	f := newFixture(t)
	f.reg.View(dashygotui.View[state]{
		ID: "diag", Title: "Diagnostics", Mode: "DIAG",
		Status:          func(*ctx, dashygotui.Size) []string { return []string{"all good"} },
		OptionsTitle:    "Actions",
		OptionsSubtitle: "pick one",
		Header: func(c *ctx) []panel.Segment {
			return []panel.Segment{{Text: "Mode: ON", Prio: 95}}
		},
		Hints: func(*ctx) []dashygotui.Hint {
			return []dashygotui.Hint{{Key: "F5", Label: "Probe", Tone: dashygotui.ToneWarn, Prio: 90}}
		},
		Options: func(*ctx) []option {
			return []option{{Number: 1, Label: "Probe now", Run: func(*ctx) tea.Cmd { return nil }}}
		},
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "v", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("diag"); return nil }})
	d := f.driver(120, 30)
	d.Press("v")
	d.RequireContains("Diagnostics", "Mode: ON", "Actions", "pick one", "[F5] Probe", "[Esc/b] Back", "all good")
	d.RequireAbsent("Options")
	head := d.Lines()[0]
	if strings.Index(head, "Diagnostics") > strings.Index(head, "Mode: ON") {
		t.Errorf("header segments from the view go after its title: %q", head)
	}
}

func TestViewHeaderSegmentsDropBeforeTheTitle(t *testing.T) {
	f := newFixture(t)
	f.reg.View(dashygotui.View[state]{
		ID: "v", Title: "ViewTitle",
		Body:   func(*ctx, dashygotui.Size) []string { return []string{"x"} },
		Header: func(*ctx) []panel.Segment { return []panel.Segment{{Text: "Extra-Segment-That-Is-Long", Prio: 10}} },
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "v", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("v"); return nil }})
	d := f.driver(120, 30)
	d.Press("v")
	d.RequireContains("Extra-Segment-That-Is-Long")
	d.Resize(40, 30)
	d.RequireContains("TestApp")
	d.RequireAbsent("Extra-Segment-That-Is-Long")
}

// panelView registers a scrolling view in a panel and a shortcut "v" to open it.
func panelView(f *fixture, n int, tail bool, sizes *[]dashygotui.Size) {
	f.reg.View(dashygotui.View[state]{
		ID: "pv", Title: "Panelled", Scroll: true, Tail: tail,
		Panel: &dashygotui.ViewPanel[state]{Title: "Live Logs", Subtitle: "follow",
			Badge: func(c *ctx) string { return c.Badge("42 lines") }},
		Body: func(_ *ctx, size dashygotui.Size) []string {
			if sizes != nil {
				*sizes = append(*sizes, size)
			}
			out := make([]string, n)
			for i := range out {
				out[i] = fmt.Sprintf("line-%03d", i+1)
			}
			return out
		},
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "v", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("pv"); return nil }})
}

func TestViewPanelFramesAScrollingBody(t *testing.T) {
	f := newFixture(t)
	var sizes []dashygotui.Size
	panelView(f, 100, false, &sizes)
	d := f.driver(80, 24)
	d.Press("v")
	d.RequireContains("Live Logs", "[42 lines]", "follow", "rows 1-20 of 100", "line-001", "line-020")
	d.RequireAbsent("line-021")
	if first := d.Lines()[1]; !strings.ContainsAny(first, "╭┌") {
		t.Errorf("the body should start with a panel border, got %q", first)
	}
	last := sizes[len(sizes)-1]
	if last.Cols != 80-4-1 || last.Rows != 20 { // border and padding, and a column for the scroll bar
		t.Errorf("Body was given %+v, want the panel's inner size less the bar", last)
	}
	d.Press("pgdown")
	d.RequireContains("rows 20-39 of 100")
	d.Press("end")
	d.RequireContains("rows 81-100 of 100", "line-100")
}

func TestViewPanelWithoutScrollbackShowsNoPosition(t *testing.T) {
	f := newFixture(t)
	panelView(f, 3, false, nil)
	d := f.driver(80, 24)
	d.Press("v")
	d.RequireContains("Live Logs", "follow", "line-003")
	d.RequireAbsent("rows 1-")
}

func TestViewPanelTailHoldsItsPlace(t *testing.T) {
	f := newFixture(t)
	n := 100
	f.reg.View(dashygotui.View[state]{
		ID: "pt", Title: "T", Scroll: true, Tail: true, Panel: &dashygotui.ViewPanel[state]{},
		Body: func(*ctx, dashygotui.Size) []string {
			out := make([]string, n)
			for i := range out {
				out[i] = fmt.Sprintf("event-%03d", i+1)
			}
			return out
		},
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "v", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("pt"); return nil }})
	d := f.driver(80, 24)
	d.Press("v")
	d.RequireContains("event-100", "rows 81-100 of 100") // starts live, at the bottom
	d.Press("up", "up", "up")
	d.RequireContains("rows 78-97 of 100")
	n = 110 // rows are appended while the user is reading back
	d.Tick()
	d.RequireContains("event-097", "rows 78-97 of 110")
	d.Press("end")
	d.RequireContains("event-110")
}

func TestPanelNeedsABody(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.View(dashygotui.View[state]{ID: "v", Title: "V", Panel: &dashygotui.ViewPanel[state]{}})
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "Panel") {
		t.Errorf("Validate = %v, want an error about Panel without Body", err)
	}
}

func TestViewPanelFitsEverySize(t *testing.T) {
	f := newFixture(t)
	panelView(f, 200, true, nil)
	d := f.driver(80, 24)
	d.Press("v")
	for _, sz := range [][2]int{{1, 1}, {20, 6}, {39, 9}, {40, 10}, {41, 11}, {80, 24}, {200, 80}} {
		d.Resize(sz[0], sz[1])
		d.Press("up", "pgup", "home", "end")
		lines := d.Lines()
		if len(lines) != sz[1] {
			t.Fatalf("%dx%d: %d rows", sz[0], sz[1], len(lines))
		}
		for _, l := range lines {
			if w := text.Width(l); w != sz[0] {
				t.Fatalf("%dx%d: a row is %d wide", sz[0], sz[1], w)
			}
		}
	}
}

func TestStartViewOpensTheStack(t *testing.T) {
	f := newFixture(t)
	var events []string
	for _, id := range []string{"diag", "logs"} {
		f.reg.View(dashygotui.View[state]{
			ID: id, Title: strings.ToUpper(id),
			Body:    func(*ctx, dashygotui.Size) []string { return []string{"this is " + id} },
			OnEnter: func(*ctx) tea.Cmd { events = append(events, "enter "+id); return nil },
			OnLeave: func(*ctx) tea.Cmd { events = append(events, "leave "+id); return nil },
		})
	}
	f.cfg.StartView = []string{"diag", "logs"}
	d := f.driver(100, 30)
	d.RequireContains("this is logs")
	d.RequireAbsent("Stats")
	if got := strings.Join(events, ","); got != "enter diag,leave diag,enter logs" {
		t.Errorf("hooks = %q, want them to run as if the user had navigated", got)
	}
	d.Press("esc")
	d.RequireContains("this is diag")
	d.Press("esc")
	d.RequireContains("Stats", "Monitor") // the pages
	d.RequireAbsent("this is")
}

func TestStartViewHookCommandsRun(t *testing.T) {
	f := newFixture(t)
	f.reg.View(dashygotui.View[state]{
		ID: "a", Title: "A", Body: func(*ctx, dashygotui.Size) []string { return []string{"view a"} },
		OnEnter: func(c *ctx) tea.Cmd {
			return c.RunTask("enter", "", func(context.Context) (string, error) { return "probe started", nil })
		},
	})
	f.cfg.StartView = []string{"a"}
	d := f.driver(80, 24)
	d.RequireContains("probe started")
}

func TestStartViewMustExist(t *testing.T) {
	c := baseConfig(dashygotui.NewRegistry[state]())
	c.StartView = []string{"nope"}
	if msg := errText(t, c); !strings.Contains(msg, `unknown view "nope"`) {
		t.Errorf("error = %q", msg)
	}
}
