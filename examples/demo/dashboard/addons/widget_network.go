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

// Network: a live mirrored waveform, download above the axis and upload below. It samples on every tick with an OnTick hook
// and keeps its history in closure variables, not package state, so the file
// stays self-contained.

func init() {
	const window = 2 * time.Minute
	in, out := chart.NewSeries(window), chart.NewSeries(window)
	scaler := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100_000})

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		if c.Ready() {
			s := c.State()
			in.Add(now, s.NetIn)
			out.Add(now, s.NetOut)
			scaler.UpdateSync(s.NetIn, s.NetOut, now) // both directions share one scale
		}
		return nil
	})

	dashboard.Registry.Widget(dashboard.Widget{
		Key: "network", Page: "monitor", Order: 20, Title: "Network", Subtitle: "last 2 minutes",
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, s := c.Theme(), c.State()
			rows := []string{
				panel.Row(th, "In", th.Green.Render(th.G.Down+" ")+text.Rate(s.NetIn)),
				panel.Row(th, "Out", th.Cyan.Render(th.G.Up+" ")+text.Rate(s.NetOut)),
			}
			// A mirrored wave wants an odd height, so the axis sits in the middle.
			h := chart.GraphRows(size.Cols, size.Rows-len(rows))
			if h%2 == 0 {
				h--
			}
			if h < 3 {
				return rows
			}
			// Download grows upward from the axis, upload downward.
			recv := chart.Resample(in.Samples(), c.Now(), window, chart.DefaultHold, 2*size.Cols)
			sent := chart.Resample(out.Samples(), c.Now(), window, chart.DefaultHold, 2*size.Cols)
			return append(rows, chart.DualWave(th, recv, sent, scaler.Scale(), size.Cols, h)...)
		},
	})
}
