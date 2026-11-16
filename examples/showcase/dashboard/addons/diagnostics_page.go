package addons

import (
	"log"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// What the Diagnostics window does while it is open: the badge beside its title
// in the header, and a Session that runs while it is on screen. The window is a
// page placed as a row (diagnostics_map.go): it has panels of its own and its own
// numbering from 1, its rows' hotkeys (t) work only inside it, and Back returns
// to the page it was opened from. The row that opens it is diagnostics_opt.go.

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
