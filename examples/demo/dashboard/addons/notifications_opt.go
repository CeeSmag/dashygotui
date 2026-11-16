package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
)

// Notifications: a simple on/off toggle that is remembered between runs.

const notificationsKey = "notifications.enabled"

func notificationsOn(c *dashboard.Ctx) bool {
	on := true // on by default
	_, _ = c.Store().Get(notificationsKey, &on)
	return on
}

var notificationsOpt = dashboard.Option{
	Label: "Notifications",
	Badge: func(c *dashboard.Ctx) string { return c.OnOff(notificationsOn(c)) },
	Do: func(c *dashboard.Ctx) {
		on := !notificationsOn(c)
		c.FlashResult("Notifications "+map[bool]string{true: "on", false: "off"}[on]+".", c.Store().Set(notificationsKey, on))
	},
}
