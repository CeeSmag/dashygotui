package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// ?: key help. A shortcut with the default scope (ScopePages) works on the
// dashboard pages only, not inside views or over pop-ups.

func init() {
	dashboard.Registry.Shortcut(dashboard.Shortcut{
		Key: "?", Label: "Help", Prio: 45,
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Notice[dashboard.State]{
				Title: "Keys", Badge: c.Badge("pages"),
				Lines: []string{
					"Up / Down      move in a panel",
					"Left / Right   change panel",
					"Tab            next page (Shift+Tab back)",
					"1..99 + Enter  jump to a numbered option, on any page",
					"Enter / Space  open the highlighted option",
					"",
					"s  storage limit       d  diagnostics",
					"p  processes           g  logs",
					"m  colour theme",
					"",
					"Esc / b / Backspace   back, or close a pop-up",
					"+ / -                 refresh faster / slower",
					"q                     quit",
					"Ctrl+C                quit (asks first)",
					"Ctrl+Q                stop the service, from anywhere",
					"Ctrl+R                refresh now, from anywhere",
					"Ctrl+L                clear the logs, from anywhere",
				},
			})
			return nil
		},
	})
}
