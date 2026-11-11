package dashygotui_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/store"
	"github.com/CeeSmag/dashygotui/text"
)

// fixture is a dashboard with options on two pages, a widget, and a poll that
// counts how often it ran.
type fixture struct {
	t     *testing.T
	reg   *reg
	cfg   cfg
	polls int
	log   []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t, reg: dashygotui.NewRegistry[state]()}
	f.cfg = baseConfig(f.reg)
	f.cfg.Poll = func(context.Context) (state, error) {
		f.polls++
		return state{Count: f.polls}, nil
	}
	record := func(name string) func(*ctx) tea.Cmd {
		return func(*ctx) tea.Cmd { f.log = append(f.log, name); return nil }
	}
	f.reg.Option(option{Key: "backup", Section: "actions", Number: 1, Label: "Run Backup", Run: record("backup"),
		Badge: func(c *ctx) string { return c.Badge("idle") }})
	f.reg.Option(option{Key: "restart", Section: "actions", Number: 2, Label: "Restart", Run: record("restart")})
	f.reg.Option(option{Key: "storage", Section: "settings", Number: 12, Label: "Storage", Run: record("storage"),
		Badge: func(c *ctx) string { return c.OnOff(true) }})
	f.reg.Option(option{Key: "palette", Section: "appearance", Number: 20, Label: "Colour Palette", Run: record("palette")})
	f.reg.Widget(widget{Key: "stats", Page: "monitor", Title: "Stats", Subtitle: "live",
		Render: func(c *ctx, _ dashygotui.Size) []string {
			return []string{fmt.Sprintf("count=%d ready=%v", c.State().Count, c.Ready())}
		}})
	return f
}

func (f *fixture) driver(cols, rows int) *dashtest.Driver {
	f.t.Helper()
	return dashtest.New(f.t, f.cfg, cols, rows)
}

// rowWith returns the screen row containing s.
func rowWith(t *testing.T, d *dashtest.Driver, s string) string {
	t.Helper()
	for _, line := range d.Lines() {
		if strings.Contains(line, s) {
			return line
		}
	}
	t.Fatalf("no row contains %q:\n%s", s, d.Screen())
	return ""
}

func TestFirstFrame(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.RequireContains("TestApp", "Monitor", "Setup", "Theme", "Stats", "count=1 ready=true", "[Tab]", "LIVE 1s", "[q] Quit")
	d.RequireAbsent("Quick Actions", "Terminal too small")
	if f.polls != 1 {
		t.Errorf("polls = %d, want the initial one", f.polls)
	}
}

func TestPollRunsAgainOnEachTick(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Tick()
	d.RequireContains("count=2")
	d.Tick()
	d.RequireContains("count=3")
	if f.polls != 3 {
		t.Errorf("polls = %d, want 3", f.polls)
	}
}

func TestPollErrorKeepsTheLastState(t *testing.T) {
	f := newFixture(t)
	fail := false
	f.cfg.Poll = func(context.Context) (state, error) {
		f.polls++
		if fail {
			return state{}, errors.New("backend down")
		}
		return state{Count: f.polls}, nil
	}
	var seen error
	f.reg.Widget(widget{Key: "err", Page: "monitor", Title: "Err", Render: func(c *ctx, _ dashygotui.Size) []string {
		seen = c.Err()
		return nil
	}})
	d := f.driver(120, 40)
	fail = true
	d.Tick()
	d.RequireContains("count=1")
	if seen == nil || seen.Error() != "backend down" {
		t.Errorf("Err() = %v, want the poll error", seen)
	}
	fail = false
	d.Tick()
	d.RequireContains("count=3") // rendering is what refreshes seen
	if seen != nil {
		t.Errorf("Err() = %v after recovery, want nil", seen)
	}
}

func TestNoPollMeansZeroState(t *testing.T) {
	f := newFixture(t)
	f.cfg.Poll = nil
	d := f.driver(120, 40)
	d.RequireContains("count=0 ready=false")
}

func TestTabCyclesPages(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.RequireAbsent("Quick Actions")

	d.Press("tab")
	d.RequireContains("Quick Actions", "Settings", "[1 option]", "[2 options]", "Page 3/3")
	d.RequireAbsent("count=")

	d.Press("tab")
	d.RequireContains("Appearance", "Colour Palette")
	d.Press("tab") // wraps to the first page
	d.RequireContains("count=")
	d.Press("shift+tab") // wraps backwards to the last
	d.RequireContains("Appearance")
}

