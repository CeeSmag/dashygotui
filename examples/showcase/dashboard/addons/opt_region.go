package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [13] Region. This is the raw modal API on show, to demonstrate Choice's fields.
// For an ordinary pick-one setting do not write this: use stdaddons.Pick (see [17]
// Log Level, which is a dozen lines).
//
// A modal.Choice with live OnMove feedback, OnSelect to save and
// OnCancel. The saved value highlights when the picker reopens. Badge puts the
// saved region in the top border, Intro adds live lines above the list, Rows
// marks the saved entry afresh on every frame, and FooterSelect and
// FooterCancel reword the footer.

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

	dashboard.Registry.Add(dashboard.Option{
		Key: "region", Section: "settings", Number: 13, Label: "Region",
		Badge: func(c *dashboard.Ctx) string { return c.Badge(regions[current(c)].Label) },
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Choice[dashboard.State]{
				Title: "Region",
				Help:  "Where the service runs. Moving the highlight reports each choice; Enter saves it.",
				Badge: func(c *dashboard.Ctx) string { return c.Badge("now " + regions[current(c)].Label) },
				Intro: func(c *dashboard.Ctx, _ int) []string {
					return []string{c.Theme().Muted.Render("Changing the region moves the service on the next restart.")}
				},
				Rows: func(c *dashboard.Ctx) []modal.Item {
					items := make([]modal.Item, len(regions))
					copy(items, regions)
					items[current(c)].Badge = c.Theme().Green.Render("[current]")
					return items
				},
				FooterSelect: "Save", FooterCancel: "Keep current",
				Cursor: current(c),
				OnMove: func(c *dashboard.Ctx, i int) { c.Flash("Highlighted " + regions[i].Label) },
				OnSelect: func(c *dashboard.Ctx, i int) tea.Cmd {
					c.FlashResult("Region set to "+regions[i].Label+".", c.Store().Set(regionKey, regions[i].Label))
					return nil
				},
				OnCancel: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Region unchanged"); return nil },
			}
		},
	})
}
