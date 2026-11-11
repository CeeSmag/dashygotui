package dashygotui_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/text"
)

// sink collects the messages a Sender is given.
type sink struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (s *sink) Send(m tea.Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, m)
}

func (s *sink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

func TestNotifyPollsAtOnce(t *testing.T) {
	f := newFixture(t)
	d := f.driver(120, 40)
	before := f.polls
	d.Notify() // no tick, no key: the outside world pushed
	if f.polls != before+1 {
		t.Fatalf("polls = %d, want %d", f.polls, before+1)
	}
	d.RequireContains(fmt.Sprintf("count=%d", f.polls))
}

func TestNotifyDuringAPollPollsAgain(t *testing.T) {
	f := newFixture(t)
	var drv *dashtest.Driver
	f.cfg.Poll = func(context.Context) (state, error) {
		f.polls++
		if f.polls == 2 && drv != nil {
			drv.Notify() // the data changed while this poll was reading it
		}
		return state{Count: f.polls}, nil
	}
	drv = f.driver(120, 40)
	drv.Advance(time.Second) // poll 2, which notifies, then one more poll
	if f.polls != 3 {
		t.Fatalf("polls = %d, want 3 (the notification during poll 2 must not be lost)", f.polls)
	}
}

func TestNotifyIsHarmlessWhenNotRunning(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	dash, err := dashygotui.New(baseConfig(r))
	if err != nil {
		t.Fatal(err)
	}
	dash.Notify() // not attached: must neither block nor panic
}

func TestAttachAndDetach(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	dash, err := dashygotui.New(baseConfig(r))
	if err != nil {
		t.Fatal(err)
	}
	s := &sink{}
	detach := dash.Attach(s)
	dash.Notify()
	dash.Notify()
	if s.count() != 2 {
		t.Fatalf("delivered %d messages, want 2", s.count())
	}
	detach()
	dash.Notify()
	if s.count() != 2 {
		t.Fatalf("a detached dashboard still delivered (%d messages)", s.count())
	}
}

func TestNotifyFromManyGoroutines(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	dash, err := dashygotui.New(baseConfig(r))
	if err != nil {
		t.Fatal(err)
	}
	s := &sink{}
	defer dash.Attach(s)()
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				dash.Notify()
			}
		}()
	}
	wg.Wait()
	if s.count() != 400 {
		t.Fatalf("delivered %d, want 400", s.count())
	}
}

// longView registers a scrolling view of n numbered lines and a shortcut "v"
// that opens it. It records the sizes Body was asked for.
func longView(f *fixture, n int, sizes *[]dashygotui.Size) {
	f.reg.View(dashygotui.View[state]{
		ID: "long", Title: "Long", Mode: "LONG", Scroll: true,
		Body: func(_ *ctx, size dashygotui.Size) []string {
			if sizes != nil {
				*sizes = append(*sizes, size)
			}
			out := make([]string, n)
			for i := range out {
				out[i] = fmt.Sprintf("row-%03d", i+1)
			}
			return out
		},
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "v", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("long"); return nil }})
}

func TestScrollingViewScrollsAndDrawsABar(t *testing.T) {
	f := newFixture(t)
	longView(f, 100, nil)
	d := f.driver(60, 20) // body = 18 rows
	d.Press("v")
	d.RequireContains("row-001", "row-018", "Scroll")
	d.RequireAbsent("row-019")
	if !strings.Contains(d.Screen(), "■") {
		t.Errorf("a scrollable body should draw a scroll bar:\n%s", d.Screen())
	}

	d.Press("down", "down", "down")
	d.RequireContains("row-004", "row-021")
	d.RequireAbsent("row-001")
	d.Press("pgdown")
	d.RequireAbsent("row-004")
	d.Press("end")
	d.RequireContains("row-100")
	d.RequireAbsent("row-001")
	d.Press("down", "down") // stops at the end
	d.RequireContains("row-100")
	d.Press("home")
	d.RequireContains("row-001")
	d.Press("up") // stops at the top
	d.RequireContains("row-001")
}

