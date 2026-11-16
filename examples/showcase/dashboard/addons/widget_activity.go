package addons

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Activity: a rolling list of request bursts, built from a Hook's Tick.

// activityLog is the widget's lines and the request total it last saw, kept per
// dashboard.
type activityLog struct {
	lines []string
	seen  int
}

func activityOf(c *dashboard.Ctx) *activityLog { return dashygotui.Local[activityLog](c, "activity") }

var activityTick = dashboard.Hook{Tick: func(c *dashboard.Ctx, now time.Time) tea.Cmd {
	if !c.Ready() || c.Err() != nil {
		return nil
	}
	if total, a := c.State().Requests, activityOf(c); total != a.seen {
		a.lines = append(a.lines, fmt.Sprintf("%s  +%d requests (%d total)", now.Format("15:04:05"), total-a.seen, total))
		a.seen = total
		if len(a.lines) > 100 {
			a.lines = a.lines[len(a.lines)-100:]
		}
	}
	return nil
}}

var activityWidget = dashboard.Widget{
	Key: "activity", Page: "monitor", Order: 30, Title: "Activity", Subtitle: "newest at the bottom",
	Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
		th := c.Theme()
		lines := activityOf(c).lines
		if len(lines) == 0 {
			return []string{th.Muted.Render("waiting for traffic...")}
		}
		shown := lines[max(len(lines)-size.Rows, 0):]
		out := make([]string, len(shown))
		for i, l := range shown {
			out[i] = th.Muted.Render(l[:8]) + l[8:]
		}
		return out
	},
}
