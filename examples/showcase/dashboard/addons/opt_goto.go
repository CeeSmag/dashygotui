package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [60] [61] [62] Context.GotoPage, and Option.Order: the numbers rise but the
// rows are listed in reverse, because Order sorts before Number. Typing a number
// still finds the right row.

func init() {
	jump := func(number, order int, key, label, page, title string) {
		dashboard.Registry.Option(dashboard.Option{
			Key: key, Section: "navigation", Number: number, Order: order, Label: label,
			Badge: func(c *dashboard.Ctx) string { return c.Badge(title) },
			Run: func(c *dashboard.Ctx) tea.Cmd {
				if !c.GotoPage(page) {
					c.FlashError("no such page: " + page)
					return nil
				}
				c.Flash("Now on " + title)
				return nil
			},
		})
	}
	jump(60, 3, "goto_charts", "Jump to Charts", "charts", "Charts")
	jump(61, 2, "goto_theme", "Jump to Theme", "theme", "Theme")
	jump(62, 1, "goto_monitor", "Jump to Monitor", "monitor", "Monitor")
}
