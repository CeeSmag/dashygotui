package addons

import (
	"fmt"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// System: meters for CPU, memory and disk, read straight from the polled state.

func init() {
	dashboard.Registry.Widget(dashboard.Widget{
		Key: "system", Page: "monitor", Order: 10, Title: "System", Subtitle: "synthetic readings", MinHeight: 7,
		Badge: func(c *dashboard.Ctx) string {
			if !c.Ready() {
				return c.Badge("starting")
			}
			th := c.Theme()
			return th.Green.Render(th.G.Bullet + " LIVE")
		},
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, s := c.Theme(), c.State()
			w := chart.MeterWidth(size.Cols)
			meter := func(pct float64) string { return fmt.Sprintf("%s %5.1f%%", chart.Meter(th, pct, w), pct) }
			return []string{
				panel.Row(th, "CPU", meter(s.CPU)),
				panel.Row(th, "Memory", meter(s.Mem)),
				panel.Row(th, "Disk", meter(s.Disk)),
				panel.Row(th, "Uptime", s.Uptime.Truncate(time.Second).String()),
			}
		},
	})
}