func TestCursorMovesAndWraps(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Press("tab")
	if !strings.Contains(rowWith(t, d, "Run Backup"), "❯ [1]") {
		t.Fatalf("the cursor should start on the first row:\n%s", d.Screen())
	}
	d.Press("down")
	if !strings.Contains(rowWith(t, d, "Restart"), "❯ [2]") || strings.Contains(rowWith(t, d, "Run Backup"), "❯") {
		t.Fatalf("down should move to the second row:\n%s", d.Screen())
	}
	d.Press("down") // wraps
	if !strings.Contains(rowWith(t, d, "Run Backup"), "❯") {
		t.Fatalf("down should wrap to the first row:\n%s", d.Screen())
	}
	d.Press("up") // wraps backwards
	if !strings.Contains(rowWith(t, d, "Restart"), "❯") {
		t.Fatalf("up should wrap to the last row:\n%s", d.Screen())
	}
	d.Press("home")
	if !strings.Contains(rowWith(t, d, "Run Backup"), "❯") {
		t.Fatalf("home should go to the first row")
	}
	d.Press("end")
	if !strings.Contains(rowWith(t, d, "Restart"), "❯") {
		t.Fatalf("end should go to the last row")
	}
}

func TestEnterRunsTheHighlightedOption(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Press("tab", "down", "enter")
	d.Press("right", "enter") // the Settings panel
	if got := strings.Join(f.log, ","); got != "restart,storage" {
		t.Errorf("log = %q, want restart,storage", got)
	}
}

func TestPanelsRememberTheirRow(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Press("tab", "down", "right", "left")
	if !strings.Contains(rowWith(t, d, "Restart"), "❯") {
		t.Errorf("the first panel should keep its row after leaving and returning:\n%s", d.Screen())
	}
}

func TestPageRemembersItsCursor(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Press("tab", "down") // setup page, second row
	d.Press("tab", "tab")  // away and back round to the first page, then setup again
	d.Press("tab")
	if !strings.Contains(rowWith(t, d, "Restart"), "❯") {
		t.Errorf("the page should remember its cursor:\n%s", d.Screen())
	}
}

func TestNumberJumpCrossesPages(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)

	d.Type("2") // [2] Restart lives on the Setup page
	d.RequireContains("Quick Actions", "#2")
	if !strings.Contains(rowWith(t, d, "Restart"), "❯ [2]") {
		t.Fatalf("typing 2 should land on [2]:\n%s", d.Screen())
	}
	d.Press("enter")
	if got := strings.Join(f.log, ","); got != "restart" {
		t.Errorf("log = %q, want restart", got)
	}
	d.RequireAbsent("#2") // opening clears the typed number

	d.Type("2", "0") // [20] lives on the Theme page
	d.RequireContains("Appearance")
	d.Press("enter")
	if got := strings.Join(f.log, ","); got != "restart,palette" {
		t.Errorf("log = %q", got)
	}
}

func TestNumberJumpPrefixesAndRestarts(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Press("tab")
	d.Type("1") // first match is [1]
	if !strings.Contains(rowWith(t, d, "Run Backup"), "❯") {
		t.Fatalf("1 should select [1]:\n%s", d.Screen())
	}
	d.Type("2") // 12 continues the prefix
	if !strings.Contains(rowWith(t, d, "Storage"), "❯ [12]") {
		t.Fatalf("1 then 2 should select [12]:\n%s", d.Screen())
	}
	d.Type("2") // 122 does not exist; restart with 2
	if !strings.Contains(rowWith(t, d, "Restart"), "❯ [2]") {
		t.Fatalf("a digit that cannot continue should start afresh:\n%s", d.Screen())
	}
	d.Press("esc") // esc clears the number
	d.RequireAbsent("#2")
}

func TestNumberWithNoOptionIsReported(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Type("9")
	d.RequireContains("No option [9]")
}

