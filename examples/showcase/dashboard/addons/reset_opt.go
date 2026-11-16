package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// Reset All Settings: a type-the-word confirmation that ends the program
// with a reason, so main can do the destructive part once the terminal is back.
// Order sorts it last whatever its number.

var resetOption = dashboard.Option{
	Label: "Reset All Settings",
	Modal: func(c *dashboard.Ctx) dashboard.Modal {
		return &modal.Typed[dashboard.State]{
			Title:  "Reset all settings?",
			Lines:  []string{"Every saved setting is forgotten and the showcase closes."},
			Word:   "RESET",
			Prompt: "Type RESET to continue",
			OnConfirm: func(c *dashboard.Ctx) tea.Cmd {
				return c.Quit("reset", "All settings were reset.")
			},
		}
	},
}
