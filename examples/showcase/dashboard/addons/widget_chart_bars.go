package addons

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// Charts page, panel 2: a bar graph of inbound traffic, auto-scaled with
// chart.GraphScale, over a heat strip of memory.

func init() {
	var net, mem []float64
	dashboard.Registry.OnTick(func(c *dashboard.Ctx, _ time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			s := c.State()
			net, mem = append(net, s.NetIn), append(mem, s.Mem)
			if len(net) > 400 {
				net, mem = net[len(net)-400:], mem[len(mem)-400:]
			}
		}
		return nil
	})

	dashboard.Registry.Widget(dashboard.Widget{
		Key: "chart_bars", Page: "charts", Order: 20, Title: "Bars & Heat", Subtitle: "chart.BarGraph and chart.HeatStrip",
		Badge: func(c *dashboard.Ctx) string {
			scale := chart.GraphScale(net[max(len(net)-60, 0):], 100_000)
			return c.Theme().Muted.Render("scale " + text.RateShort(scale))
		},
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			w := max(size.Cols-panel.LabelWidth-1, 1)
			recent := net[max(len(net)-w, 0):]
			scale := chart.GraphScale(recent, 100_000)
			rows := []string{panel.Row(th, "Memory", chart.HeatStrip(th, mem, 100, w))}
			if h := size.Rows - 1; h >= 1 {
				for _, r := range chart.BarGraph(th, net, scale, w, h) {
					rows = append(rows, "         "+r)
				}
			}
			return rows
		},
	})
}
