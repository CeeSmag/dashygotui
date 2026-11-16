package addons

import (
	"fmt"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// The Charts page's compact form: the numbers without the graphs. When to show
// it is any rule at all (Page.FallbackWhen, in the map).

func chartsSummary(c *dashboard.Ctx, size dashygotui.Size) []string {
	th, s := c.Theme(), c.State()
	return []string{
		panel.Row(th, "CPU", fmt.Sprintf("%5.1f%%", s.CPU)),
		panel.Row(th, "Memory", fmt.Sprintf("%5.1f%%", s.Mem)),
		panel.Row(th, "Traffic", text.Rate(s.NetIn)+" in, "+text.Rate(s.NetOut)+" out"),
		th.Muted.Render(th.Trunc("compact view: the graphs need a larger window", size.Cols)),
	}
}

func chartsTooSmall(s dashygotui.Size) bool { return (s.Cols < 80 && s.Rows < 22) || s.Rows < 10 }