func TestShortcutsRunAndShowHints(t *testing.T) {
	f := newFixture(t)
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "d", Label: "Diag", Run: func(*ctx) tea.Cmd {
		f.log = append(f.log, "diag")
		return nil
	}})
	d := f.driver(160, 40)
	d.RequireContains("[d] Diag")
	d.Press("d")
	if got := strings.Join(f.log, ","); got != "diag" {
		t.Errorf("log = %q, want diag", got)
	}
}

func TestQuit(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.Press("q")
	if !d.Quit() {
		t.Error("q should quit")
	}
	d2 := f.driver(120, 40)
	d2.Press("ctrl+q")
	if !d2.Quit() {
		t.Error("ctrl+q should quit")
	}
}

func TestContextQuitCarriesAReason(t *testing.T) {
	f := newFixture(t)
	var got dashygotui.ExitInfo
	f.reg.Option(option{Key: "uninstall", Section: "actions", Number: 99, Label: "Uninstall", Run: func(c *ctx) tea.Cmd {
		got = dashygotui.ExitInfo{Reason: "uninstall", Message: "bye"}
		return c.Quit(got.Reason, got.Message)
	}})
	d := f.driver(120, 40)
	d.Type("9", "9")
	d.Press("enter")
	if !d.Quit() {
		t.Error("Quit should end the dashboard")
	}
	if got.Reason != "uninstall" {
		t.Errorf("reason = %q", got.Reason)
	}
}

// stubModal records keys and closes on esc.
type stubModal struct{ keys []string }

func (m *stubModal) HandleKey(c *ctx, msg tea.KeyPressMsg) tea.Cmd {
	m.keys = append(m.keys, msg.String())
	if msg.String() == "esc" {
		c.CloseModal()
	}
	return nil
}

func (m *stubModal) Render(c *ctx, size dashygotui.Size) []string {
	return panel.Panel(c.Theme(), "Confirm", "", "", 30, 5, []string{"Really do it?"})
}

func TestModalTakesKeysAndForceQuitStillWorks(t *testing.T) {
	f := newFixture(t)
	mod := &stubModal{}
	f.reg.Option(option{Key: "ask", Section: "actions", Number: 3, Label: "Ask", Run: func(c *ctx) tea.Cmd {
		c.OpenModal(mod)
		return nil
	}})
	d := f.driver(120, 40)
	d.Type("3")
	d.Press("enter")
	d.RequireContains("Confirm", "Really do it?")

	d.Press("q", "down", "tab") // the modal gets them; the page does not move or quit
	if d.Quit() {
		t.Error("q must not quit while a modal is open")
	}
	if got := strings.Join(mod.keys, ","); got != "q,down,tab" {
		t.Errorf("modal keys = %q", got)
	}
	d.Press("esc")
	d.RequireAbsent("Really do it?")

	d.Press("enter") // opens it again
	d.RequireContains("Really do it?")
	d.Press("ctrl+q")
	if !d.Quit() {
		t.Error("ctrl+q must quit even over a modal")
	}
}

// scrollModal is a ScrollableModal.
type scrollModal struct{ stubModal }

func (m *scrollModal) ScrollMax(*ctx, dashygotui.Size) int { return 4 }

func TestScrollableModalScrolls(t *testing.T) {
	f := newFixture(t)
	mod := &scrollModal{}
	var offset func() int
	f.reg.Option(option{Key: "ask", Section: "actions", Number: 3, Label: "Ask", Run: func(c *ctx) tea.Cmd {
		c.OpenModal(mod)
		offset = c.ModalScroll
		return nil
	}})
	d := f.driver(120, 40)
	d.Type("3")
	d.Press("enter", "down", "down", "pgdown", "pgdown")
	if got := offset(); got != 4 {
		t.Errorf("scroll = %d, want it clamped to ScrollMax 4", got)
	}
	d.Press("home")
	if got := offset(); got != 0 {
		t.Errorf("home: scroll = %d, want 0", got)
	}
	d.Press("end")
	if got := offset(); got != 4 {
		t.Errorf("end: scroll = %d, want 4", got)
	}
	d.Press("up", "up", "pgup", "pgup")
	if got := offset(); got != 0 {
		t.Errorf("scroll = %d, want it clamped at 0", got)
	}
	if len(mod.keys) != 0 {
		t.Errorf("scroll keys leaked to the modal: %v", mod.keys)
	}
}