func TestScrollBarThumbMovesWithThePosition(t *testing.T) {
	f := newFixture(t)
	longView(f, 100, nil)
	d := f.driver(60, 20)
	d.Press("v")
	thumbRows := func() (first, last int) {
		first, last = -1, -1
		for i, l := range d.Lines() {
			if strings.HasSuffix(strings.TrimRight(l, " "), "■") {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		return first, last
	}
	top, _ := thumbRows()
	d.Press("end")
	_, bottom := thumbRows()
	if top < 0 || bottom < 0 {
		t.Fatalf("no scroll bar thumb found:\n%s", d.Screen())
	}
	if bottom <= top+4 {
		t.Errorf("the thumb should travel: top at %d, bottom at %d", top, bottom)
	}
}

func TestScrollingViewThatFitsHasNoBarAndFullWidth(t *testing.T) {
	f := newFixture(t)
	var sizes []dashygotui.Size
	longView(f, 5, &sizes)
	d := f.driver(60, 20)
	d.Press("v")
	d.RequireContains("row-005")
	if len(sizes) == 0 {
		t.Fatal("Body was never called")
	}
	if strings.Contains(d.Screen(), "■") {
		t.Error("nothing to scroll, so no scroll bar")
	}
	for _, s := range sizes {
		if s.Cols != 60 {
			t.Errorf("Body was given %d columns; with nothing to scroll it should get all 60", s.Cols)
		}
	}
	d.RequireAbsent("Scroll") // and no scroll hint
}

func TestScrollingBodyLosesOneColumnForTheBar(t *testing.T) {
	f := newFixture(t)
	var sizes []dashygotui.Size
	longView(f, 100, &sizes)
	d := f.driver(60, 20)
	d.Press("v")
	d.Screen() // Body runs when the frame is drawn
	if len(sizes) == 0 {
		t.Fatal("Body was never called")
	}
	last := sizes[len(sizes)-1]
	if last.Cols != 59 || last.Rows != 18 {
		t.Errorf("Body size = %+v, want 59x18", last)
	}
}

func TestScrollingViewStartsAtTheTopEachTime(t *testing.T) {
	f := newFixture(t)
	longView(f, 100, nil)
	d := f.driver(60, 20)
	d.Press("v", "end")
	d.Press("esc", "v")
	d.RequireContains("row-001")
}

func TestScrollingViewSurvivesEverySize(t *testing.T) {
	f := newFixture(t)
	longView(f, 100, nil)
	d := f.driver(60, 20)
	d.Press("v", "pgdown", "end")
	for _, sz := range [][2]int{{1, 1}, {20, 6}, {39, 9}, {40, 10}, {60, 14}, {200, 80}} {
		d.Resize(sz[0], sz[1])
		d.Press("down", "pgup", "end")
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

func TestScrollNeedsABody(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.View(dashygotui.View[state]{ID: "v", Title: "V", Scroll: true,
		Status: func(*ctx, dashygotui.Size) []string { return nil }})
	if msg := errText(t, withRegistry(r)); !strings.Contains(msg, "Scroll") {
		t.Errorf("error = %q, want it to name Scroll", msg)
	}
}

func withRegistry(r *reg) cfg { return baseConfig(r) }

func TestViewHooksFireOnOpenAndBack(t *testing.T) {
	f := newFixture(t)
	hook := func(name string) func(*ctx) tea.Cmd {
		return func(*ctx) tea.Cmd { f.log = append(f.log, name); return nil }
	}
	f.reg.View(dashygotui.View[state]{
		ID: "a", Title: "A", Body: func(*ctx, dashygotui.Size) []string { return []string{"view a"} },
		OnEnter: hook("enter a"), OnLeave: hook("leave a"),
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "a", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("a"); return nil }})
	d := f.driver(80, 24)
	d.Press("a")
	d.Press("a") // pressing the shortcut again inside the view does not re-enter it
	d.Press("esc")
	d.Press("a", "esc")
	if got := strings.Join(f.log, ", "); got != "enter a, leave a, enter a, leave a" {
		t.Fatalf("hooks = %q", got)
	}
}

func TestViewHooksNestedAndGotoPage(t *testing.T) {
	f := newFixture(t)
	hook := func(name string) func(*ctx) tea.Cmd {
		return func(*ctx) tea.Cmd { f.log = append(f.log, name); return nil }
	}
	f.reg.View(dashygotui.View[state]{
		ID: "b", Title: "B", Body: func(*ctx, dashygotui.Size) []string { return []string{"view b"} },
		OnEnter: hook("enter b"), OnLeave: hook("leave b"),
	})
	f.reg.View(dashygotui.View[state]{
		ID: "a", Title: "A", Body: func(*ctx, dashygotui.Size) []string { return []string{"view a"} },
		OnEnter: hook("enter a"), OnLeave: hook("leave a"),
		Keys: func(c *ctx, msg tea.KeyPressMsg) (tea.Cmd, bool) {
			switch msg.String() {
			case "x":
				_ = c.OpenView("b")
				return nil, true
			case "g":
				c.GotoPage("theme")
				return nil, true
			}
			return nil, false
		},
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "a", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("a"); return nil }})
	d := f.driver(80, 24)

	step := func(want string, keys ...string) {
		t.Helper()
		f.log = nil
		d.Press(keys...)
		if got := strings.Join(f.log, ", "); got != want {
			t.Fatalf("after %v: hooks = %q, want %q", keys, got, want)
		}
	}
	step("enter a", "a")
	step("leave a, enter b", "x")
	step("leave b, enter a", "esc") // back to a: a is visible again
	step("leave a", "esc")
	step("enter a", "a")
	step("leave a", "g") // jumping to a page leaves the view
	d.RequireAbsent("view a")
}

func TestViewHookCommandsRun(t *testing.T) {
	f := newFixture(t)
	f.reg.View(dashygotui.View[state]{
		ID: "a", Title: "A", Body: func(*ctx, dashygotui.Size) []string { return []string{"view a"} },
		OnEnter: func(c *ctx) tea.Cmd {
			return c.RunTask("enter", "", func(context.Context) (string, error) { return "probe started", nil })
		},
	})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "a", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("a"); return nil }})
	d := f.driver(80, 24)
	d.Press("a")
	d.RequireContains("probe started") // the command OnEnter returned was executed
}

