package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Ctrl+L: clear the log buffer, from anywhere, including inside the log view.

func init() {
	dashboard.Registry.Shortcut(dashboard.Shortcut{
		Key: "ctrl+l", Display: "Ctrl+L", Tone: dashygotui.ToneWarn, Label: "Clear logs", Scope: dashygotui.ScopeEverywhere, Prio: 10,
		Run: func(c *dashboard.Ctx) tea.Cmd {
			dashboard.Logs.Clear()
			c.Flash("Logs cleared")
			return nil
		},
	})
}
