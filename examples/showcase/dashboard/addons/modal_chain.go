package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [42] New Project: pop-ups chained into a flow. Each step closes itself and
// opens the next from its callback: Confirm, then Edit, then Notice.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "chain", Section: "custom", Number: 42, Label: "New Project (3 steps)",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Confirm[dashboard.State]{
				Title: "Step 1 of 3", Lines: []string{"Create a new project?"}, Accept: "Continue",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd {
					c.OpenModal(&modal.Edit[dashboard.State]{
						Title: "Step 2 of 3", Help: "Name the project.",
						Submit: func(c *dashboard.Ctx, name string) tea.Cmd {
							c.OpenModal(&modal.Notice[dashboard.State]{
								Title: "Step 3 of 3", Badge: c.Badge("done"),
								Lines: []string{"Project \"" + name + "\" was created."},
							})
							return nil
						},
					})
					return nil
				},
			})
			return nil
		},
	})
}
