package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// Ctrl+Q: stop everything. A ScopeEverywhere shortcut works on every page, in
// every view and over any pop-up, and may take a force-quit key. ctrl+c still
// uses the library's force-quit, intercepted by Config.OnForceQuit. Display
// writes the key as "Ctrl+Q" in the footer and Tone draws the hint in the
// theme's danger colour.

func init() {
	dashboard.Registry.Add(dashboard.Shortcut{
		Key: "ctrl+q", Display: "Ctrl+Q", Tone: dashygotui.ToneDanger, Label: "Stop & Exit", Scope: dashygotui.ScopeEverywhere, Prio: 25,
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Confirm[dashboard.State]{
				Title:  "Stop and exit?",
				Lines:  []string{"The service is stopped and the dashboard closes."},
				Accept: "Stop",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd {
					return c.Quit("stop", "Service stopped.")
				},
			}
		},
	})
}