func TestViewOpensAndBackRestoresTheCursor(t *testing.T) {
	f := newFixture(t)
	f.reg.View(dashygotui.View[state]{
		ID: "diag", Title: "Diagnostics", Mode: "DIAG", Subtitle: "health",
		Status: func(*ctx, dashygotui.Size) []string { return []string{"all systems nominal"} },
		Options: func(*ctx) []option {
			return []option{{Key: "x", Number: 1, Label: "Run Self-Test", Run: func(*ctx) tea.Cmd {
				f.log = append(f.log, "selftest")
				return nil
			}}}
		},
	})
	f.reg.Option(option{Key: "diag", Section: "actions", Number: 3, Label: "Diagnostics", Run: func(c *ctx) tea.Cmd {
		if err := c.OpenView("diag"); err != nil {
			t.Error(err)
		}
		return nil
	}})
	d := f.driver(120, 40)
	d.Press("tab", "down", "down") // cursor on [3]
	d.Press("enter")
	d.RequireContains("Diagnostics", "all systems nominal", "Run Self-Test", "DIAG", "[Esc/b] Back")
	d.RequireAbsent("Quick Actions")

	d.Press("enter") // the view's own menu
	if got := strings.Join(f.log, ","); got != "selftest" {
		t.Errorf("log = %q", got)
	}
	d.Type("1") // numbers work inside a view too
	d.Press("esc")
	d.RequireContains("Quick Actions")
	if !strings.Contains(rowWith(t, d, "Diagnostics"), "❯ [3]") {
		t.Errorf("Back should restore the cursor:\n%s", d.Screen())
	}
}

func TestViewBodyAndKeys(t *testing.T) {
	f := newFixture(t)
	var opened func(c *ctx) error
	f.reg.View(dashygotui.View[state]{
		ID: "logs", Title: "Logs",
		Body: func(c *ctx, size dashygotui.Size) []string { return []string{"line one", "line two"} },
		Keys: func(c *ctx, msg tea.KeyPressMsg) (tea.Cmd, bool) {
			if msg.String() == "x" {
				f.log = append(f.log, "x")
				return nil, true
			}
			return nil, false
		},
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "l", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("logs"); return nil }})
	_ = opened
	d := f.driver(120, 40)
	d.Press("l")
	d.RequireContains("line one", "line two")
	d.Press("x")
	if got := strings.Join(f.log, ","); got != "x" {
		t.Errorf("the view should see x first: %q", got)
	}
	d.Press("b")
	d.RequireAbsent("line one")
	d.Press("l", "q")
	if !d.Quit() {
		t.Error("q should quit from a view")
	}
}

func TestOpenViewUnknown(t *testing.T) {
	f := newFixture(t)
	var err error
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "z", Run: func(c *ctx) tea.Cmd { err = c.OpenView("missing"); return nil }})
	d := f.driver(120, 40)
	d.Press("z")
	if err == nil {
		t.Error("opening an unknown view must fail")
	}
}

func TestRunTaskReportsInTheFooter(t *testing.T) {
	f := newFixture(t)
	f.reg.Option(option{Key: "job", Section: "actions", Number: 3, Label: "Job", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("job", "working…", func(context.Context) (string, error) { return "all done", nil })
	}})
	f.reg.Option(option{Key: "bad", Section: "actions", Number: 4, Label: "Bad", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("bad", "", func(context.Context) (string, error) { return "upload", errors.New("disk full") })
	}})
	d := f.driver(160, 40)
	d.Type("3")
	d.Press("enter")
	d.RequireContains("✓ all done")
	polls := f.polls
	d.Type("4")
	d.Press("enter")
	d.RequireContains("✗ upload disk full")
	if f.polls != polls {
		t.Error("a failed task must not trigger a refresh")
	}
}

func TestSuccessfulTaskRefreshes(t *testing.T) {
	f := newFixture(t)
	f.reg.Option(option{Key: "job", Section: "actions", Number: 3, Label: "Job", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("job", "", func(context.Context) (string, error) { return "ok", nil })
	}})
	d := f.driver(160, 40)
	before := f.polls
	d.Type("3")
	d.Press("enter")
	if f.polls != before+1 {
		t.Errorf("polls = %d, want %d: a successful task should refresh", f.polls, before+1)
	}
}

