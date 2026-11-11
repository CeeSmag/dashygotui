package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [11] Notifications: a plain on/off toggle remembered between runs, using the
// OnOff badge and FlashResult.

func init() {
	const enabledKey = "notifications.enabled"

	enabled := func(c *dashboard.Ctx) bool {
		on := true // on by default
		_, _ = c.Store().Get(enabledKey, &on)
		return on
	}

	dashboard.Registry.Option(dashboard.Option{
		Key: "notifications", Section: "settings", Number: 11, Label: "Notifications",
		Badge: func(c *dashboard.Ctx) string { return c.OnOff(enabled(c)) },
		Run: func(c *dashboard.Ctx) tea.Cmd {
			on := !enabled(c)
			c.FlashResult("Notifications "+map[bool]string{true: "on", false: "off"}[on]+".", c.Store().Set(enabledKey, on))
			return nil
		},
	})
}
