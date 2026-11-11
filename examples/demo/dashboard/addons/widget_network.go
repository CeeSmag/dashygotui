package addons

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// Network: a live Braille graph. It samples on every tick with an OnTick hook
// and keeps its history in closure variables, not package state, so the file
// stays self-contained.

func init() {
	const window = 2 * time.Minute
	series := chart.NewSeries(window)
	scaler := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100_000})

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		if c.Ready() {
			in := c.State().NetIn
			series.Add(now, in)
			scaler.Update(in, now)
		}
		return nil
	})

	dashboard.Registry.Widget(dashboard.Widget{
		Key: "network", Page: "monitor", Order: 20, Title: "Network", Subtitle: "last 2 minutes",
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, s := c.Theme(), c.State()
			rows := []string{
				panel.Row(th, "In", th.Green.Render("▼ ")+text.Rate(s.NetIn)),
				panel.Row(th, "Out", th.Cyan.Render("▲ ")+text.Rate(s.NetOut)),
			}
			h := chart.GraphRows(size.Cols, size.Rows-len(rows))
			if h == 0 {
				return rows
			}
			vals := chart.Resample(series.Samples(), c.Now(), window, chart.DefaultHold, 2*size.Cols)
			return append(rows, chart.Braille(th, vals, scaler.Scale(), size.Cols, h)...)
		},
	})
}