func TestResultHookCanClaimATask(t *testing.T) {
	f := newFixture(t)
	var got dashygotui.TaskResult
	f.reg.OnResult(func(c *ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
		if res.Tag != "special" {
			return false, nil
		}
		got = res
		return true, nil
	})
	f.reg.Option(option{Key: "job", Section: "actions", Number: 3, Label: "Job", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("special", "", func(context.Context) (string, error) { return "claimed", nil })
	}})
	d := f.driver(160, 40)
	d.Type("3")
	d.Press("enter")
	if got.Tag != "special" || got.Text != "claimed" || got.Token == 0 {
		t.Errorf("hook saw %+v", got)
	}
	d.RequireAbsent("claimed") // the footer message was suppressed
}

// customMsg is a message an addon's own command produces.
type customMsg struct{ n int }

func TestMessageHookHandlesAddonMessages(t *testing.T) {
	f := newFixture(t)
	var seen []int
	f.reg.OnMsg(func(c *ctx, msg tea.Msg) tea.Cmd {
		if m, ok := msg.(customMsg); ok {
			seen = append(seen, m.n)
			c.Flash(fmt.Sprintf("custom %d", m.n))
			return func() tea.Msg { return nil }
		}
		return nil
	})
	f.reg.Option(option{Key: "emit", Section: "actions", Number: 3, Label: "Emit", Run: func(*ctx) tea.Cmd {
		return func() tea.Msg { return customMsg{n: 7} }
	}})
	d := f.driver(160, 40)
	d.Type("3")
	d.Press("enter")
	if len(seen) != 1 || seen[0] != 7 {
		t.Errorf("hook saw %v", seen)
	}
	d.RequireContains("custom 7")
}

func TestTickHookRunsOnEveryTick(t *testing.T) {
	f := newFixture(t)
	var ticks []time.Time
	f.reg.OnTick(func(c *ctx, now time.Time) tea.Cmd {
		ticks = append(ticks, now)
		return nil
	})
	d := f.driver(120, 40)
	d.Advance(time.Second)
	d.Advance(time.Second)
	if len(ticks) != 2 || !ticks[1].Equal(dashtest.Epoch().Add(2*time.Second)) {
		t.Errorf("ticks = %v", ticks)
	}
}

func TestFlashExpires(t *testing.T) {
	f := newFixture(t)
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "f", Run: func(c *ctx) tea.Cmd { c.Flash("heads up"); return nil }})
	d := f.driver(160, 40)
	d.Press("f")
	d.RequireContains("heads up")
	d.Advance(time.Second)
	d.RequireContains("heads up") // still within FlashHint
	d.Advance(5 * time.Second)
	d.RequireAbsent("heads up")
}

func TestHeaderAndClock(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	d.RequireContains("12:00:00")
	d.Advance(90 * time.Second)
	d.RequireContains("12:01:30")

	f.cfg.Header = func(c *ctx) []panel.Segment {
		return []panel.Segment{{Text: "Custom Header", Prio: 100}}
	}
	d = f.driver(120, 40)
	d.RequireContains("Custom Header")
	d.RequireAbsent("12:00:00")
}

func TestPageBadgeAndTitles(t *testing.T) {
	f := newFixture(t)
	f.cfg.Pages = []dashygotui.Page[state]{
		{ID: "monitor", Title: "Monitor Everything", ShortTitle: "Monitor", TinyTitle: "Mon",
			Badge: func(c *ctx) string { return "(live)" }},
		{ID: "setup", Title: "First-Time Setup", ShortTitle: "Setup", TinyTitle: "Set"},
		{ID: "theme", Title: "Theme"},
	}
	// The active tab shows its badge, and the title shrinks with the terminal
	// whether or not there is a badge.
	wide := f.driver(120, 40)
	wide.RequireContains("Monitor Everything (live)", "First-Time Setup")
	mid := f.driver(90, 40)
	mid.RequireContains("Monitor (live)", "Setup")
	mid.RequireAbsent("First-Time Setup", "Monitor Everything")
	narrow := f.driver(60, 40)
	narrow.RequireContains("Mon (live)", "Set")
	narrow.RequireAbsent("Setup", "Monitor (live)")
}

