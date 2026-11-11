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

func init() {
	var cpu, in, out []float64
	scaler := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100_000})
	dashboard.Registry.OnTick(func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			s := c.State()
			cpu, in, out = append(cpu, s.CPU), append(in, s.NetIn), append(out, s.NetOut)
			scaler.UpdateSync(s.NetIn, s.NetOut, now)
			if len(cpu) > 400 {
				cpu, in, out = cpu[len(cpu)-400:], in[len(in)-400:], out[len(out)-400:]
			}
		}
		return nil
	})

	dashboard.Registry.Widget(dashboard.Widget{
		Key: "chart_trace", Page: "charts", Order: 30, Title: "Trace & Mirror", Subtitle: "chart.LineTrace and chart.DualWave",
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
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
