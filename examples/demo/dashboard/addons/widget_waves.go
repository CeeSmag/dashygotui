package addons

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// Waveforms: the same CPU history drawn four ways, to show the chart package's
// renderers. The history is kept with dashygotui.Local and fed by a tick hook.

func init() {
	const keep = 120

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, _ time.Time) tea.Cmd {
		if c.Ready() {
			cpu := dashygotui.Local[[]float64](c, "waves.cpu")
			*cpu = append(*cpu, c.State().CPU)
			if len(*cpu) > keep {
				*cpu = (*cpu)[len(*cpu)-keep:]
			}
		}
		return nil
	})

	dashboard.Registry.Add(dashboard.Widget{
		Key: "waves", Page: "monitor", Order: 40, Title: "CPU Waveforms", Subtitle: "one series, four styles",
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, cpu := c.Theme(), *dashygotui.Local[[]float64](c, "waves.cpu")
			w := max(size.Cols-panel.LabelWidth-1, 1)
			indent := strings.Repeat(" ", panel.LabelWidth+1)
			rows := []string{
				panel.Row(th, "Spark", chart.Sparkline(th, cpu, 100, w)),
				panel.Row(th, "Heat", chart.HeatStrip(th, cpu, 100, w)),
			}
			// What is left is shared between a bar graph and a line trace.
			if h := (size.Rows - len(rows)) / 2; h >= 1 {
				for i, r := range chart.BarGraph(th, cpu, 100, w, h) {
					label := indent
					if i == 0 {
						label = th.Muted.Render("Bars    ") + " "
					}
					rows = append(rows, label+r)
				}
				for i, r := range chart.LineTrace(th, cpu, 100, w, h) {
					label := indent
					if i == 0 {
						label = th.Muted.Render("Trace   ") + " "
					}
					rows = append(rows, label+r)
				}
			}
			return rows
		},
	})
}
