package addons

import (
	"log"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Diagnostics (hotkey d): a sub-page, a Page placed as a row of a panel in the
// map. It has panels of its own and its own numbering from 1; its rows' hotkeys
// (t) work only inside it; Back returns to the page it was opened from. This
// file holds what the sub-page does: the badge beside its title in the header,
// and a Session that runs while it is on screen. The map says what is on it.

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
