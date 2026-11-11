package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// ?: key help. A shortcut with the default scope (ScopePages) works on the
// dashboard pages only, not inside views or over pop-ups.

func init() {
	dashboard.Registry.Add(dashboard.Shortcut{
		Key: "?", Label: "Help", Prio: 45,
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Notice[dashboard.State]{
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
					"Ctrl+O                open the web UI (Actions page)",
					"Ctrl+L                clear the logs, from anywhere",
				},
			}
		},
	})
}
