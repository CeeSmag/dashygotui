package addons

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// Streams: chart.Stream and chart.DualStream are ready-made live graphs that
// keep their own history and adaptive scale. Push a sample per tick, render per
// frame, and there is no series, resampling or scaler code to write. The scale
// tag ("≤1.2M/s") says what the top of the graph means.

func init() {
	load := chart.NewStream(chart.StreamOpts{Scaler: chart.ScalerOpts{MinScale: 100_000}})
	io := chart.NewDualStream("Read", "Write", chart.StreamOpts{Scaler: chart.ScalerOpts{MinScale: 100_000}})

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, _ time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			s := c.State()
			load.Push(s.NetIn)
			io.Push(s.NetIn*0.6, s.NetOut*1.4)
		}
		return nil
	})

	dashboard.Registry.Widget(dashboard.Widget{
		Key: "chart_streams", Page: "charts", Order: 40, Title: "Streams", Subtitle: "chart.Stream and chart.DualStream", MinHeight: 9,
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			rows := []string{
				panel.Row(th, "Load", load.Sparkline(th, max(size.Cols-panel.LabelWidth-14, 4))+" "+load.ScaleTag(th, "")),
			}
			if h := size.Rows - len(rows); h >= 1 {
				rows = append(rows, io.Render(th, size.Cols, h)...)
			}
			return rows
		},
	})
}
