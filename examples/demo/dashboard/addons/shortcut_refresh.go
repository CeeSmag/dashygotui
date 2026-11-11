package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
)

// Ctrl+R: fetch fresh data now, from anywhere.

func init() {
	dashboard.Registry.Shortcut(dashygotui.Shortcut[dashboard.State]{
		Key:   "ctrl+r",
		Label: "Refresh",
		Scope: dashygotui.ScopeEverywhere,
		Prio:  20,
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.Flash("Refreshing...")
			return c.Refresh()
		},
	})
}