func fallbackConfig(f *fixture) {
	f.cfg.Pages[0].Fallback = func(c *ctx, size dashygotui.Size) []string {
		return []string{fmt.Sprintf("summary %dx%d count=%d", size.Cols, size.Rows, c.State().Count)}
	}
	f.cfg.Pages[0].FallbackBelow = dashygotui.Size{Cols: 70, Rows: 16}
}

func TestPageFallbackReplacesTheLayoutWhenSmall(t *testing.T) {
	f := newFixture(t)
	fallbackConfig(f)
	d := f.driver(120, 40)
	d.RequireContains("Stats")
	d.RequireAbsent("summary")

	d.Resize(120, 12) // too short
	d.RequireContains("summary 116x", "Monitor")
	d.RequireAbsent("Stats")

	d.Resize(60, 40) // too narrow
	d.RequireContains("summary")

	d.Resize(120, 40)
	d.RequireContains("Stats")
	d.RequireAbsent("summary")
}

func TestPageFallbackIsReadOnly(t *testing.T) {
	f := newFixture(t)
	fallbackConfig(f)
	f.cfg.Pages[1].Fallback = f.cfg.Pages[0].Fallback
	f.cfg.Pages[1].FallbackBelow = dashygotui.Size{Rows: 16}
	d := f.driver(120, 12)
	d.Press("tab") // the setup page, which also falls back at this size
	d.RequireContains("summary")
	d.Press("enter", "down", "enter")
	d.Type("1")
	d.Press("enter")
	if len(f.log) != 0 {
		t.Fatalf("options ran while hidden behind the fallback: %v", f.log)
	}
	d.Resize(120, 40) // room again: the options work
	d.Type("1")
	d.Press("enter")
	if got := strings.Join(f.log, ","); got != "backup" {
		t.Errorf("log = %q, want backup", got)
	}
}

func TestPageFallbackRespectsEachDimension(t *testing.T) {
	f := newFixture(t)
	f.cfg.Pages[0].Fallback = func(*ctx, dashygotui.Size) []string { return []string{"summary"} }
	f.cfg.Pages[0].FallbackBelow = dashygotui.Size{Rows: 20} // only the height matters
	d := f.driver(40, 40)
	d.RequireAbsent("summary")
	d.Resize(40, 15)
	d.RequireContains("summary")
}

func TestPageFallbackNeedsASize(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	c := baseConfig(r)
	c.Pages[0].Fallback = func(*ctx, dashygotui.Size) []string { return nil }
	if msg := errText(t, c); !strings.Contains(msg, "FallbackBelow") {
		t.Errorf("error = %q", msg)
	}
	c = baseConfig(dashygotui.NewRegistry[state]())
	c.Pages[0].FallbackBelow = dashygotui.Size{Cols: 10}
	if msg := errText(t, c); !strings.Contains(msg, "no Fallback") {
		t.Errorf("error = %q", msg)
	}
}

func TestPageFallbackFitsEverySize(t *testing.T) {
	f := newFixture(t)
	fallbackConfig(f)
	f.cfg.Pages[0].FallbackBelow = dashygotui.Size{Cols: 300, Rows: 300} // always
	d := f.driver(80, 24)
	for _, sz := range [][2]int{{1, 1}, {20, 6}, {39, 9}, {40, 10}, {80, 24}, {200, 80}} {
		d.Resize(sz[0], sz[1])
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
