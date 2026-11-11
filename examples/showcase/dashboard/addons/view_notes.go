package addons

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [53] Release Notes (hotkey n): a view whose Body is longer than the screen.
// Scroll: true makes the dashboard do the scrolling (Up/Down, PgUp/PgDn, Home,
// End) and draw the scroll bar, so the view writes none of that. OnEnter and
// OnLeave run when the view appears and disappears; here they count visits and
// log to the "ui" category. The visit count is per-dashboard state (dashygotui.Local), not package state.

func init() {
	dashboard.Registry.Add(dashboard.View{
		ID: "notes", Title: "Release Notes", Mode: "NOTES", Scroll: true,
		Entry: &dashboard.Entry{Section: "views", Number: 53, Hotkey: "n"},
		Badge: func(c *dashboard.Ctx) string { return c.Count(*dashygotui.Local[int](c, "notes.visits"), "visits") },
		Body: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			var rows []string
			for v := 40; v >= 1; v-- {
				rows = append(rows, th.Title.Render(fmt.Sprintf("Version 0.%d.0", v)))
				rows = append(rows,
					fmt.Sprintf("  %s added feature number %d", th.G.Bullet, v),
					fmt.Sprintf("  %s fixed bug number %d", th.G.Bullet, v*3),
					"")
			}
			return rows
		},
		OnEnter: func(c *dashboard.Ctx) tea.Cmd {
			visits := dashygotui.Local[int](c, "notes.visits")
			*visits++
			dashboard.UILog.Debug("release notes shown", "visit", *visits)
			return nil
		},
		OnLeave: func(*dashboard.Ctx) tea.Cmd {
			dashboard.UILog.Debug("release notes hidden")
			return nil
		},
	})
}
