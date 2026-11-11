package addons

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// Charts page, panel 1: a Braille area graph (chart.Braille) over a sparkline.

func init() {
	var cpu []float64
	dashboard.Registry.OnTick(func(c *dashboard.Ctx, _ time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			cpu = append(cpu, c.State().CPU)
			if len(cpu) > 400 {
				cpu = cpu[len(cpu)-400:]
			}
		}
		return nil
	})

	dashboard.Registry.Widget(dashboard.Widget{
		Key: "chart_area", Page: "charts", Order: 10, Title: "Area & Sparkline", Subtitle: "chart.Braille and chart.Sparkline",
		Badge: func(c *dashboard.Ctx) string { return c.Theme().Muted.Render("CPU %") },
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			w := max(size.Cols-panel.LabelWidth-1, 1)
			rows := []string{panel.Row(th, "Spark", chart.Sparkline(th, cpu, 100, w))}
			if h := size.Rows - 1; h >= 1 {
				pad := "         "
				for _, r := range chart.Braille(th, cpu, 100, w, h) {
					rows = append(rows, pad+r)
				}
			}
			return rows
		},
	})
}
