package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [90] Reset All Settings: a type-the-word confirmation that ends the program
// with a reason, so main can do the destructive part once the terminal is back.
// Order sorts it last whatever its number.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "reset", Section: "danger", Number: 90, Order: 1000, Label: "Reset All Settings",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Typed[dashboard.State]{
				Title:  "Reset all settings?",
				Lines:  []string{"Every saved setting is forgotten and the showcase closes."},
				Word:   "RESET",
				Prompt: "Type RESET to continue",
				OnConfirm: func(c *dashboard.Ctx) tea.Cmd {
					return c.Quit("reset", "All settings were reset.")
				},
			})
			return nil
		},
	})
}
