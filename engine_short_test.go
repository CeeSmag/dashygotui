package dashygotui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/internal/collect"
	"github.com/CeeSmag/dashygotui/modal"
)

// plainView registers a one-line view and returns its ID.
func plainView(r *reg, id string) dashygotui.View[state] {
	return dashygotui.View[state]{ID: id, Title: strings.ToUpper(id) + " Title",
		Body: func(*ctx, dashygotui.Size) []string { return []string{"this is " + id} }}
}

func TestOptionActionsOpensDoModal(t *testing.T) {
	f := newFixture(t)
	f.reg.View(plainView(f.reg, "diag"))
	f.reg.Add(
		option{Label: "Open It", Opens: "diag"},
		option{Label: "Do It", Do: func(*ctx) { f.log = append(f.log, "did") }},
		option{Label: "Ask It", Modal: func(*ctx) dashygotui.Modal[state] {
			return &modal.Notice[state]{Title: "Hello Modal", Lines: []string{"from an option"}}
		}},
	)
	d := f.driver(120, 40)
	d.Press("tab")
	d.Type("4") // after the fixture's [1] and [2]: Open It, Do It, Ask It
	d.Press("enter")
	if got := strings.Join(f.log, ","); got != "did" {
		t.Fatalf("Do did not run: %q", got)
	}
	d.Type("5")
	d.Press("enter")
	d.RequireContains("Hello Modal", "from an option")
	d.Press("esc")
	d.Type("3")
	d.Press("enter")
	d.RequireContains("this is diag")
	d.Press("esc")
	d.RequireContains("Open It")
}

func TestShortcutActionsOpensDoModal(t *testing.T) {
	f := newFixture(t)
	f.reg.Add(
		plainView(f.reg, "diag"),
		dashygotui.Shortcut[state]{Key: "x", Opens: "diag"},
		dashygotui.Shortcut[state]{Key: "y", Do: func(*ctx) { f.log = append(f.log, "y") }},
		dashygotui.Shortcut[state]{Key: "z", Modal: func(*ctx) dashygotui.Modal[state] {
			return &modal.Notice[state]{Title: "Help Pop-up", Lines: []string{"help"}}
		}},
	)
	d := f.driver(120, 40)
	d.Press("y")
	if got := strings.Join(f.log, ","); got != "y" {
		t.Fatalf("log = %q", got)
	}
	d.Press("z")
	d.RequireContains("Help Pop-up")
	d.Press("esc")
	d.Press("x")
	d.RequireContains("this is diag")
}

