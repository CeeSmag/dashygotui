package addons

import (
	"fmt"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// System: meters, a labelled divider inside a panel, and the footer-style
// spinner driven by Context.Ticks. Shows Widget.Badge, MinHeight and Subtitle.

func init() {
	dashboard.Registry.Widget(dashboard.Widget{
		Key: "system", Page: "monitor", Order: 10, Title: "System", Subtitle: "meters and a divider", MinHeight: 9,
		Badge: func(c *dashboard.Ctx) string {
			th := c.Theme()
			if c.Err() != nil {
				return th.Red.Render(th.G.Cross + " no data")
			}
			return th.Green.Render(th.Spinner(c.Ticks()) + " live")
		},
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, s := c.Theme(), c.State()
			w := chart.MeterWidth(size.Cols)
			meter := func(pct float64) string { return fmt.Sprintf("%s %5.1f%%", chart.Meter(th, pct, w), pct) }
			return []string{
				panel.Row(th, "CPU", meter(s.CPU)),
				panel.Row(th, "Memory", meter(s.Mem)),
				panel.Row(th, "Disk", meter(s.Disk)),
				panel.Divider("TRAFFIC"), // renders as a labelled rule across the panel
				panel.Row(th, "In", th.Green.Render(th.G.Down+" ")+text.Rate(s.NetIn)),
				panel.Row(th, "Out", th.Cyan.Render(th.G.Up+" ")+text.Rate(s.NetOut)),
				panel.Row(th, "Uptime", s.Uptime.Truncate(time.Second).String()),
			}
		},
	})
}
