package stdaddons

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/panel"
)

// LogView is a full-screen log view of Buffer and the row that opens it. Place
// it in a panel's Items in the map:
//
//	stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs}
//
// The view shows the newest entry on top and scrolls with the arrow keys (or j
// and k), PgUp and PgDn; Home returns to the live top and End jumps to the
// oldest entry. Scrolled away from the top, it keeps the line being read while
// new entries arrive. The f key cycles the minimum level shown. Entries filed
// under a category with [logs.Buffer.In] get a strip of tabs along the top,
// "All" first; CategoryKey moves along them, each tab keeps its own scroll
// position, and a tab you are not on shows how many entries arrived since you
// left it.
//
// Divert the application's logger into Buffer before the dashboard starts:
//
//	restore := buf.Divert(log.Default())
//	defer restore()
//
// The option's badge counts the entries, and the errors among them.
type LogView[S any] struct {
	// Buffer is the log buffer shown. Required.
	Buffer *logs.Buffer
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
		Label: "Logs",
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