func TestExactlyOneActionIsRequired(t *testing.T) {
	r := collect.New[state]()
	r.Add(
		option{Label: "None"},
		option{Label: "Two", Opens: "v", Do: func(*ctx) {}},
		dashygotui.Shortcut[state]{Key: "q1"},
		dashygotui.Shortcut[state]{Key: "q2", Opens: "v", Do: func(*ctx) {}},
	)
	err := validate(r)
	if err == nil {
		t.Fatal("want errors")
	}
	for _, want := range []string{`option "None" needs exactly one`, `option "Two" needs exactly one`,
		`shortcut "q1" needs exactly one`, `shortcut "q2" needs exactly one`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
}

func TestOpensMustNameARegisteredView(t *testing.T) {
	r := collect.New[state]()
	r.Add(
		option{Label: "O", Opens: "nope"},
		dashygotui.Shortcut[state]{Key: "x", Opens: "nada"},
	)
	c := baseConfig(r)
	msg := errText(t, c)
	for _, want := range []string{`option "O" opens unknown view "nope"`, `shortcut "x" opens unknown view "nada"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
}

func TestAddRejectsNil(t *testing.T) {
	r := collect.New[state]()
	r.Add(nil)
	if err := validate(r); err == nil || !strings.Contains(err.Error(), "nil row") {
		t.Errorf("Validate = %v", err)
	}
}

func TestViewEntryRegistersTheOption(t *testing.T) {
	f := newFixture(t)
	v := plainView(f.reg, "diag")
	v.Entry = &dashygotui.Entry[state]{Hotkey: "d", HotkeyHint: "Diagnostics",
		HotkeyDisplay: "D", Note: "health and logs"}
	f.reg.Add(v)
	d := f.driver(140, 40)
	d.RequireContains("[D] Diagnostics") // the footer hint, without a separate Shortcut
	d.Press("tab")
	d.RequireContains("DIAG Title (D)", "[health and logs]") // label defaults to the view title; Note is shown
	d.Press("d")
	d.RequireContains("this is diag")
}

func TestSubPageRowsAreNumberedFromOne(t *testing.T) {
	f := newFixture(t)
	noop := func(*ctx) {}
	f.reg.Add(dashygotui.Page[state]{Title: "Tools", Entry: &dashygotui.Entry[state]{Hotkey: "t"},
		Panels: []panelT{{Title: "Tools", Items: []dashygotui.Addon[state]{
			option{Label: "First Tool", Do: func(*ctx) { f.log = append(f.log, "first") }},
			option{Label: "Second Tool", Do: noop, Note: "muted note"},
			option{Label: "Third Tool", Do: noop},
		}}}})
	d := f.driver(120, 30)
	d.Press("t")
	d.RequireContains("TOOLS") // the footer's mode is the title in capitals
	d.RequireContains("[1]", "First Tool", "[2]", "Second Tool", "[muted note]", "[3]", "Third Tool")
	d.Type("1")
	d.Press("enter")
	if got := strings.Join(f.log, ","); got != "first" {
		t.Errorf("log = %q", got)
	}
}

func TestViewFollowAndFramed(t *testing.T) {
	f := newFixture(t)
	n := 60
	body := func(*ctx, dashygotui.Size) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "log line"
		}
		return out
	}
	f.reg.Add(
		dashygotui.View[state]{ID: "log", Title: "Live Log", Follow: true, Body: body},
		dashygotui.View[state]{ID: "list", Title: "Streams", Framed: true, Body: func(*ctx, dashygotui.Size) []string { return []string{"one stream"} }},
		dashygotui.Shortcut[state]{Key: "x", Opens: "log"},
		dashygotui.Shortcut[state]{Key: "y", Opens: "list"},
	)
	d := f.driver(80, 24)
	d.Press("x")
	d.RequireContains("Live Log", "rows 41-60 of 60", "LIVE LOG") // Follow = Scroll + Tail + panel; starts at the bottom
	d.Press("up")
	d.RequireContains("rows 40-59 of 60")
	d.Press("esc")
	d.Press("y")
	d.RequireContains("Streams", "one stream")
	if first := d.Lines()[1]; !strings.ContainsAny(first, "╭┌") {
		t.Errorf("Framed should draw a panel, got %q", first)
	}
}

func TestFollowAndFramedNeedABody(t *testing.T) {
	r := collect.New[state]()
	r.Add(dashygotui.View[state]{ID: "v", Title: "V", Follow: true})
	if err := validate(r); err == nil || !strings.Contains(err.Error(), "Follow or Framed") {
		t.Errorf("Validate = %v", err)
	}
}

func TestViewTagAndSession(t *testing.T) {
	f := newFixture(t)
	var events []string
	on := true
	v := plainView(f.reg, "diag")
	v.Tag = func(*ctx) string { return map[bool]string{true: "Mode: ON", false: "Mode: OFF"}[on] }
	v.Session = func(*ctx) func() {
		events = append(events, "begin")
		return func() { events = append(events, "end") }
	}
	f.reg.Add(v, dashygotui.Shortcut[state]{Key: "x", Opens: "diag"})
	d := f.driver(100, 24)
	d.Press("x")
	d.RequireContains("Mode: ON")
	on = false
	d.Tick()
	d.RequireContains("Mode: OFF")
	if got := strings.Join(events, ","); got != "begin" {
		t.Fatalf("events = %q, want begin only while the view is open", got)
	}
	d.Press("esc")
	if got := strings.Join(events, ","); got != "begin,end" {
		t.Fatalf("events = %q, want end after leaving", got)
	}
	d.Press("x") // each visit is its own session
	d.Press("esc")
	if got := strings.Join(events, ","); got != "begin,end,begin,end" {
		t.Errorf("events = %q", got)
	}
}

func TestSessionEndsWhenCoveredAndStartsWhenUncovered(t *testing.T) {
	f := newFixture(t)
	var events []string
	a := plainView(f.reg, "a")
	a.Session = func(*ctx) func() { events = append(events, "a+"); return func() { events = append(events, "a-") } }
	f.reg.Add(a, plainView(f.reg, "b"),
		dashygotui.Shortcut[state]{Key: "x", Opens: "a"},
		dashygotui.Shortcut[state]{Key: "w", Scope: dashygotui.ScopeEverywhere, Opens: "b"})
	d := f.driver(100, 24)
	d.Press("x", "w") // b opens over a
	d.Press("esc")    // back to a
	d.Press("esc")    // back to the page
	if got := strings.Join(events, ","); got != "a+,a-,a+,a-" {
		t.Errorf("events = %q", got)
	}
}

func TestPageSessionFollowsThePageOnScreen(t *testing.T) {
	f := newFixture(t)
	var events []string
	session := func(name string) func(*ctx) func() {
		return func(*ctx) func() {
			events = append(events, name+"+")
			return func() { events = append(events, name+"-") }
		}
	}
	f.cfg.Map.Pages[0].Session = session("monitor")
	f.cfg.Map.Pages[2].Session = session("theme")
	v := plainView(f.reg, "v")
	v.Keys = func(c *ctx, msg tea.KeyPressMsg) (tea.Cmd, bool) {
		if msg.String() == "g" {
			c.GotoPage("monitor")
			return nil, true
		}
		return nil, false
	}
	f.reg.Add(v,
		dashygotui.Shortcut[state]{Key: "x", Opens: "v"},
		dashygotui.Shortcut[state]{Key: "n", Do: func(c *ctx) {
			c.OpenModal(&modal.Notice[state]{Title: "Notice", Lines: []string{"hello"}})
		}})
	d := f.driver(100, 24)

	step := func(want string, keys ...string) {
		t.Helper()
		events = nil
		d.Press(keys...)
		if got := strings.Join(events, ","); got != want {
			t.Fatalf("after %v: events = %q, want %q", keys, got, want)
		}
	}
	if got := strings.Join(events, ","); got != "monitor+" {
		t.Fatalf("at start: events = %q, want the first page's session", got)
	}
	step("", "n", "esc")              // a pop-up over the page does not end it
	step("monitor-", "1")             // a number on another page moves there
	step("theme+", "tab")             // a page without a session in between
	step("theme-", "x")               // a view over the page ends it
	step("theme+", "esc")             // back on the page: a new session
	step("theme-,monitor+", "x", "g") // GotoPage from a view starts only the page it lands on
	step("monitor-,theme+", "shift+tab")
}

func TestPageSessionWaitsForTheStartView(t *testing.T) {
	f := newFixture(t)
	var events []string
	f.cfg.Map.Pages[0].Session = func(*ctx) func() {
		events = append(events, "begin")
		return func() { events = append(events, "end") }
	}
	f.reg.Add(plainView(f.reg, "v"))
	f.cfg.StartView = []string{"v"}
	d := f.driver(100, 24)
	if len(events) != 0 {
		t.Fatalf("events = %q, want none while the start view covers the page", events)
	}
	d.Press("esc")
	if got := strings.Join(events, ","); got != "begin" {
		t.Errorf("events = %q, want the page's session once the view closes", got)
	}
}

func TestHotkeyHintOnlyWhenAsked(t *testing.T) {
	f := newFixture(t)
	f.reg.Add(
		option{Label: "A", Hotkey: "u", HotkeyHint: "Upload", Do: func(*ctx) {}},
		option{Label: "B", Hotkey: "i", Do: func(*ctx) {}},
	)
	d := f.driver(160, 40)
	d.RequireContains("[u] Upload")
	d.RequireAbsent("[i]")
}

// A bounded history drops old rows as new ones arrive, so its length never
// changes; Appended is how a view scrolled back still holds its place.
func TestTailHoldsItsPlaceInABoundedHistory(t *testing.T) {
	const keep = 50
	f := newFixture(t)
	total := 100 // rows ever appended; the body shows the last `keep`
	body := func(*ctx, dashygotui.Size) []string {
		out := make([]string, keep)
		for i := range out {
			out[i] = fmt.Sprintf("event-%04d", total-keep+i+1)
		}
		return out
	}
	f.reg.Add(
		dashygotui.View[state]{ID: "t", Title: "T", Scroll: true, Tail: true, Body: body,
			Appended: func(*ctx) int { return total }},
		dashygotui.View[state]{ID: "u", Title: "U", Scroll: true, Tail: true, Body: body}, // no Appended: drifts
		dashygotui.Shortcut[state]{Key: "x", Opens: "t"},
		dashygotui.Shortcut[state]{Key: "y", Opens: "u"},
	)
	d := f.driver(60, 20)
	d.Press("x")
	d.RequireContains("event-0100")
	d.Press("up", "up", "up")
	d.RequireContains("event-0097")
	d.RequireAbsent("event-0098")
	total = 110 // ten rows arrive, ten drop off the top; the length is still 50
	d.Tick()
	d.RequireContains("event-0097") // the reader's place did not move
	d.RequireAbsent("event-0110")
	d.Press("end")
	d.RequireContains("event-0110")
}

func TestAppendedNeedsTail(t *testing.T) {
	r := collect.New[state]()
	r.Add(dashygotui.View[state]{ID: "v", Title: "V", Scroll: true, Appended: func(*ctx) int { return 0 },
		Body: func(*ctx, dashygotui.Size) []string { return nil }})
	if err := validate(r); err == nil || !strings.Contains(err.Error(), "Appended but not Tail") {
		t.Errorf("Validate = %v", err)
	}
}

func TestInvalidRecordsAReasonAtTheCallSite(t *testing.T) {
	r := collect.New[state]()
	r.Add(dashygotui.Invalid[state]{Reason: "this setting makes no sense"})
	err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "this setting makes no sense") || !strings.Contains(err.Error(), "Setup > Quick Actions") {
		t.Errorf("Validate = %v", err)
	}
}

func TestLocalStateBelongsToTheDashboardNotTheProcess(t *testing.T) {
	f := newFixture(t)
	f.reg.Add(dashygotui.Shortcut[state]{Key: "x", Do: func(c *ctx) {
		n := dashygotui.Local[int](c, "presses")
		*n++
		c.Flash(fmt.Sprintf("presses=%d", *n))
	}})
	a := f.driver(80, 24)
	a.Press("x", "x", "x")
	a.RequireContains("presses=3")

	b := f.driver(80, 24) // a second dashboard on the same registry, as a test or an embedding makes
	b.Press("x")
	b.RequireContains("presses=1") // it did not inherit a's count
	a.Press("x")
	a.RequireContains("presses=4") // and a did not lose its own
}

func TestLocalKeysAndTypesAreSeparate(t *testing.T) {
	f := newFixture(t)
	var got []string
	f.reg.Add(dashygotui.Shortcut[state]{Key: "x", Do: func(c *ctx) {
		i1 := dashygotui.Local[int](c, "k")
		i2 := dashygotui.Local[int](c, "k")
		other := dashygotui.Local[int](c, "other")
		s := dashygotui.Local[string](c, "k") // same key, another type: another value
		*i1 = 7
		*s = "text"
		got = []string{fmt.Sprint(*i2), fmt.Sprint(*other), *s, fmt.Sprint(i1 == i2)}
	}})
	d := f.driver(80, 24)
	d.Press("x")
	if strings.Join(got, ",") != "7,0,text,true" {
		t.Errorf("got %v: same key and type share a value; a different key or type does not", got)
	}
}

func TestLocalWithBuildsOnceAndKeepsThePointer(t *testing.T) {
	f := newFixture(t)
	builds := 0
	var first, second *[]string
	f.reg.Add(dashygotui.Shortcut[state]{Key: "x", Do: func(c *ctx) {
		p := dashygotui.LocalWith(c, "list", func() *[]string { builds++; return &[]string{"seed"} })
		*p = append(*p, "more")
		if first == nil {
			first = p
		}
		second = p
	}})
	d := f.driver(80, 24)
	d.Press("x", "x")
	if builds != 1 || first != second || len(*second) != 3 {
		t.Errorf("builds %d, same pointer %v, list %v", builds, first == second, *second)
	}
}
