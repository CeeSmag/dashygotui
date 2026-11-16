package addons

import (
	"fmt"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// What the Monitor page does, apart from its panels: the tab badge that says
// whether data is flowing, and the compact one-panel summary drawn on a
// terminal too small for the grid (Page.Fallback).

func monitorBadge(c *dashboard.Ctx) string {
	th := c.Theme()
	if c.Err() != nil {
		return th.Red.Render("STALE")
	}
	return th.Green.Render("LIVE")
}

func monitorSummary(c *dashboard.Ctx, size dashygotui.Size) []string {
	th, s := c.Theme(), c.State()
	w := chart.MeterWidth(size.Cols)
	meter := func(pct float64) string { return fmt.Sprintf("%s %5.1f%%", chart.Meter(th, pct, w), pct) }
	rows := []string{
		panel.Row(th, "CPU", meter(s.CPU)),
		panel.Row(th, "Memory", meter(s.Mem)),
		panel.Row(th, "Disk", meter(s.Disk)),
		panel.Row(th, "Traffic", text.Rate(s.NetIn)+" in, "+text.Rate(s.NetOut)+" out"),
	}
	return append(rows, th.Muted.Render("compact view: make the window larger for the full page"))
}
