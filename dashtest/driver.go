// Package dashtest drives a dashygotui dashboard without a terminal, for tests
// of the library, of an application's dashboard, and of its addons.
//
// A [Driver] feeds key presses and resizes to the dashboard, executes the
// commands it returns, and lets the test read the rendered screen as plain
// text. Time is fake: the clock only moves when the test says so, and timers
// (refresh ticks, poll schedules) fire only on [Driver.Tick], so a test never
// sleeps and never races.
package dashtest

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
)

// Epoch returns the fixed time a Driver's clock starts at.
func Epoch() time.Time { return time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC) }

// ansi matches the escape sequences that style terminal output.
var ansi = regexp.MustCompile(`\x1b\[[0-9;:?]*[ -/]*[@-~]`)

// Driver runs one dashboard under test.
type Driver struct {
	tb     testing.TB
	model  tea.Model
	now    time.Time
	timers []func(time.Time) tea.Msg
	quit   bool
	cols   int
	rows   int
	notify func()
}

// New builds the dashboard described by cfg, sized cols x rows, starts it, and
// runs its first commands (including the first poll). It overrides cfg.Clock
// and cfg.Timer with the Driver's fake ones. A configuration error fails the
// test.
func New[S any](tb testing.TB, cfg dashygotui.Config[S], cols, rows int) *Driver {
	tb.Helper()
	d := &Driver{tb: tb, now: Epoch(), cols: cols, rows: rows}
	cfg.Clock = func() time.Time { return d.now }
	cfg.Timer = func(_ time.Duration, fn func(time.Time) tea.Msg) tea.Cmd {
		d.timers = append(d.timers, fn)
		return func() tea.Msg { return nil } // the timer fires on Tick, not by itself
	}

	dash, err := dashygotui.New(cfg)
	if err != nil {
		tb.Fatalf("dashtest: building the dashboard: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	tb.Cleanup(cancel)

	d.model = dash.Model(ctx)
	d.notify = dash.Notify
	tb.Cleanup(dash.Attach(d))
	d.Send(tea.WindowSizeMsg{Width: cols, Height: rows})
	d.run(d.model.Init())
	return d
}

// Send delivers msg to the dashboard and executes the commands that come back.
func (d *Driver) Send(msg tea.Msg) {
	d.tb.Helper()
	switch m := msg.(type) {
	case nil:
		return
	case tea.BatchMsg:
		for _, cmd := range m {
			d.run(cmd)
		}
		return
	case tea.QuitMsg:
		d.quit = true
		return
	case tea.WindowSizeMsg:
		d.cols, d.rows = m.Width, m.Height
	}
	next, cmd := d.model.Update(msg)
	d.model = next
	d.run(cmd)
}

// run executes cmd and delivers its message. Commands must return promptly:
// timers are faked, so only the application's own Poll and task functions run
// here.
func (d *Driver) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	d.Send(cmd())
}

// Press sends key presses by name: "enter", "tab", "shift+tab", "esc", "up",
// "down", "left", "right", "pgup", "pgdown", "home", "end", "space",
// "backspace", "delete", "ctrl+<letter>", "shift+<letter>" (or the capital
// letter itself), or a single character such as "q" or "7".
func (d *Driver) Press(keys ...string) {
	d.tb.Helper()
	for _, k := range keys {
		d.Send(KeyPress(k))
	}
}

// Type sends each character of each argument as its own key press, so
// Type("1", "2") and Type("12") are the same.
func (d *Driver) Type(parts ...string) {
	d.tb.Helper()
	for _, s := range parts {
		for _, r := range s {
			d.Send(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
	}
}

// Resize changes the terminal size.
func (d *Driver) Resize(cols, rows int) {
	d.tb.Helper()
	d.Send(tea.WindowSizeMsg{Width: cols, Height: rows})
}

// Notify calls Dashboard.Notify, as a background goroutine of the application
// would: the dashboard polls at once.
func (d *Driver) Notify() {
	d.tb.Helper()
	d.notify()
}

// Tick fires every timer that is waiting, once: a refresh tick, a scheduled
// poll. Timers they set in turn wait for the next Tick.
func (d *Driver) Tick() {
	d.tb.Helper()
	pending := d.timers
	d.timers = nil
	for _, fn := range pending {
		d.Send(fn(d.now))
	}
}

// Advance moves the fake clock forward by delta and fires the waiting timers.
func (d *Driver) Advance(delta time.Duration) {
	d.tb.Helper()
	d.now = d.now.Add(delta)
	d.Tick()
}

// Now returns the fake clock's time.
func (d *Driver) Now() time.Time { return d.now }

// Quit reports whether the dashboard asked to quit.
func (d *Driver) Quit() bool { return d.quit }

// Exit returns why the dashboard asked to quit: the reason and message given to
// Context.Quit, or the zero value after an ordinary quit or before any quit.
func (d *Driver) Exit() dashygotui.ExitInfo {
	if m, ok := d.model.(interface{ ExitInfo() dashygotui.ExitInfo }); ok {
		return m.ExitInfo()
	}
	return dashygotui.ExitInfo{}
}

// Model returns the underlying Bubble Tea model, for tests that need to go
// around the driver.
func (d *Driver) Model() tea.Model { return d.model }

// Size returns the current terminal size.
func (d *Driver) Size() (cols, rows int) { return d.cols, d.rows }

// Raw returns the rendered frame with its ANSI styling.
func (d *Driver) Raw() string {
	d.tb.Helper()
	return d.model.View().Content
}

// Screen returns the rendered frame as plain text, rows joined by newlines.
func (d *Driver) Screen() string {
	d.tb.Helper()
	return Plain(d.Raw())
}

// Lines returns the rendered frame as plain-text rows.
func (d *Driver) Lines() []string {
	d.tb.Helper()
	if s := d.Screen(); s != "" {
		return strings.Split(s, "\n")
	}
	return nil
}

// Contains reports whether the plain-text screen contains s.
func (d *Driver) Contains(s string) bool {
	d.tb.Helper()
	return strings.Contains(d.Screen(), s)
}

// RequireContains fails the test, showing the screen, unless every s is on it.
func (d *Driver) RequireContains(want ...string) {
	d.tb.Helper()
	screen := d.Screen()
	for _, s := range want {
		if !strings.Contains(screen, s) {
			d.tb.Fatalf("screen does not contain %q:\n%s", s, screen)
		}
	}
}

// RequireAbsent fails the test, showing the screen, if any s is on it.
func (d *Driver) RequireAbsent(unwanted ...string) {
	d.tb.Helper()
	screen := d.Screen()
	for _, s := range unwanted {
		if strings.Contains(screen, s) {
			d.tb.Fatalf("screen unexpectedly contains %q:\n%s", s, screen)
		}
	}
}

// Plain strips ANSI escape sequences from s.
func Plain(s string) string { return ansi.ReplaceAllString(s, "") }

// KeyPress builds the key press message for a key name; see [Driver.Press].
func KeyPress(name string) tea.KeyPressMsg {
	switch name {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	}
	if rest, ok := strings.CutPrefix(name, "shift+"); ok && len([]rune(rest)) == 1 {
		r := []rune(rest)[0]
		up := strings.ToUpper(rest)
		return tea.KeyPressMsg{Code: r, ShiftedCode: []rune(up)[0], Mod: tea.ModShift, Text: up}
	}
	if rest, ok := strings.CutPrefix(name, "ctrl+"); ok && len(rest) == 1 {
		return tea.KeyPressMsg{Code: rune(rest[0]), Mod: tea.ModCtrl}
	}
	r := []rune(name)
	if len(r) == 1 {
		return tea.KeyPressMsg{Code: r[0], Text: name}
	}
	return tea.KeyPressMsg{Text: name}
}
