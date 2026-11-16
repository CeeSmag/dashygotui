package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Ctrl+L: clear the log buffer, from anywhere, including inside the log view.

var clearlogsShortcut = dashboard.Shortcut{
	Key: "ctrl+l", Display: "Ctrl+L", Tone: dashygotui.ToneWarn, Label: "Clear logs", Scope: dashygotui.ScopeEverywhere, Prio: 10,
	Do: func(c *dashboard.Ctx) {
		c.State().ClearLogs()
		c.Flash("Logs cleared")
	},
}
