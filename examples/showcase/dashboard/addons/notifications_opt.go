package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// Notifications: an on/off setting remembered between runs. stdaddons.Toggle
// is the whole addon: the badge, flipping it, saving it and the message.

var notificationsSetting = stdaddons.Toggle[dashboard.State]{
	Key: "notifications.enabled", Label: "Notifications", Default: true,
}
