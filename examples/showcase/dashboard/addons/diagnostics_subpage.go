package addons

import (
	"log"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Diagnostics: a subpage, a window opened from a row. This file is the subpage
// itself: the option that opens it (only its label; its hotkey and what is inside
// are the map's, diagnostics_map.go), the badge beside its title in the header,
// and a Session that runs while it is on screen. What it holds is named after it:
// diagnostics_<name>_sub<kind>.go.

var diagnosticsSubpage = dashboard.Option{Label: "Diagnostics"}

func diagnosticsBadge(c *dashboard.Ctx) string {
	th := c.Theme()
	if c.Err() != nil {
		return th.Red.Render(th.G.Cross + " UNREACHABLE")
	}
	return th.Green.Render(th.G.Bullet + " HEALTHY")
}

func diagnosticsSession(*dashboard.Ctx) func() {
	log.Print("[INFO] diagnostics opened")
	return func() { log.Print("[INFO] diagnostics closed") }
}
