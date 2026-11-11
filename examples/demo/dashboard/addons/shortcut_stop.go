package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// Ctrl+Q: stop everything. It works on every page, in every view and over any
// pop-up, and replaces the library's default quit with a confirmation.

func init() {
	dashboard.Registry.Add(dashygotui.Shortcut[dashboard.State]{
		Key:   "ctrl+q",
		Label: "Stop",
		Scope: dashygotui.ScopeEverywhere,
		Prio:  25,
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Confirm[dashboard.State]{
				Title:  "Stop and exit?",
				Lines:  []string{"The demo service is stopped and the dashboard closes."},
				Accept: "Stop",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd {
					return c.Quit("stop", "Service stopped.")
				},
			}
		},
	})
}
