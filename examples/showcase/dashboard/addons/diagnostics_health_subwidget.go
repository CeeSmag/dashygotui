package addons

import (
	"fmt"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// Health: the top panel of the Diagnostics subpage.

var diagnosticsHealthWidget = dashboard.Widget{
	Title: "Health", Subtitle: "health of the service", MinHeight: 8,
	Render: func(c *dashboard.Ctx, _ dashygotui.Size) []string {
		th, s := c.Theme(), c.State()
		rows := []string{
			panel.Row(th, "Uptime", s.Uptime.Truncate(time.Second).String()),
			panel.Row(th, "Requests", fmt.Sprintf("%d served", s.Requests)),
			panel.Row(th, "Pushed", fmt.Sprintf("%d events (Dashboard.Notify)", s.Events)),
			panel.Row(th, "Traffic", text.Rate(s.NetIn)+" in, "+text.Rate(s.NetOut)+" out"),
		}
		if err := c.Err(); err != nil {
			rows = append(rows, panel.Row(th, "Error", th.Red.Render(err.Error())))
		}
		return rows
	},
}
