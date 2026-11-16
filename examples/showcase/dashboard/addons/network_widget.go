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

// Network: a mirrored live waveform, download above the axis and upload below.
// History is kept with dashygotui.Local and fed by the widget's Tick, not held in package state.
// The graph follows the clock: it always spans the last 2 minutes, so it scrolls
// at the same speed whatever the refresh rate (compare CPU Waveforms, which
// follows refreshes and speeds up with +).

// traffic is the widget's two series and the scale they share, kept per
// dashboard.
type traffic struct {
	in, out *chart.Series
	scaler  *chart.WaveScaler
}

const window = 2 * time.Minute

func networkHistory(c *dashboard.Ctx) *traffic {
	return dashygotui.LocalWith(c, "network", func() *traffic {
		return &traffic{in: chart.NewSeries(window), out: chart.NewSeries(window),
			scaler: chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100_000})}
	})
}

func networkTick(c *dashboard.Ctx, now time.Time) tea.Cmd {
	if c.Ready() && c.Err() == nil {
		s := c.State()
		tr := networkHistory(c)
		tr.in.Add(now, s.NetIn)
		tr.out.Add(now, s.NetOut)
		tr.scaler.UpdateSync(s.NetIn, s.NetOut, now) // both directions share one scale
	}
	return nil
}

var networkWidget = dashboard.Widget{
	Tick:  networkTick,
	Title: "Network", Subtitle: "last 2 minutes, by the clock",
	Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
		th, s, tr := c.Theme(), c.State(), networkHistory(c)
		rows := []string{
			panel.Row(th, "In", th.Green.Render(th.G.Down+" ")+text.Rate(s.NetIn)),
			panel.Row(th, "Out", th.Cyan.Render(th.G.Up+" ")+text.Rate(s.NetOut)),
		}
		h := chart.GraphRows(size.Cols, size.Rows-len(rows))
		if h%2 == 0 { // a mirrored wave wants an odd height, so the axis sits in the middle
			h--
		}
		if h < 3 {
			return rows
		}
		recv := chart.Resample(tr.in.Samples(), c.Now(), window, chart.DefaultHold, 2*size.Cols)
		sent := chart.Resample(tr.out.Samples(), c.Now(), window, chart.DefaultHold, 2*size.Cols)
		return append(rows, chart.DualWave(th, recv, sent, tr.scaler.Scale(), size.Cols, h)...)
	},
}
