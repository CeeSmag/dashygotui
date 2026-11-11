package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [33] [34] [35] modal.Confirm (yes/no), Confirm used as an acknowledgement
// ("Back"), and modal.Typed with an OnEmpty escape hatch.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "confirm", Section: "standard", Number: 33, Label: "Confirm",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Confirm[dashboard.State]{
				Title: "Archive old logs?", Lines: []string{"Entries older than a week are moved to cold storage."},
				Accept:   "Archive",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Archived"); return nil },
				OnCancel: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Left as it was"); return nil },
			})
			return nil
		},
	})

	dashboard.Registry.Option(dashboard.Option{
		Key: "acknowledge", Section: "standard", Number: 34, Label: "Confirm (acknowledge)",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Confirm[dashboard.State]{
				Title: "Heads up", Lines: []string{"With Accept set to \"Back\", Enter and Esc both just close the pop-up."},
				Accept: "Back",
			})
			return nil
		},
	})

	dashboard.Registry.Option(dashboard.Option{
		Key: "typed", Section: "standard", Number: 35, Label: "Type to confirm",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Typed[dashboard.State]{
				Title: "Delete the archive?", Lines: []string{"This cannot be undone."},
				Word: "DELETE", Prompt: "Type DELETE to continue, or press Enter to skip",
				OnConfirm: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Archive deleted"); return nil },
				OnEmpty:   func(c *dashboard.Ctx) tea.Cmd { c.Flash("Nothing deleted"); return nil },
			})
			return nil
		},
	})
}
