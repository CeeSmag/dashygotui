package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [52] Logs (hotkey g): the library's scrollable log view, newest entry first,
// holding your place as new entries arrive. It captures the standard logger and
// slog; [73] writes some of each. A strip of category tabs runs along the top:
// press c for the next tab (C for the previous). "net", "disk" and "auth" are
// declared; "ui" is not, and gets its tab when its first entry arrives.

func init() {
	stdaddons.LogsWith(dashboard.Registry, dashboard.Logs, "views", 52,
		stdaddons.LogOptions{Categories: []string{"net", "disk", "auth"}, CategoryKey: "c"},
		stdaddons.Setup{Hotkey: "g"})
}
