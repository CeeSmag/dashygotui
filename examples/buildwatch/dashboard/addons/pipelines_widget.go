package addons

import (
	"fmt"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
)

// The latest build of every pipeline, coloured by its status.

var pipelinesWidget = dashboard.Widget{
	Title: "Pipelines",
	Render: func(c *dashboard.Ctx, _ dashygotui.Size) []string {
		th := c.Theme()
		var rows []string
		for _, b := range c.State().Builds {
			mark := map[string]string{"passed": th.Green.Render(th.G.Check), "failed": th.Red.Render(th.G.Cross)}[b.Status]
			if mark == "" {
				mark = th.Amber.Render(th.G.Bullet)
			}
			rows = append(rows, fmt.Sprintf("%s #%d  %s  %s  %s", mark, b.Number, th.White.Render(b.Pipeline), b.Status, b.Took))
		}
		return rows
	},
}
