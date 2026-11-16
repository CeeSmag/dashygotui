package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [60] [61] [62] Context.GotoPage, and Option.Order: the numbers rise but the
// rows are listed in reverse, because Order sorts before Number. Typing a number
// still finds the right row.

// jump is a composite addon: one value per row, each its own Option. A type of
// the file's own that returns addons is how a file declares several alike.
type jump struct {
	number, order           int
	key, label, page, title string
}

// Addons implements dashygotui.Addon.
func (j jump) Addons() []dashboard.Addon {
	return []dashboard.Addon{dashboard.Option{
		Key: j.key, Section: "navigation", Number: j.number, Order: j.order, Label: j.label,
		Badge: func(c *dashboard.Ctx) string { return c.Badge(j.title) },
		Do: func(c *dashboard.Ctx) {
			if !c.GotoPage(j.page) {
				c.FlashError("no such page: " + j.page)
				return
			}
			c.Flash("Now on " + j.title)
		},
	}}
}

var gotoChartsOption = jump{number: 60, order: 3, key: "goto_charts", label: "Jump to Charts", page: "charts", title: "Charts"}

var gotoThemeOption = jump{number: 61, order: 2, key: "goto_theme", label: "Jump to Theme", page: "theme", title: "Theme"}

var gotoMonitorOption = jump{number: 62, order: 1, key: "goto_monitor", label: "Jump to Monitor", page: "monitor", title: "Monitor"}
