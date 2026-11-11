package addons

import (
	"fmt"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// Symbols: the active glyph set. Switch it with [23] Display Mode, or start
// with -glyphs tty, to see the console-safe set.

func init() {
	dashboard.Registry.Add(dashboard.Widget{
		Key: "glyphs", Page: "theme", Order: 10, Title: "Symbols", Subtitle: "theme.Glyphs in use", MinHeight: 8,
		Badge: func(c *dashboard.Ctx) string {
			if c.TTY() {
				return c.Theme().Amber.Render("TTY")
			}
			return c.Theme().Green.Render("Unicode")
		},
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			g := th.G
			pair := func(name, glyph string) string { return fmt.Sprintf("%-9s %s", name, th.Key.Render(glyph)) }
			return []string{
				panel.Row(th, "Mode", fmt.Sprintf("%s (chosen: %s)", map[bool]string{true: "TTY", false: "Unicode"}[c.TTY()], c.GlyphMode())),
				panel.Row(th, "Marks", pair("pointer", g.Pointer)+"   "+pair("bullet", g.Bullet)+"   "+pair("ok", g.Check)+"   "+pair("fail", g.Cross)),
				panel.Row(th, "Arrows", pair("up", g.Up)+"   "+pair("down", g.Down)+"   "+pair("ellipsis", g.Ellipsis)),
				panel.Row(th, "Corners", g.CornerTL+" "+g.CornerTR+" "+g.CornerBL+" "+g.CornerBR),
				panel.Row(th, "Meter", chart.Meter(th, 60, 12)),
				panel.Row(th, "Ramp", th.Cyan.Render(fmt.Sprint(joinGlyphs(g.Ramp)))),
			}
		},
	})
}

func joinGlyphs(gs []string) string {
	out := ""
	for _, g := range gs {
		out += g
	}
	return out
}
