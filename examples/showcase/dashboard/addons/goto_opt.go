package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Context.GotoPage: three rows that move to another page by its ID.

// jump is a composite addon: one value per row, each its own Option. A type of
// the file's own that returns addons is how a file declares several alike.
type jump struct {
	label, page, title string
}

// Addons implements dashygotui.Addon.
func (j jump) Addons() []dashboard.Addon {
	return []dashboard.Addon{dashboard.Option{
		Label: j.label,
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

var gotoChartsOption = jump{label: "Jump to Charts", page: "charts", title: "Charts"}

var gotoThemeOption = jump{label: "Jump to Theme", page: "theme", title: "Theme"}

var gotoMonitorOption = jump{label: "Jump to Monitor", page: "monitor", title: "Monitor"}
