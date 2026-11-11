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

// streams are the widget's two live graphs, kept per dashboard.
type streams struct {
	load *chart.Stream
	io   *chart.DualStream
}

func init() {
	graphs := func(c *dashboard.Ctx) *streams {
		return dashygotui.LocalWith(c, "chart_streams", func() *streams {
			opts := chart.StreamOpts{Scaler: chart.ScalerOpts{MinScale: 100_000}}
			return &streams{load: chart.NewStream(opts), io: chart.NewDualStream("Read", "Write", opts)}
		})
	}

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, _ time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			s, g := c.State(), graphs(c)
			g.load.Push(s.NetIn)
			g.io.Push(s.NetIn*0.6, s.NetOut*1.4)
		}
		return nil
	})

	dashboard.Registry.Add(dashboard.Widget{
		Key: "chart_streams", Page: "charts", Order: 40, Title: "Streams", Subtitle: "chart.Stream and chart.DualStream", MinHeight: 9,
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, g := c.Theme(), graphs(c)
			load, io := g.load, g.io
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
