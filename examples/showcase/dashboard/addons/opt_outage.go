package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [71] Simulate Outage: sets a flag in the Store that makes Poll fail. Watch
// the Monitor tab turn STALE and the header turn red: Context.Err reports the
// poll error while Context.State keeps the last good data.

func init() {
	outage := func(c *dashboard.Ctx) bool {
		var on bool
		_, _ = c.Store().Get(dashboard.OutageKey, &on)
		return on
	}

	dashboard.Registry.Option(dashboard.Option{
		Key: "outage", Section: "hooks", Number: 71, Label: "Simulate Outage",
		Badge: func(c *dashboard.Ctx) string { return c.OnOff(outage(c)) },
		Run: func(c *dashboard.Ctx) tea.Cmd {
			on := !outage(c)
			if err := c.Store().Set(dashboard.OutageKey, on); err != nil {
				c.FlashError(err.Error())
				return nil
			}
			c.Flash(map[bool]string{true: "Outage started: the next poll fails", false: "Outage over"}[on])
			return c.Refresh() // poll now instead of waiting for the next interval
		},
	})
}
