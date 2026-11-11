package dashtest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/text"
)

// AuditOption tunes [Audit].
type AuditOption func(*auditSettings)

type auditSettings struct{ walk bool }

// WalkOptions makes [Audit] also open every option of every page and check what
// it shows. That runs the application's real actions, so use it with a fake
// poll function and an in-memory store, and not with options that start real
// processes.
func WalkOptions() AuditOption { return func(s *auditSettings) { s.walk = true } }

// auditSizes are the terminal sizes an audit draws everything at, from absurdly
// small to large.
func auditSizes() [][2]int {
	return [][2]int{{1, 1}, {5, 3}, {20, 6}, {39, 9}, {40, 10}, {41, 11}, {60, 15}, {80, 24}, {120, 40}, {200, 60}}
}

// ttySafe reports whether r may be drawn on the Linux console.
func ttySafe(r rune) bool {
	return (r >= 0x20 && r <= 0x7e) || strings.ContainsRune("─│┌┐└┘├┤░▒▓█\n", r)
}

// Audit checks an application's dashboard against the library's rendering
// rules, with no application code of its own to write: put one call in the
// application's tests.
//
// It builds the dashboard (a configuration or registration error fails the
// test), then draws every page and every view at a sweep of terminal sizes from
// 1x1 to 200x60 and requires each frame to be exactly the terminal's size, with
// no panic. It then repeats the pages and views in TTY mode and requires that
// only characters the Linux console can show were drawn. With [WalkOptions] it
// also opens every option of every page and checks that the result obeys the
// same rules.
//
// The audit never touches the user's settings if cfg.Store is nil (it is in
// memory then), and cfg.Poll should be a fake.
func Audit[S any](tb testing.TB, cfg dashygotui.Config[S], opts ...AuditOption) {
	tb.Helper()
	var set auditSettings
	for _, o := range opts {
		o(&set)
	}
	dash, err := dashygotui.New(cfg)
	if err != nil {
		tb.Fatalf("dashtest.Audit: building the dashboard: %v", err)
	}
	out := dash.Outline()

	for _, tty := range []bool{false, true} {
		c := cfg
		if tty {
			c.Glyphs = dashygotui.GlyphsTTY
		}
		mode := map[bool]string{false: "unicode", true: "tty"}[tty]

		for i, id := range out.Pages {
			what := "page " + id + " (" + mode + ")"
			try(tb, what, func() {
				d := New(tb, c, 120, 40)
				fill(d)
				d.Press(strings.Split(strings.Repeat("tab,", i), ",")[:i]...)
				audit(tb, d, what, tty)
			})
		}
		for _, id := range out.Views {
			what := "view " + id + " (" + mode + ")"
			try(tb, what, func() {
				vc := c
				vc.StartView = []string{id}
				d := New(tb, vc, 120, 40)
				fill(d)
				audit(tb, d, what, tty)
			})
		}
		if set.walk {
			for _, o := range out.Options {
				what := fmt.Sprintf("option [%d] %s (%s)", o.Number, o.Label, mode)
				try(tb, what, func() {
					d := New(tb, c, 120, 40)
					fill(d)
					d.Type(fmt.Sprint(o.Number))
					d.Press("enter")
					audit(tb, d, what, tty)
				})
			}
		}
	}
}

// try runs one audited scenario and reports a panic in it, instead of letting
// it end the test with no hint of where.
func try(tb testing.TB, what string, f func()) {
	tb.Helper()
	defer func() {
		if r := recover(); r != nil {
			tb.Errorf("audit: %s panicked: %v", what, r)
		}
	}()
	f()
}

// fill lets the graphs and logs gather a few samples.
func fill(d *Driver) {
	for range 4 {
		d.Advance(time.Second)
	}
}

// audit draws what is on screen at every size and reports every broken rule. A
// panic ends the scenario; try reports it.
func audit(tb testing.TB, d *Driver, what string, tty bool) {
	tb.Helper()
	for _, sz := range auditSizes() {
		cols, rows := sz[0], sz[1]
		d.Resize(cols, rows)
		lines := d.Lines()
		if len(lines) != rows {
			tb.Errorf("audit: %s at %dx%d drew %d rows, want %d", what, cols, rows, len(lines), rows)
			continue
		}
		for i, l := range lines {
			if w := text.Width(l); w != cols {
				tb.Errorf("audit: %s at %dx%d: row %d is %d cells wide, want %d", what, cols, rows, i+1, w, cols)
				break
			}
		}
	}
	d.Resize(120, 40)
	if !tty {
		return
	}
	seen := map[rune]bool{}
	for _, r := range d.Screen() {
		if !ttySafe(r) && !seen[r] {
			seen[r] = true
			tb.Errorf("audit: %s drew %q (U+%04X) in TTY mode; use the theme's glyphs (Context.Theme().G) and plain ASCII", what, r, r)
		}
	}
}
