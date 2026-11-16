package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Open Web UI. The map gives it a key combination as its hotkey (Ctrl+O), written
// and coloured there.

var webOption = dashboard.Option{
	Label: "Open Web UI",
	Do: func(c *dashboard.Ctx) {
		c.Flash("Would open http://localhost:8080 in your browser")
	},
}