func TestSettingsAreAppliedAndPersisted(t *testing.T) {
	f := newFixture(t)
	st := store.NewMemory()
	f.cfg.Store = st
	var palettes []string
	f.reg.Option(option{Key: "look", Section: "actions", Number: 3, Label: "Look", Run: func(c *ctx) tea.Cmd {
		c.SetSpacing(dashygotui.Spacing{TerminalPage: 99, PageSection: 1, Section: 4, SectionOption: 2})
		if !c.SetPalette("nord") {
			t.Error("nord should exist")
		}
		if c.SetPalette("nope") {
			t.Error("unknown palettes must be refused")
		}
		c.SetRefreshInterval(2 * time.Second)
		c.SetRefreshInterval(-1) // ignored
		for _, p := range c.Palettes() {
			palettes = append(palettes, p.ID)
		}
		return nil
	}})
	d := f.driver(120, 40)
	d.Type("3")
	d.Press("enter")
	d.RequireContains("LIVE")

	var sp dashygotui.Spacing
	if ok, err := st.Get("dashygotui.spacing", &sp); err != nil || !ok {
		t.Fatalf("spacing not stored: %v %v", ok, err)
	}
	if sp.TerminalPage != 6 || sp.Section != 4 {
		t.Errorf("stored spacing = %+v, want TerminalPage clamped to 6", sp)
	}
	if got := strings.Join(palettes, ","); got != "default,catppuccin,dracula,nord,console" {
		t.Errorf("palettes = %s", got)
	}

	// A new run on the same store starts with the saved choices.
	var gotSpacing dashygotui.Spacing
	var gotPalette string
	var gotInterval time.Duration
	f2 := newFixture(t)
	f2.cfg.Store = st
	f2.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd {
		gotSpacing, gotPalette, gotInterval = c.Spacing(), c.PaletteID(), c.RefreshInterval()
		return nil
	}})
	d2 := f2.driver(120, 40)
	d2.Press("p")
	if gotSpacing.Section != 4 || gotPalette != "nord" || gotInterval != 2*time.Second {
		t.Errorf("restored %+v %q %v", gotSpacing, gotPalette, gotInterval)
	}
	d2.RequireContains("LIVE 2s")
}

func TestRateKeysStepTheInterval(t *testing.T) {
	f := newFixture(t)
	var got time.Duration
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd { got = c.RefreshInterval(); return nil }})
	d := f.driver(120, 40)
	d.Press("+")
	d.Press("p")
	if got != 500*time.Millisecond {
		t.Errorf("after + interval = %v, want 500ms", got)
	}
	d.RequireContains("LIVE 500ms")
	d.Press("-", "-")
	d.Press("p")
	if got != 2*time.Second {
		t.Errorf("after -- interval = %v, want 2s", got)
	}
	for range 10 {
		d.Press("-")
	}
	d.RequireContains("(slowest)")
	for range 10 {
		d.Press("+")
	}
	d.RequireContains("(fastest)")
}

func TestPalettesAreConfigurable(t *testing.T) {
	f := newFixture(t)
	f.cfg.Palette = "dracula"
	var id string
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd { id = c.PaletteID(); return nil }})
	d := f.driver(120, 40)
	d.Press("p")
	if id != "dracula" {
		t.Errorf("palette = %q, want the configured one", id)
	}
}

func TestGotoPageAndPage(t *testing.T) {
	f := newFixture(t)
	var cur string
	var ok, missing bool
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "g", Run: func(c *ctx) tea.Cmd {
		ok = c.GotoPage("theme")
		missing = c.GotoPage("nope")
		cur = c.Page()
		return nil
	}})
	d := f.driver(120, 40)
	d.Press("g")
	if !ok || missing || cur != "theme" {
		t.Errorf("GotoPage ok=%v missing=%v page=%q", ok, missing, cur)
	}
	d.RequireContains("Appearance")
}

func TestConfiguredKeysOverrideTheDefaults(t *testing.T) {
	f := newFixture(t)
	km := dashygotui.DefaultKeyMap()
	km.Quit.SetKeys("x")
	f.cfg.Keys = dashygotui.KeyMap{Quit: km.Quit}
	d := f.driver(120, 40)
	d.Press("q")
	if d.Quit() {
		t.Error("q is no longer bound to quit")
	}
	d.Press("x")
	if !d.Quit() {
		t.Error("x should quit")
	}
	// Unset bindings keep their defaults.
	d2 := f.driver(120, 40)
	d2.Press("tab")
	d2.RequireContains("Quick Actions")
}

