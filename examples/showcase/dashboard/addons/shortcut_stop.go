package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// Ctrl+Q: stop everything. A ScopeEverywhere shortcut works on every page, in
// every view and over any pop-up, and may take a force-quit key. ctrl+c still
// uses the library's force-quit, intercepted by Config.OnForceQuit.

func init() {
	dashboard.Registry.Shortcut(dashboard.Shortcut{
		Key: "ctrl+q", Label: "Stop", Scope: dashygotui.ScopeEverywhere, Prio: 25,
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Confirm[dashboard.State]{
				Title:  "Stop and exit?",
				Lines:  []string{"The service is stopped and the dashboard closes."},
				Accept: "Stop",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd {
					return c.Quit("stop", "Service stopped.")
				},
			})
			return nil
		},
	})
}
