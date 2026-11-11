package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [33] [34] [35] modal.Confirm (yes/no), Confirm used as an acknowledgement
// ("Back"), and modal.Typed with an OnEmpty escape hatch.

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "confirm", Section: "standard", Number: 33, Label: "Confirm",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Confirm[dashboard.State]{
				Title: "Archive old logs?", Lines: []string{"Entries older than a week are moved to cold storage."},
				Accept:   "Archive",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Archived"); return nil },
				OnCancel: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Left as it was"); return nil },
			}
		},
	})

	dashboard.Registry.Add(dashboard.Option{
		Key: "acknowledge", Section: "standard", Number: 34, Label: "Confirm (acknowledge)",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Confirm[dashboard.State]{
				Title: "Heads up", Lines: []string{"With Accept set to \"Back\", Enter and Esc both just close the pop-up."},
				Accept: "Back",
			}
		},
	})

	dashboard.Registry.Add(dashboard.Option{
		Key: "typed", Section: "standard", Number: 35, Label: "Type to confirm",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Typed[dashboard.State]{
				Title: "Delete the archive?", Lines: []string{"This cannot be undone."},
				Word: "DELETE", Prompt: "Type DELETE to continue, or press Enter to skip",
				OnConfirm: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Archive deleted"); return nil },
				OnEmpty:   func(c *dashboard.Ctx) tea.Cmd { c.Flash("Nothing deleted"); return nil },
			}
		},
	})
}
