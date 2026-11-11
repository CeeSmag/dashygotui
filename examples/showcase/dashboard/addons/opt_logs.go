package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [52] Logs (hotkey g): the library's scrollable log view, newest entry first,
// holding your place as new entries arrive. It captures the standard logger and
// slog; [73] writes some of each.

func init() {
	stdaddons.Logs(dashboard.Registry, dashboard.Logs, "views", 52, stdaddons.Setup{Hotkey: "g"})
}
