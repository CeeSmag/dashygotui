package dashtest_test

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
)

type state struct{ N int }

type ctx = dashygotui.Context[state]

// failure is what a recording TB panics with, so a test can assert that a
// helper failed the test.
type failure struct{ msg string }

// recorder is a testing.TB whose Fatalf panics instead of ending the test.
type recorder struct {
	testing.TB
	cleanups []func()
}

func (r *recorder) Fatalf(format string, args ...any) {
	panic(failure{msg: strings.TrimSpace(strings.ReplaceAll(format, "%", "") + " " + strings.Join(toStrings(args), " "))})
}
func (r *recorder) Helper()          {}
func (r *recorder) Cleanup(f func()) { r.cleanups = append(r.cleanups, f) }

func toStrings(args []any) []string {
	out := make([]string, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case string:
			out[i] = v
		case error:
			out[i] = v.Error()
		default:
			out[i] = "?"
		}
	}
	return out
}

// mustFail runs f and reports whether it called Fatalf on rec.
func mustFail(t *testing.T, f func()) (msg string) {
	t.Helper()
	defer func() {
		r := recover()
		fl, ok := r.(failure)
		if !ok {
			t.Fatalf("expected the helper to fail the test, recovered %v", r)
		}
		msg = fl.msg
	}()
	f()
	return ""
}

