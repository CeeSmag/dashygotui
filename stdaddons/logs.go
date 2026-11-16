package stdaddons

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/panel"
)

// Logs registers a full-screen log view of buf, and an option that opens it.
//
// The view shows the newest entry on top and scrolls with the arrow keys (or j
// and k), PgUp and PgDn; Home returns to the live top and End jumps to the
// oldest entry. Scrolled away from the top, it keeps the line being read while
// new entries arrive. The f key cycles the minimum level shown. Use [LogsWith]
// for category tabs.
//
// Divert the application's logger into buf before the dashboard starts:
//
//	restore := buf.Divert(log.Default())
//	defer restore()
//
// The option's badge counts the entries, and the errors among them.
func Logs[S any](reg *dashygotui.Registry[S], buf *logs.Buffer, section string, number int, setup ...Setup) {
	LogsWith(reg, buf, section, number, LogOptions{}, setup...)
}

// LogOptions tunes [LogsWith].
type LogOptions struct {
	// Categories are tabs always shown, in this order, ahead of any category
	// the buffer meets later. Optional.
	Categories []string
	// CategoryKey is the key that selects the next category tab; the same letter
	// in upper case selects the previous one. It overrides the scrolling keys.
	// Without it the tabs cannot be switched.
	CategoryKey string
}

// LogsWith is [Logs] with category tabs. Entries are filed under a category with
// [logs.Buffer.In]; the view shows a strip of tabs along the top, "All" first,
// with the selected one in brackets. CategoryKey moves along them, wrapping
// round. Each category keeps its own scroll position, and a tab you are not on
// shows how many entries arrived since you left it.
func LogsWith[S any](reg *dashygotui.Registry[S], buf *logs.Buffer, section string, number int, opts LogOptions, setup ...Setup) {
	v := LogView[S]{Buffer: buf, Section: section, Number: number, Categories: opts.Categories, CategoryKey: opts.CategoryKey}
	if len(setup) > 0 {
		v.Hotkey, v.Order = setup[0].Hotkey, setup[0].Order
	}
	reg.Add(v)
}

// LogView is a full-screen log view of Buffer and the option that opens it: the
// value form of [Logs] and [LogsWith]. Declare it as an addon:
//
//	var logView = stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs, Section: "views", Number: 52}
//
// Divert the application's logger into Buffer before the dashboard starts. The
// option's badge counts the entries, and the errors among them.
type LogView[S any] struct {
	// Buffer is the log buffer shown. Required.
	Buffer *logs.Buffer
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Hotkey runs the option from any page, like [dashygotui.Option].Hotkey. Optional.
	Hotkey string
	// Order sorts the option within its section. Optional.
	Order int
	// Categories are tabs always shown, in this order, ahead of any category
	// the buffer meets later. Optional.
	Categories []string
	// CategoryKey is the key that selects the next category tab; the same letter
	// in upper case selects the previous one. Without it the tabs cannot be
	// switched. Optional.
	CategoryKey string
}

// Addons implements [dashygotui.Addon].
func (a LogView[S]) Addons() []dashygotui.Addon[S] {
	const viewID = "stdaddons.logs"
	if a.Buffer == nil {
		return []dashygotui.Addon[S]{dashygotui.Invalid[S]{Reason: "log view has no Buffer"}}
	}
	buf := a.Buffer
	// The viewer holds the reader's place (tab, filter, scroll position), which
	// belongs to one dashboard, not to the process.
	viewerOf := func(ctx *dashygotui.Context[S]) *logs.Viewer {
		return dashygotui.LocalWith(ctx, viewID, func() *logs.Viewer {
			return logs.NewViewer(buf, logs.WithCategories(a.Categories...), logs.WithCategoryKey(a.CategoryKey))
		})
	}
	view := dashygotui.View[S]{
		ID:    viewID,
		Title: "Logs",
		Mode:  "LOGS",
		Body: func(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
			th, viewer := ctx.Theme(), viewerOf(ctx)
			sub := fmt.Sprintf("%s scroll  Home live  End oldest  f filter: %s",
				th.G.UpDown, strings.ToLower(viewer.MinLevel().String()))
			if a.CategoryKey != "" {
				sub += "  " + a.CategoryKey + " category"
			}
			return panel.Panel(th, "Logs", viewer.Badge(th), sub, size.Cols, size.Rows,
				viewer.Rows(th, size.Cols-4, size.Rows-2))
		},
		Keys: func(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) (tea.Cmd, bool) {
			// The body is the screen minus header and footer, minus the panel's borders.
			return nil, viewerOf(ctx).HandleKey(msg.String(), ctx.Size().Rows-4)
		},
	}
	option := dashygotui.Option[S]{
		Key:     viewID,
		Section: a.Section,
		Number:  a.Number,
		Order:   a.Order,
		Hotkey:  a.Hotkey,
		Label:   "Logs",
		Badge: func(ctx *dashygotui.Context[S]) string {
			badge := ctx.Count(buf.Len(), "entries")
			if n := buf.Count(logs.Error); n > 0 {
				badge += " " + ctx.Theme().Red.Render(fmt.Sprintf("%d errors", n))
			}
			return badge
		},
		Opens: viewID,
	}
	return []dashygotui.Addon[S]{view, option}
}
