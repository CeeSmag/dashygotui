package dashygotui_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/internal/collect"
)

// rawModel drives the Bubble Tea model by hand, so a test can hold a command
// back and deliver its message later: a poll or a task "in flight". The
// dashtest driver runs every command at once, which cannot show an overlap.
type rawModel struct {
	t      *testing.T
	m      tea.Model
	now    time.Time
	timers []func(time.Time) tea.Msg
}

func newRaw(t *testing.T, c cfg) (*rawModel, tea.Cmd) {
	t.Helper()
	r := &rawModel{t: t, now: dashtest.Epoch()}
	c.Clock = func() time.Time { return r.now }
	c.Timer = func(_ time.Duration, fn func(time.Time) tea.Msg) tea.Cmd {
		r.timers = append(r.timers, fn)
		return func() tea.Msg { return nil } // a timer fires only from advance
	}
	d, err := dashygotui.New(c)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	r.m = d.Model(ctx)
	r.send(tea.WindowSizeMsg{Width: 120, Height: 30})
	return r, r.m.Init()
}

// send delivers msg and returns the command that came back, without running it.
func (r *rawModel) send(msg tea.Msg) tea.Cmd {
	r.t.Helper()
	next, cmd := r.m.Update(msg)
	r.m = next
	return cmd
}

// exec runs cmd and everything that follows from it.
func (r *rawModel) exec(cmd tea.Cmd) {
	r.t.Helper()
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range msg {
			r.exec(c)
		}
	default:
		r.exec(r.send(msg))
	}
}

// messages runs cmd once and returns the messages it produced, flattening
// batches, without delivering them.
func messages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case nil:
		return nil
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range msg {
			out = append(out, messages(c)...)
		}
		return out
	default:
		return []tea.Msg{msg}
	}
}

// advance moves the clock and fires the waiting timers, delivering their
// messages but not running the commands that come back.
func (r *rawModel) advance(d time.Duration) {
	r.t.Helper()
	r.now = r.now.Add(d)
	due := r.timers
	r.timers = nil
	for _, fn := range due {
		_ = r.send(fn(r.now)) // the tick's own commands (the next timer) are not needed here
	}
}

func (r *rawModel) screen() string { return dashtest.Plain(r.m.View().Content) }

func TestPollPastItsDeadlineIsAFailureNotAFreeze(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	var calls atomic.Int32 // the abandoned call and the next one run at the same time
	c := baseConfig(collect.New[state]())
	c.PollTimeout = 20 * time.Millisecond
	c.Poll = func(context.Context) (state, error) {
		if calls.Add(1) == 1 {
			<-release // ignores its ctx: the dashboard must give up on it anyway
		}
		return state{Count: 7}, nil
	}
	d := dashtest.New(t, c, 120, 30) // without a deadline this never returns
	d.RequireContains("STALE", "poll failed: poll took longer than 20ms")

	d.Tick() // the next poll is scheduled as usual and succeeds
	d.RequireContains("LIVE 1s")
	d.RequireAbsent("STALE")
}

func TestPollTimeoutReachesThePollContext(t *testing.T) {
	t.Parallel()
	c := baseConfig(collect.New[state]())
	c.PollTimeout = 20 * time.Millisecond
	ended := make(chan error, 1)
	c.Poll = func(ctx context.Context) (state, error) {
		<-ctx.Done()
		ended <- ctx.Err()
		return state{}, ctx.Err()
	}
	d := dashtest.New(t, c, 120, 30)
	if got := <-ended; !errors.Is(got, context.DeadlineExceeded) {
		t.Fatalf("poll ctx ended with %v, want a deadline", got)
	}
	d.RequireContains("STALE")
}

func TestPollErrorIsShownUntilAPollSucceeds(t *testing.T) {
	t.Parallel()
	fail := true
	c := baseConfig(collect.New[state]())
	c.Poll = func(context.Context) (state, error) {
		if fail {
			return state{}, errors.New("backend down")
		}
		return state{Count: 1}, nil
	}
	d := dashtest.New(t, c, 120, 30)
	d.RequireContains("STALE", "poll failed: backend down")
	d.RequireAbsent("LIVE 1s")

	d.Advance(10 * time.Second) // the message times out; the marker stays while polls keep failing
	d.RequireContains("STALE")
	d.RequireAbsent("poll failed")

	fail = false
	d.Tick()
	d.RequireContains("LIVE 1s")
	d.RequireAbsent("STALE")
}

func TestRefreshDuringAPollPollsAgain(t *testing.T) {
	t.Parallel()
	r := collect.New[state]()
	r.Row("actions", dashygotui.Hotkey[state]{Key: "a", Item: option{Label: "Apply", Run: func(c *ctx) tea.Cmd { return c.Refresh() }}})
	polls := 0
	c := baseConfig(r)
	c.Poll = func(context.Context) (state, error) { polls++; return state{Count: polls}, nil }
	m, start := newRaw(t, c) // the first poll is now in flight: start holds it

	m.exec(m.send(dashtest.KeyPress("a"))) // the change happens while that poll may already have read the data
	if polls != 0 {
		t.Fatalf("polls = %d before the first poll returned, want 0", polls)
	}
	m.exec(start)
	if polls != 2 {
		t.Fatalf("polls = %d, want 2: the poll in flight may predate the change, so it must be repeated", polls)
	}
}

