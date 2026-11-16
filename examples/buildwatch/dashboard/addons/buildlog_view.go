package addons

import (
	"fmt"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
)

// The build log: a Follow view (a log that holds its place). The map places it
// as a row, which opens it, and gives that row its hotkey.

var buildlogView = dashboard.View{
	ID: "buildlog", Title: "Build Log", Follow: true,
	Badge: func(c *dashboard.Ctx) string { return c.Count(len(c.State().Builds)*20, "lines") },
	Body: func(c *dashboard.Ctx, _ dashygotui.Size) []string {
		var lines []string
		for _, b := range c.State().Builds {
			for step := 1; step <= 20; step++ {
				lines = append(lines, fmt.Sprintf("%s #%d  step %02d ok", b.Pipeline, b.Number, step))
			}
		}
		return lines
	},
}
