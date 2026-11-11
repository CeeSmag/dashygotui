package addons

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// Activity: a rolling log of request bursts, built from an OnTick hook.

func init() {
	var lines []string
	seen := 0

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		if !c.Ready() {
			return nil
		}
		if total := c.State().Requests; total != seen {
			lines = append(lines, fmt.Sprintf("%s  +%d requests (%d total)", now.Format("15:04:05"), total-seen, total))
			seen = total
			if len(lines) > 100 {
				lines = lines[len(lines)-100:]
			}
		}
		return nil
	})

	dashboard.Registry.Widget(dashboard.Widget{
		Key: "activity", Page: "monitor", Order: 30, Title: "Activity", Subtitle: "newest at the bottom",
		Render: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			if len(lines) == 0 {
				return []string{th.Muted.Render("waiting for traffic...")}
			}
			shown := lines[max(len(lines)-size.Rows, 0):]
			out := make([]string, len(shown))
			for i, l := range shown {
				out[i] = th.Muted.Render(l[:8]) + l[8:]
			}
			return panel.VStack(out)
		},
	})
}
