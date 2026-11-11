package addons

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// Charts page, panel 3: an oscilloscope-style line trace and a mirrored wave,
// the latter kept steady by a chart.WaveScaler with hysteresis.

// traceSamples are the widget's series and the scale they share, kept per
// dashboard.
type traceSamples struct {
	cpu, in, out []float64
	scaler       *chart.WaveScaler
}

func init() {
	samples := func(c *dashboard.Ctx) *traceSamples {
		return dashygotui.LocalWith(c, "chart_trace", func() *traceSamples {
			return &traceSamples{scaler: chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100_000})}
		})
	}

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			s, t := c.State(), samples(c)
			t.cpu, t.in, t.out = append(t.cpu, s.CPU), append(t.in, s.NetIn), append(t.out, s.NetOut)
			t.scaler.UpdateSync(s.NetIn, s.NetOut, now)
			if len(t.cpu) > 400 {
				t.cpu, t.in, t.out = t.cpu[len(t.cpu)-400:], t.in[len(t.in)-400:], t.out[len(t.out)-400:]
			}
		}
		return nil
	})

	dashboard.Registry.Add(dashboard.Widget{
		Key: "chart_trace", Page: "charts", Order: 30, Title: "Trace & Mirror", Subtitle: "chart.LineTrace and chart.DualWave",
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th, t := c.Theme(), samples(c)
			cpu, in, out, scaler := t.cpu, t.in, t.out, t.scaler
			w := max(size.Cols-panel.LabelWidth-1, 1)
			pad := "         "
			half := size.Rows / 2
			var rows []string
			if half >= 1 {
				for _, r := range chart.LineTrace(th, cpu, 100, w, half) {
					rows = append(rows, pad+r)
				}
			}
			dual := size.Rows - half
			if dual%2 == 0 {
				dual-- // keep the axis in the middle
			}
			if dual >= 3 {
				for _, r := range chart.DualWave(th, in, out, scaler.Scale(), w, dual) {
					rows = append(rows, pad+r)
				}
			}
			return rows
		},
	})
}
