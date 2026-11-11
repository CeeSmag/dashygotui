package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [13] Region: a modal.Choice with live OnMove feedback, OnSelect to save and
// OnCancel. The saved value highlights when the picker reopens.

func init() {
	const regionKey = "region"
	regions := []modal.Item{
		{Label: "us-east", Detail: "Virginia, lowest latency to the Americas"},
		{Label: "us-west", Detail: "Oregon"},
		{Label: "eu-west", Detail: "Ireland"},
		{Label: "eu-central", Detail: "Frankfurt, strict data residency"},
		{Label: "ap-south", Detail: "Mumbai"},
	}
	current := func(c *dashboard.Ctx) int {
		var name string
		_, _ = c.Store().Get(regionKey, &name)
		for i, r := range regions {
			if r.Label == name {
				return i
			}
		}
		return 2 // eu-west
	}

	dashboard.Registry.Option(dashboard.Option{
		Key: "region", Section: "settings", Number: 13, Label: "Region",
		Badge: func(c *dashboard.Ctx) string { return c.Badge(regions[current(c)].Label) },
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Choice[dashboard.State]{
				Title:  "Region",
				Help:   "Where the service runs. Moving the highlight reports each choice; Enter saves it.",
				Items:  regions,
				Cursor: current(c),
				OnMove: func(c *dashboard.Ctx, i int) { c.Flash("Highlighted " + regions[i].Label) },
				OnSelect: func(c *dashboard.Ctx, i int) tea.Cmd {
					c.FlashResult("Region set to "+regions[i].Label+".", c.Store().Set(regionKey, regions[i].Label))
					return nil
				},
				OnCancel: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Region unchanged"); return nil },
			})
			return nil
		},
	})
}