func config(opts ...func(*dashygotui.Config[state])) dashygotui.Config[state] {
	r := dashygotui.NewRegistry[state]()
	r.Option(dashygotui.Option[state]{Key: "inc", Section: "s", Number: 1, Label: "Increment", Run: func(*ctx) tea.Cmd { return nil }})
	c := dashygotui.Config[state]{
		Title:    "Probe",
		Registry: r,
		Pages:    []dashygotui.Page[state]{{ID: "p", Title: "Page"}},
		Sections: []dashygotui.SectionSpec[state]{{ID: "s", Page: "p", Title: "Section"}},
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func TestKeyPressNamesRoundTrip(t *testing.T) {
	names := []string{
		"enter", "tab", "shift+tab", "esc", "up", "down", "left", "right",
		"pgup", "pgdown", "home", "end", "ctrl+q", "ctrl+c", "q", "7", "+", "-",
	}
	for _, n := range names {
		if got := dashtest.KeyPress(n).String(); got != n {
			t.Errorf("KeyPress(%q).String() = %q", n, got)
		}
	}
	if got := dashtest.KeyPress("space").String(); got != "space" && got != " " {
		t.Errorf("space = %q", got)
	}
}

func TestPlain(t *testing.T) {
	in := "\x1b[38;2;1;2;3mred\x1b[0m \x1b[1;32mbold\x1b[m plain"
	if got := dashtest.Plain(in); got != "red bold plain" {
		t.Errorf("Plain = %q", got)
	}
	if got := dashtest.Plain("no escapes"); got != "no escapes" {
		t.Errorf("Plain = %q", got)
	}
}

func TestDriverStartsAndRenders(t *testing.T) {
	d := dashtest.New(t, config(), 100, 30)
	cols, rows := d.Size()
	if cols != 100 || rows != 30 {
		t.Errorf("Size = %dx%d", cols, rows)
	}
	d.RequireContains("Probe", "Page", "Section", "[1] Increment")
	d.RequireAbsent("definitely not on screen")
	if !d.Contains("Probe") || d.Contains("zzz") {
		t.Error("Contains is wrong")
	}
	if got := len(d.Lines()); got != 30 {
		t.Errorf("Lines = %d, want 30", got)
	}
	if strings.Contains(d.Screen(), "\x1b") {
		t.Error("Screen must be plain text")
	}
	if !strings.Contains(d.Raw(), "\x1b") {
		t.Error("Raw keeps the styling")
	}
	if d.Model() == nil {
		t.Error("Model is nil")
	}
	d.Resize(60, 20)
	if c, r := d.Size(); c != 60 || r != 20 {
		t.Errorf("after Resize: %dx%d", c, r)
	}
	if got := len(d.Lines()); got != 20 {
		t.Errorf("Lines after Resize = %d", got)
	}
}

func TestZeroSizeHasNoLines(t *testing.T) {
	d := dashtest.New(t, config(), 0, 0)
	if d.Lines() != nil || d.Screen() != "" {
		t.Errorf("a 0x0 terminal renders nothing, got %q", d.Screen())
	}
}

func TestClockOnlyMovesWhenTold(t *testing.T) {
	d := dashtest.New(t, config(), 100, 30)
	if !d.Now().Equal(dashtest.Epoch()) {
		t.Errorf("Now = %v, want the epoch", d.Now())
	}
	d.RequireContains("12:00:00")
	d.Advance(75 * time.Second)
	if want := dashtest.Epoch().Add(75 * time.Second); !d.Now().Equal(want) {
		t.Errorf("Now = %v, want %v", d.Now(), want)
	}
	d.RequireContains("12:01:15")
}

func TestTickFiresEachTimerOnce(t *testing.T) {
	polls := 0
	var ticks []time.Time
	d := dashtest.New(t, config(func(c *dashygotui.Config[state]) {
		c.Poll = func(context.Context) (state, error) { polls++; return state{N: polls}, nil }
		c.Registry.OnTick(func(_ *ctx, now time.Time) tea.Cmd { ticks = append(ticks, now); return nil })
	}), 100, 30)

	if polls != 1 || len(ticks) != 0 {
		t.Fatalf("before Tick: polls=%d ticks=%d, want the first poll only", polls, len(ticks))
	}
	d.Tick()
	if polls != 2 || len(ticks) != 1 {
		t.Errorf("after one Tick: polls=%d ticks=%d, want 2 and 1", polls, len(ticks))
	}
	d.Tick()
	if polls != 3 || len(ticks) != 2 {
		t.Errorf("after two Ticks: polls=%d ticks=%d, want 3 and 2", polls, len(ticks))
	}
}

func TestPressAndTypeDeliverKeys(t *testing.T) {
	var got []string
	d := dashtest.New(t, config(func(c *dashygotui.Config[state]) {
		c.Registry.Shortcut(dashygotui.Shortcut[state]{Key: "x", Run: func(*ctx) tea.Cmd { got = append(got, "x"); return nil }})
		c.Registry.Shortcut(dashygotui.Shortcut[state]{Key: "y", Run: func(*ctx) tea.Cmd { got = append(got, "y"); return nil }})
	}), 100, 30)
	d.Press("x", "y")
	d.Type("yx", "x")
	if strings.Join(got, "") != "xyyxx" {
		t.Errorf("keys = %q", got)
	}
}

func TestQuitIsReported(t *testing.T) {
	d := dashtest.New(t, config(), 100, 30)
	if d.Quit() {
		t.Error("not quit yet")
	}
	d.Press("q")
	if !d.Quit() {
		t.Error("q should quit")
	}
}

func TestSendIgnoresNilAndUnknownMessages(t *testing.T) {
	d := dashtest.New(t, config(), 100, 30)
	d.Send(nil)
	d.Send(struct{ weird bool }{true})
	d.RequireContains("Probe")
}

func TestBatchedCommandsAllRun(t *testing.T) {
	var ran []string
	d := dashtest.New(t, config(func(c *dashygotui.Config[state]) {
		c.Registry.Shortcut(dashygotui.Shortcut[state]{Key: "z", Run: func(*ctx) tea.Cmd {
			return tea.Batch(
				func() tea.Msg { ran = append(ran, "one"); return nil },
				func() tea.Msg { ran = append(ran, "two"); return nil },
			)
		}})
	}), 100, 30)
	d.Press("z")
	if strings.Join(ran, ",") != "one,two" {
		t.Errorf("ran = %v", ran)
	}
}

func TestRequireHelpersFailTheTest(t *testing.T) {
	rec := &recorder{TB: t}
	d := dashtest.New(rec, config(), 100, 30)

	msg := mustFail(t, func() { d.RequireContains("nowhere to be found") })
	if !strings.Contains(msg, "does not contain") {
		t.Errorf("message = %q", msg)
	}
	msg = mustFail(t, func() { d.RequireAbsent("Probe") })
	if !strings.Contains(msg, "unexpectedly contains") {
		t.Errorf("message = %q", msg)
	}
	d.RequireContains("Probe") // and the passing case does not fail
}

func TestNewFailsOnABrokenConfig(t *testing.T) {
	rec := &recorder{TB: t}
	msg := mustFail(t, func() {
		dashtest.New(rec, dashygotui.Config[state]{}, 100, 30)
	})
	if !strings.Contains(msg, "building the dashboard") {
		t.Errorf("message = %q", msg)
	}
}

func TestEachDriverHasItsOwnClockAndTimers(t *testing.T) {
	a := dashtest.New(t, config(), 100, 30)
	b := dashtest.New(t, config(), 100, 30)
	a.Advance(time.Hour)
	if !b.Now().Equal(dashtest.Epoch()) {
		t.Error("drivers must not share a clock")
	}
}