func TestPageWithoutPanelsRenders(t *testing.T) {
	f := newFixture(t)
	f.cfg.Pages = append(f.cfg.Pages, dashygotui.Page[state]{ID: "empty", Title: "Empty"})
	d := f.driver(120, 40)
	d.Press("shift+tab")
	d.RequireContains("Empty")
}

func TestCustomSectionBadgeAndNoun(t *testing.T) {
	f := newFixture(t)
	f.cfg.Sections[0].Noun = "action"
	f.cfg.Sections[1].Badge = func(c *ctx) string { return "[custom]" }
	d := f.driver(120, 40)
	d.Press("tab")
	d.RequireContains("[2 actions]", "[custom]")
}

func TestWidgetMinHeightAndBadge(t *testing.T) {
	f := newFixture(t)
	f.reg.Widget(widget{Key: "tall", Page: "monitor", Order: 5, Title: "Tall", MinHeight: 6,
		Badge: func(c *ctx) string { return "BADGE" },
		Render: func(c *ctx, size dashygotui.Size) []string {
			return []string{fmt.Sprintf("inner %dx%d", size.Cols, size.Rows)}
		}})
	d := f.driver(120, 40)
	d.RequireContains("Tall", "BADGE", "inner ")
}

func TestOptionsAreOrderedByOrderThenNumber(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	c := baseConfig(r)
	r.Option(option{Key: "c", Section: "actions", Number: 3, Label: "C", Run: noop})
	r.Option(option{Key: "a", Section: "actions", Number: 9, Order: 1, Label: "A", Run: noop})
	r.Option(option{Key: "b", Section: "actions", Number: 2, Label: "B", Run: noop})
	d := dashtest.New(t, c, 120, 40)
	d.Press("tab")
	var order []string
	for _, line := range d.Lines() {
		for _, l := range []string{"[9] A", "[2] B", "[3] C"} {
			if strings.Contains(line, l) {
				order = append(order, l)
			}
		}
	}
	if got := strings.Join(order, " | "); got != "[9] A | [2] B | [3] C" {
		t.Errorf("order = %s, want Order first, then Number", got)
	}
}

func TestRegistrationOrderDoesNotAffectLayout(t *testing.T) {
	build := func(reverse bool) string {
		r := dashygotui.NewRegistry[state]()
		opts := []option{validOption("a", 1), validOption("b", 2), validOption("c", 3)}
		if reverse {
			opts[0], opts[2] = opts[2], opts[0]
		}
		for _, o := range opts {
			r.Option(o)
		}
		d := dashtest.New(t, baseConfig(r), 120, 40)
		d.Press("tab")
		return d.Screen()
	}
	if build(false) != build(true) {
		t.Error("init() order must not change the screen")
	}
}

func TestWidthsOfEveryFrame(t *testing.T) {
	f := newFixture(t)
	f.reg.View(dashygotui.View[state]{ID: "v", Title: "View", Body: func(*ctx, dashygotui.Size) []string { return []string{"body"} }})
	f.reg.Option(option{Key: "go", Section: "actions", Number: 3, Label: "Go", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("v"); return nil }})
	f.reg.Option(option{Key: "pop", Section: "actions", Number: 4, Label: "Pop", Run: func(c *ctx) tea.Cmd { c.OpenModal(&stubModal{}); return nil }})

	sizes := [][2]int{{0, 0}, {1, 1}, {5, 3}, {39, 9}, {40, 10}, {41, 11}, {60, 12}, {79, 20}, {80, 24}, {100, 30}, {101, 40}, {200, 60}}
	for _, sz := range sizes {
		for _, scenario := range []struct {
			name  string
			setup func(d *dashtest.Driver)
		}{
			{"monitor", func(d *dashtest.Driver) {}},
			{"setup", func(d *dashtest.Driver) { d.Press("tab") }},
			{"theme", func(d *dashtest.Driver) { d.Press("tab", "tab") }},
			{"view", func(d *dashtest.Driver) { d.Type("3"); d.Press("enter") }},
			{"modal", func(d *dashtest.Driver) { d.Type("4"); d.Press("enter") }},
		} {
			t.Run(fmt.Sprintf("%s_%dx%d", scenario.name, sz[0], sz[1]), func(t *testing.T) {
				d := dashtest.New(t, f.cfg, 120, 40)
				scenario.setup(d)
				d.Resize(sz[0], sz[1])
				raw := d.Raw()
				if sz[0] <= 0 || sz[1] <= 0 {
					if raw != "" {
						t.Errorf("a zero-size terminal must render nothing, got %q", raw)
					}
					return
				}
				lines := strings.Split(raw, "\n")
				if len(lines) != sz[1] {
					t.Fatalf("%d rows, want %d", len(lines), sz[1])
				}
				for i, line := range lines {
					if w := text.Width(line); w != sz[0] {
						t.Fatalf("row %d is %d cells wide, want %d: %q", i, w, sz[0], dashtest.Plain(line))
					}
				}
			})
		}
	}
}

