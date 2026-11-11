package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [4] Logs: the library's scrollable log view, newest entry first, with "g" as
// its hotkey from any page.

func init() {
	stdaddons.Logs(dashboard.Registry, dashboard.Logs, "actions", 4, stdaddons.Setup{Hotkey: "g"})
}
