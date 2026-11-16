// Package collect gathers the features of a test's dashboard by where they go,
// for the library's own tests. A test adds rows to named panels, and views,
// shortcuts and hooks, and places the List in its map with Rows or All. The
// List is read when dashygotui.New runs, so a test can build its configuration
// first and add to it afterwards.
package collect

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
)

// List is a growing set of rows, by panel, and of the views, shortcuts and
// hooks that come with them.
type List[S any] struct {
	rows   map[string][]dashygotui.Addon[S]
	extras []dashygotui.Addon[S] // views, shortcuts and hooks: placed without a row
}

// New returns an empty list.
func New[S any]() *List[S] { return &List[S]{rows: map[string][]dashygotui.Addon[S]{}} }

// Row adds rows to the named panel, in order.
func (l *List[S]) Row(panel string, items ...dashygotui.Addon[S]) {
	l.rows[panel] = append(l.rows[panel], items...)
}

// Add adds views, shortcuts and hooks, which are placed without a row.
func (l *List[S]) Add(items ...dashygotui.Addon[S]) { l.extras = append(l.extras, items...) }

// View adds a view with no row of its own.
func (l *List[S]) View(v dashygotui.View[S]) { l.Add(v) }

// Shortcut adds a shortcut.
func (l *List[S]) Shortcut(s dashygotui.Shortcut[S]) { l.Add(s) }

// OnTick adds a hook with a Tick function.
func (l *List[S]) OnTick(fn func(*dashygotui.Context[S], time.Time) tea.Cmd) {
	l.Add(dashygotui.Hook[S]{Tick: fn})
}

// OnMsg adds a hook with a Msg function.
func (l *List[S]) OnMsg(fn func(*dashygotui.Context[S], tea.Msg) tea.Cmd) {
	l.Add(dashygotui.Hook[S]{Msg: fn})
}

// OnResult adds a hook with a Result function.
func (l *List[S]) OnResult(fn func(*dashygotui.Context[S], dashygotui.TaskResult) (bool, tea.Cmd)) {
	l.Add(dashygotui.Hook[S]{Result: fn})
}

// Rows is the item that expands, when the dashboard is built, into the rows
// added to panel: place it in that panel's Items.
func (l *List[S]) Rows(panel string) dashygotui.Addon[S] { return placed[S]{l: l, panel: panel} }

// All is Rows with the views, shortcuts and hooks added as well. Place it in
// one panel of the map.
func (l *List[S]) All(panel string) dashygotui.Addon[S] {
	return placed[S]{l: l, panel: panel, extras: true}
}

// placed is a List's rows of one panel, read when the dashboard is built.
type placed[S any] struct {
	l      *List[S]
	panel  string
	extras bool
}

// Addons implements dashygotui.Addon.
func (p placed[S]) Addons() []dashygotui.Addon[S] {
	out := append([]dashygotui.Addon[S](nil), p.l.rows[p.panel]...)
	if p.extras {
		out = append(out, p.l.extras...)
	}
	return out
}

// From returns the List whose rows c's map places, for a test that adds to a
// configuration a helper built, or nil when there is none (so a test that
// expects one fails at once instead of adding to a list nobody reads).
func From[S any](c dashygotui.Config[S]) *List[S] {
	var walk func(pages []dashygotui.Page[S]) *List[S]
	walk = func(pages []dashygotui.Page[S]) *List[S] {
		for _, pg := range pages {
			for _, pn := range pg.Panels {
				for _, it := range pn.Items {
					switch v := it.(type) {
					case placed[S]:
						return v.l
					case dashygotui.Page[S]:
						if l := walk([]dashygotui.Page[S]{v}); l != nil {
							return l
						}
					}
				}
			}
		}
		return nil
	}
	return walk(c.Map.Pages)
}

// Errors builds a dashboard of l alone, its rows of every panel in one panel of
// one page, and returns the error: every mistake in l.
func Errors[S any](l *List[S]) error {
	var items []dashygotui.Addon[S]
	for name := range l.rows {
		items = append(items, l.Rows(name))
	}
	items = append(items, placed[S]{l: l, extras: true})
	_, err := dashygotui.New(dashygotui.Config[S]{Map: dashygotui.Map[S]{Pages: []dashygotui.Page[S]{
		{Title: "Collect", Panels: []dashygotui.Panel[S]{{Title: "Rows", Items: items}}},
	}}})
	return err
}
