package addons

import (
	"fmt"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/theme"
)

// Palette: the active palette, its gradient, every available palette (including
// the app's own Solarized) and whether the terminal's background is set.

func init() {
	dashboard.Registry.Widget(dashboard.Widget{
		Key: "palette", Page: "theme", Order: 20, Title: "Palette", Subtitle: "theme.Palette and Backdrop", MinHeight: 9,
		Badge: func(c *dashboard.Ctx) string { return c.Theme().Muted.Render(c.PaletteID()) },
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			cell := th.G.MeterFill
			ramp := make([]float64, 40)
			for i := range ramp {
				ramp[i] = float64(i+1) * 2.5
			}
			backdrop := "none (the terminal keeps its own colours)"
			if bg, _, ok := th.Backdrop(); ok {
				r, g, b, _ := bg.RGBA()
				backdrop = fmt.Sprintf("#%02X%02X%02X (set on the terminal)", r>>8, g>>8, b>>8)
			}
			rows := []string{
				panel.Row(th, "Name", th.White.Render(th.Palette.Name)+"  "+th.Muted.Render(th.Palette.Vibe)),
				panel.Row(th, "Ramp", chart.HeatStrip(th, ramp, 100, min(40, max(size.Cols-10, 1)))),
				panel.Row(th, "Backdrop", backdrop),
				panel.Divider("ALL PALETTES"),
			}
			for _, p := range c.Palettes() {
				marker := "  "
				if p.ID == c.PaletteID() {
					marker = th.Key.Render(th.G.Pointer) + " "
				}
				rows = append(rows, marker+theme.SwatchWith(p, cell)+" "+p.Name)
			}
			return rows
		},
	})
}
