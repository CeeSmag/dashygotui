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

// barSamples are the widget's two series, kept per dashboard.
type barSamples struct{ net, mem []float64 }

func init() {
	samples := func(c *dashboard.Ctx) *barSamples { return dashygotui.Local[barSamples](c, "chart_bars") }

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, _ time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			s, b := c.State(), samples(c)
			b.net, b.mem = append(b.net, s.NetIn), append(b.mem, s.Mem)
			if len(b.net) > 400 {
				b.net, b.mem = b.net[len(b.net)-400:], b.mem[len(b.mem)-400:]
			}
		}
		return nil
	})

	dashboard.Registry.Add(dashboard.Widget{
		Key: "chart_bars", Page: "charts", Order: 20, Title: "Bars & Heat", Subtitle: "chart.BarGraph and chart.HeatStrip",
		Badge: func(c *dashboard.Ctx) string {
			net := samples(c).net
			scale := chart.GraphScale(net[max(len(net)-60, 0):], 100_000)
			return c.Theme().Muted.Render("scale " + text.RateShort(scale))
		},
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, b := c.Theme(), samples(c)
			net, mem := b.net, b.mem
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
