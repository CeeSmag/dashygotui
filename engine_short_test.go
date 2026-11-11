package dashygotui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui"
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
		option{Key: "open", Section: "actions", Number: 5, Label: "Open It", Opens: "diag"},
		option{Key: "do", Section: "actions", Number: 6, Label: "Do It", Do: func(*ctx) { f.log = append(f.log, "did") }},
		option{Key: "modal", Section: "actions", Number: 7, Label: "Ask It", Modal: func(*ctx) dashygotui.Modal[state] {
			return &modal.Notice[state]{Title: "Hello Modal", Lines: []string{"from an option"}}
		}},
	)
	d := f.driver(120, 40)
	d.Press("tab")
	d.Type("6")
	d.Press("enter")
	if got := strings.Join(f.log, ","); got != "did" {
		t.Fatalf("Do did not run: %q", got)
	}
	d.Type("7")
	d.Press("enter")
	d.RequireContains("Hello Modal", "from an option")
	d.Press("esc")
	d.Type("5")
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
	r := dashygotui.NewRegistry[state]()
	r.Add(
		option{Key: "none", Section: "actions", Number: 1, Label: "None"},
		option{Key: "two", Section: "actions", Number: 2, Label: "Two", Opens: "v", Do: func(*ctx) {}},
		dashygotui.Shortcut[state]{Key: "q1"},
		dashygotui.Shortcut[state]{Key: "q2", Opens: "v", Do: func(*ctx) {}},
	)
	err := r.Validate()
	if err == nil {
		t.Fatal("want errors")
	}
	for _, want := range []string{`option "none" needs exactly one`, `option "two" needs exactly one`,
		`shortcut "q1" needs exactly one`, `shortcut "q2" needs exactly one`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
	if !strings.Contains(err.Error(), "engine_short_test.go:") {
		t.Errorf("Add must report the caller's file:line:\n%v", err)
	}
}

func TestOpensMustNameARegisteredView(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Add(
		option{Key: "o", Section: "actions", Number: 1, Label: "O", Opens: "nope"},
		dashygotui.Shortcut[state]{Key: "x", Opens: "nada"},
		dashygotui.View[state]{ID: "v", Title: "V", Items: []option{{Label: "Go", Opens: "zip"}}},
	)
	c := baseConfig(r)
	msg := errText(t, c)
	for _, want := range []string{`option "o" opens unknown view "nope"`, `shortcut "x" opens unknown view "nada"`,
		`item "Go" opens unknown view "zip"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
}

func TestAddRejectsNil(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Add(nil)
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "nil addon") {
		t.Errorf("Validate = %v", err)
	}
}

func TestViewEntryRegistersTheOption(t *testing.T) {
	f := newFixture(t)
	v := plainView(f.reg, "diag")
	v.Entry = &dashygotui.Entry[state]{Section: "actions", Number: 16, Hotkey: "d", HotkeyHint: "Diagnostics",
		HotkeyDisplay: "D", Note: "health and logs"}
	f.reg.Add(v)
	d := f.driver(140, 40)
	d.RequireContains("[D] Diagnostics") // the footer hint, without a separate Shortcut
	d.Press("tab")
	d.RequireContains("DIAG Title (D)", "[health and logs]") // label defaults to the view title; Note is shown
	d.Press("d")
	d.RequireContains("this is diag")
}

func TestViewEntryNumberCollisionIsReported(t *testing.T) {
	f := newFixture(t)
	v := plainView(f.reg, "diag")
	v.Entry = &dashygotui.Entry[state]{Section: "actions", Number: 1} // [1] is Run Backup
	f.reg.Add(v)
	_, err := dashygotui.New(f.cfg)
	if err == nil || !strings.Contains(err.Error(), "reuses number 1") {
		t.Fatalf("New = %v, want a number collision", err)
	}
}

func TestViewItemsAreNumberedByPosition(t *testing.T) {
	f := newFixture(t)
	noop := func(*ctx) {}
	f.reg.Add(dashygotui.View[state]{
		ID: "tools", Title: "Tools", Items: []option{
			{Label: "First Tool", Do: func(*ctx) { f.log = append(f.log, "first") }},
			{Label: "Second Tool", Do: noop, Note: "muted note"},
			{Number: 9, Label: "Ninth Tool", Do: noop},
		},
		Entry: &dashygotui.Entry[state]{Section: "actions", Number: 30, Hotkey: "t"},
	})
	d := f.driver(120, 30)
	d.Press("t")
	d.RequireContains("TOOLS") // Mode defaults to the title in capitals
	d.RequireAbsent("VIEW")
	d.RequireContains("[1]", "First Tool", "[2]", "Second Tool", "[muted note]", "[9]", "Ninth Tool")
	d.Type("1")
	d.Press("enter")
	if got := strings.Join(f.log, ","); got != "first" {
		t.Errorf("log = %q", got)
	}
}

func TestViewItemsAndOptionsAreExclusive(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Add(dashygotui.View[state]{ID: "v", Title: "V",
		Items:   []option{{Label: "A", Do: func(*ctx) {}}},
		Options: func(*ctx) []option { return nil }})
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "both Options and Items") {
		t.Errorf("Validate = %v", err)
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
	r := dashygotui.NewRegistry[state]()
	r.Add(dashygotui.View[state]{ID: "v", Title: "V", Follow: true, Status: func(*ctx, dashygotui.Size) []string { return nil }})
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "Follow or Framed") {
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

func TestHotkeyHintOnlyWhenAsked(t *testing.T) {
	f := newFixture(t)
	f.reg.Add(
		option{Key: "a", Section: "actions", Number: 40, Label: "A", Hotkey: "u", HotkeyHint: "Upload", Do: func(*ctx) {}},
		option{Key: "b", Section: "actions", Number: 41, Label: "B", Hotkey: "i", Do: func(*ctx) {}},
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
	r := dashygotui.NewRegistry[state]()
	r.Add(dashygotui.View[state]{ID: "v", Title: "V", Scroll: true, Appended: func(*ctx) int { return 0 },
		Body: func(*ctx, dashygotui.Size) []string { return nil }})
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "Appended but not Tail") {
		t.Errorf("Validate = %v", err)
	}
}

func TestViewItemNumbersMustBeUnique(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Add(dashygotui.View[state]{ID: "v", Title: "V", Items: []option{
		{Label: "A", Do: func(*ctx) {}},            // numbered 1 by position
		{Number: 1, Label: "B", Do: func(*ctx) {}}, // explicit 1: clash
		{Number: 7, Label: "C", Do: func(*ctx) {}},
		{Label: "D", Do: func(*ctx) {}}, // numbered 4 by position, fine
	}})
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), `item "B" has number 1, which "A" already has`) {
		t.Fatalf("Validate = %v", err)
	}
	if strings.Count(err.Error(), "already has") != 1 {
		t.Errorf("only the real clash should be reported: %v", err)
	}
}

func TestOptionKeyMayNotUseTheLibrariesPrefix(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Add(option{Key: dashygotui.ReservedKeyPrefix + "palette", Section: "actions", Number: 90, Label: "X", Do: func(*ctx) {}})
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "reserved for the library") {
		t.Errorf("Validate = %v", err)
	}
}

func TestInvalidRecordsAReasonAtTheCallSite(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Add(dashygotui.Invalid[state]{Reason: "this setting makes no sense"})
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), "this setting makes no sense") || !strings.Contains(err.Error(), "engine_short_test.go:") {
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
