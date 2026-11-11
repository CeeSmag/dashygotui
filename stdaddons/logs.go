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
	const viewID = "stdaddons.logs"
	// The viewer holds the reader's place (tab, filter, scroll position), which
	// belongs to one dashboard, not to the process.
	viewerOf := func(ctx *dashygotui.Context[S]) *logs.Viewer {
		return dashygotui.LocalWith(ctx, viewID, func() *logs.Viewer {
			return logs.NewViewer(buf, logs.WithCategories(opts.Categories...), logs.WithCategoryKey(opts.CategoryKey))
		})
	}

	reg.View(dashygotui.View[S]{
		ID:    viewID,
		Title: "Logs",
		Mode:  "LOGS",
		Body: func(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
			th, viewer := ctx.Theme(), viewerOf(ctx)
			sub := fmt.Sprintf("%s scroll  Home live  End oldest  f filter: %s",
				th.G.UpDown, strings.ToLower(viewer.MinLevel().String()))
			if opts.CategoryKey != "" {
				sub += "  " + opts.CategoryKey + " category"
			}
			return panel.Panel(th, "Logs", viewer.Badge(th), sub, size.Cols, size.Rows,
				viewer.Rows(th, size.Cols-4, size.Rows-2))
		},
		Keys: func(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) (tea.Cmd, bool) {
			// The body is the screen minus header and footer, minus the panel's borders.
			return nil, viewerOf(ctx).HandleKey(msg.String(), ctx.Size().Rows-4)
		},
	})

	reg.Option(apply(dashygotui.Option[S]{
		Key:     viewID,
		Section: section,
		Number:  number,
		Label:   "Logs",
		Badge: func(ctx *dashygotui.Context[S]) string {
			badge := ctx.Count(buf.Len(), "entries")
			if n := buf.Count(logs.Error); n > 0 {
				badge += " " + ctx.Theme().Red.Render(fmt.Sprintf("%d errors", n))
			}
			return badge
		},
		Run: func(ctx *dashygotui.Context[S]) tea.Cmd {
			if err := ctx.OpenView(viewID); err != nil {
				ctx.FlashError(err.Error())
			}
			return nil
		},
	}, setup))
}
