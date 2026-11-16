// Package collect gathers addons one call at a time, for the library's own
// tests. A List records where each call was made, with dashygotui.At, and is
// itself one addon, so a test builds its dashboard with
// Config{Addons: []dashygotui.Addon[S]{list}} and adds to the list afterwards:
// the addons are read when dashygotui.New runs.
package collect

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
)

// List is a growing list of addons. The zero value is ready to use.
type List[S any] struct {
	items []dashygotui.Addon[S]
}

// New returns an empty list.
func New[S any]() *List[S] { return &List[S]{} }

// Addons implements dashygotui.Addon.
func (l *List[S]) Addons() []dashygotui.Addon[S] {
	return append([]dashygotui.Addon[S](nil), l.items...)
}

// site is "file.go:line" of the test code that called a List method.
func site() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

func (l *List[S]) add(at string, items ...dashygotui.Addon[S]) {
	for _, it := range items {
		l.items = append(l.items, dashygotui.At(at, it))
	}
}

// Add appends items, all recorded at the caller's line.
func (l *List[S]) Add(items ...dashygotui.Addon[S]) { l.add(site(), items...) }

// Option appends an option.
func (l *List[S]) Option(o dashygotui.Option[S]) { l.add(site(), o) }

// Widget appends a widget.
func (l *List[S]) Widget(w dashygotui.Widget[S]) { l.add(site(), w) }

// View appends a view.
func (l *List[S]) View(v dashygotui.View[S]) { l.add(site(), v) }

// Shortcut appends a shortcut.
func (l *List[S]) Shortcut(s dashygotui.Shortcut[S]) { l.add(site(), s) }

// OnTick appends a hook with a Tick function.
func (l *List[S]) OnTick(fn func(*dashygotui.Context[S], time.Time) tea.Cmd) {
	l.add(site(), dashygotui.Hook[S]{Tick: fn})
}

// OnMsg appends a hook with a Msg function.
func (l *List[S]) OnMsg(fn func(*dashygotui.Context[S], tea.Msg) tea.Cmd) {
	l.add(site(), dashygotui.Hook[S]{Msg: fn})
}

// OnResult appends a hook with a Result function.
func (l *List[S]) OnResult(fn func(*dashygotui.Context[S], dashygotui.TaskResult) (bool, tea.Cmd)) {
	l.add(site(), dashygotui.Hook[S]{Result: fn})
}

// From returns the List among c's addons, for a test that adds to a
// configuration a helper built, or nil when c has none (so a test that expects
// one fails at once instead of adding to a list nobody reads).
func From[S any](c dashygotui.Config[S]) *List[S] {
	for _, a := range c.Addons {
		if l, ok := a.(*List[S]); ok {
			return l
		}
	}
	return nil
}

// Errors builds a dashboard from l on a single page with no sections and
// returns the error: every mistake in l's addons, together with an "unknown
// section" error for each option, which a test of something else ignores.
func Errors[S any](l *List[S]) error {
	_, err := dashygotui.New(dashygotui.Config[S]{
		Pages:  []dashygotui.Page[S]{{ID: "collect", Title: "Collect"}},
		Addons: []dashygotui.Addon[S]{l},
	})
	return err
}
