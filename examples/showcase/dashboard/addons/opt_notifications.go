package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [11] Notifications: an on/off setting remembered between runs. stdaddons.Toggle
// is the whole addon: the badge, flipping it, saving it and the message.

func init() {
	dashboard.Registry.Add(stdaddons.Toggle[dashboard.State]{
		Key: "notifications.enabled", Section: "settings", Number: 11, Label: "Notifications", Default: true,
	})
}
