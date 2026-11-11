package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [90] Reset All Settings: a type-the-word confirmation that ends the program
// with a reason, so main can do the destructive part once the terminal is back.

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "reset", Section: "settings", Number: 90, Order: 1000, Label: "Reset All Settings",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Typed[dashboard.State]{
				Title:  "Reset all settings?",
				Lines:  []string{"Every saved setting is forgotten and the demo closes."},
				Word:   "RESET",
				Prompt: "Type RESET to continue",
				OnConfirm: func(c *dashboard.Ctx) tea.Cmd {
					return c.Quit("reset", "All settings were reset.")
				},
			}
		},
	})
}
