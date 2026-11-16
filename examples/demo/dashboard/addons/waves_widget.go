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
// renderers. The history is kept with dashygotui.Local and fed on every tick. It
// is the last 120 refreshes, one reading each, so it scrolls faster when the
// refresh rate goes up (compare Network, which follows the clock).

const wavesKeep = 120

func wavesTick(c *dashboard.Ctx, _ time.Time) tea.Cmd {
	if c.Ready() {
		cpu := dashygotui.Local[[]float64](c, "waves.cpu")
		*cpu = append(*cpu, c.State().CPU)
		if len(*cpu) > wavesKeep {
			*cpu = (*cpu)[len(*cpu)-wavesKeep:]
		}
	}
	return nil
}

var wavesWidget = dashboard.Widget{
	Title: "CPU Waveforms", Subtitle: "last 120 refreshes, four styles",
	Tick: wavesTick,
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
}