func TestASecondTaskKeepsItsMessageWhenTheFirstEnds(t *testing.T) {
	t.Parallel()
	r := collect.New[state]()
	task := func(tag, pending string) func(*ctx) tea.Cmd {
		return func(c *ctx) tea.Cmd {
			return c.RunTask(tag, pending, func(context.Context) (string, error) { return "", nil })
		}
	}
	r.Row("actions", dashygotui.Hotkey[state]{Key: "f", Item: option{Label: "First", Run: task("first", "Scanning trash...")}})
	r.Row("actions", dashygotui.Hotkey[state]{Key: "g", Item: option{Label: "Second", Run: task("second", "Restarting...")}})
	m, start := newRaw(t, baseConfig(r))
	m.exec(start)

	first := m.send(dashtest.KeyPress("f"))
	second := m.send(dashtest.KeyPress("g"))
	if !strings.Contains(m.screen(), "Restarting...") {
		t.Fatalf("the newest task's message is not shown:\n%s", m.screen())
	}
	for _, msg := range messages(first) {
		m.exec(m.send(msg))
	}
	if !strings.Contains(m.screen(), "Restarting...") {
		t.Fatalf("the second task is still running but its message is gone:\n%s", m.screen())
	}
	for _, msg := range messages(second) {
		m.exec(m.send(msg))
	}
	if s := m.screen(); strings.Contains(s, "Restarting...") || strings.Contains(s, "Scanning trash...") {
		t.Fatalf("a task message outlived its task:\n%s", s)
	}
}

func TestATaskMessageOutlastsTheFlashDuration(t *testing.T) {
	t.Parallel()
	r := collect.New[state]()
	r.Row("actions", dashygotui.Hotkey[state]{Key: "s", Item: option{Label: "Slow", Run: func(c *ctx) tea.Cmd {
		return c.RunTask("slow", "Copying...", func(context.Context) (string, error) { return "", nil })
	}}})
	m, start := newRaw(t, baseConfig(r))
	m.exec(start)
	_ = m.send(dashtest.KeyPress("s")) // the task never finishes in this test
	m.advance(time.Minute)
	if !strings.Contains(m.screen(), "Copying...") {
		t.Fatalf("a running task's message disappeared after a minute:\n%s", m.screen())
	}
}

func TestEveryMessageHookSeesEveryMessage(t *testing.T) {
	t.Parallel()
	r := collect.New[state]()
	var seen []string
	for _, name := range []string{"first", "second"} {
		r.OnMsg(func(_ *ctx, msg tea.Msg) tea.Cmd {
			if _, ok := msg.(customMsg); !ok {
				return nil
			}
			seen = append(seen, name)
			return func() tea.Msg { return nil } // a hook with a command must not hide the message from the next one
		})
	}
	d := dashtest.New(t, baseConfig(r), 120, 30)
	d.Send(customMsg{n: 1})
	if strings.Join(seen, ",") != "first,second" {
		t.Fatalf("hooks that saw the message: %v, want both", seen)
	}
}

func TestPollFailureDoesNotHideAnAddonMessage(t *testing.T) {
	t.Parallel()
	r := collect.New[state]()
	fail := false
	r.Row("actions", dashygotui.Hotkey[state]{Key: "x", Item: option{Label: "Break", Run: func(c *ctx) tea.Cmd { fail = true; c.Flash("Outage started"); return c.Refresh() }}})
	c := baseConfig(r)
	c.Poll = func(context.Context) (state, error) {
		if fail {
			return state{}, errors.New("backend down")
		}
		return state{}, nil
	}
	d := dashtest.New(t, c, 120, 30)
	d.Press("x")
	d.RequireContains("Outage started", "STALE")
	d.RequireAbsent("poll failed")
}

// Before the first poll returns, the dashboard holds Config.Initial: a start
// view and its Session see it, and Ready is still false.
func TestInitialIsTheStateBeforeTheFirstPoll(t *testing.T) {
	var seen []string
	r := collect.New[state]()
	r.View(dashygotui.View[state]{ID: "v", Title: "V",
		Body: func(c *ctx, _ dashygotui.Size) []string { return []string{fmt.Sprintf("count=%d", c.State().Count)} },
		Session: func(c *ctx) func() {
			seen = append(seen, fmt.Sprintf("count=%d ready=%v", c.State().Count, c.Ready()))
			return nil
		}})
	c := baseConfig(r)
	c.Initial = state{Count: 7}
	c.StartView = []string{"v"}
	fail := true
	c.Poll = func(context.Context) (state, error) {
		if fail {
			return state{}, errors.New("not yet")
		}
		return state{Count: 8}, nil
	}
	d := dashtest.New(t, c, 100, 24)
	if strings.Join(seen, ",") != "count=7 ready=false" {
		t.Fatalf("the start view's session saw %q, want Initial before the first poll", seen)
	}
	d.RequireContains("count=7") // a failed poll keeps Initial
	fail = false
	d.Tick()
	d.RequireContains("count=8")
}
