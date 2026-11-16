package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
)

// [11] Notifications: a simple on/off toggle that is remembered between runs.

const enabledKey = "notifications.enabled"

func enabled(c *dashboard.Ctx) bool {
	on := true // on by default
	_, _ = c.Store().Get(enabledKey, &on)
	return on
}

var notificationsOption = dashboard.Option{
	Key: "notifications", Section: "settings", Number: 11, Label: "Notifications",
	Badge: func(c *dashboard.Ctx) string { return c.OnOff(enabled(c)) },
	Do: func(c *dashboard.Ctx) {
		on := !enabled(c)
		c.FlashResult("Notifications "+map[bool]string{true: "on", false: "off"}[on]+".", c.Store().Set(enabledKey, on))
	},
}