func TestTooSmallShowsANotice(t *testing.T) {
	f := newFixture(t)
	d := f.driver(30, 8)
	d.RequireContains("Terminal too small", "30x8")
	d.Resize(120, 40)
	d.RequireContains("Monitor")
}

func TestSpacingChangesTheLayout(t *testing.T) {
	f := newFixture(t)
	tight := dashygotui.Spacing{Section: 0}
	f.cfg.Spacing = &tight
	d := f.driver(120, 40)
	d.Press("tab")
	a := d.Screen()
	loose := dashygotui.Spacing{TerminalPage: 3, PageSection: 2, Section: 6}
	f.cfg.Spacing = &loose
	d = f.driver(120, 40)
	d.Press("tab")
	if a == d.Screen() {
		t.Error("different spacing must change the frame")
	}
}

func TestCountersAreMonotonic(t *testing.T) {
	f := newFixture(t)
	var tokens []int
	f.reg.OnResult(func(c *ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
		tokens = append(tokens, res.Token)
		return true, nil
	})
	f.reg.Option(option{Key: "job", Section: "actions", Number: 3, Label: "Job", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("j", "", func(context.Context) (string, error) { return "", nil })
	}})
	d := f.driver(120, 40)
	d.Type("3")
	for range 3 {
		d.Press("enter")
	}
	if len(tokens) != 3 || tokens[0] >= tokens[1] || tokens[1] >= tokens[2] {
		t.Errorf("tokens = %v, want strictly increasing", tokens)
	}
}

func TestAHandledResultStillClearsThePendingMessage(t *testing.T) {
	f := newFixture(t)
	f.reg.OnResult(func(c *ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
		return res.Tag == "quiet", nil // handled, and says nothing itself
	})
	f.reg.Option(option{Key: "quiet", Section: "actions", Number: 3, Label: "Quiet", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("quiet", "Working on it...", func(context.Context) (string, error) { return "done", nil })
	}})
	f.reg.Option(option{Key: "loud", Section: "actions", Number: 4, Label: "Loud", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("loud", "Also working...", func(context.Context) (string, error) { return "finished", nil })
	}})
	d := f.driver(160, 40)
	d.Type("3")
	d.Press("enter")
	d.RequireAbsent("Working on it...") // the hook handled the result; the message must not linger
	d.RequireContains("[Tab]")          // the hint bar is back

	d.Type("4")
	d.Press("enter")
	d.RequireContains("✓ finished") // an unhandled result replaces it with the outcome
	d.RequireAbsent("Also working...")
}

func TestAnUnrelatedFlashSurvivesATaskFinishing(t *testing.T) {
	f := newFixture(t)
	f.reg.Option(option{Key: "job", Section: "actions", Number: 3, Label: "Job", Run: func(c *ctx) tea.Cmd {
		task := c.RunTask("job", "Working...", func(context.Context) (string, error) { return "", nil })
		c.Flash("something else") // replaces the pending message before the task ends
		return task
	}})
	f.reg.OnResult(func(*ctx, dashygotui.TaskResult) (bool, tea.Cmd) { return true, nil })
	d := f.driver(160, 40)
	d.Type("3")
	d.Press("enter")
	d.RequireContains("something else")
}
