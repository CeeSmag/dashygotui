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
// new entries arrive. The f key cycles the minimum level shown.
//
// Divert the application's logger into buf before the dashboard starts:
//
//	restore := buf.Divert(log.Default())
//	defer restore()
//
// The option's badge counts the entries, and the errors among them.
func Logs[S any](reg *dashygotui.Registry[S], buf *logs.Buffer, section string, number int, setup ...Setup) {
	const viewID = "stdaddons.logs"
	viewer := logs.NewViewer(buf) // state lives in this closure, not in a package variable

	reg.View(dashygotui.View[S]{
		ID:    viewID,
		Title: "Logs",
		Mode:  "LOGS",
		Body: func(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
			th := ctx.Theme()
			sub := fmt.Sprintf("%s scroll  Home live  End oldest  f filter: %s",
				th.G.UpDown, strings.ToLower(viewer.MinLevel().String()))
			return panel.Panel(th, "Logs", viewer.Badge(th), sub, size.Cols, size.Rows,
				viewer.Rows(th, size.Cols-4, size.Rows-2))
		},
		Keys: func(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) (tea.Cmd, bool) {
			// The body is the screen minus header and footer, minus the panel's borders.
			return nil, viewer.HandleKey(strings.ToLower(msg.String()), ctx.Size().Rows-4)
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
