package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [4] Logs: the library's scrollable log view, newest entry first, with "g" as
// its hotkey from any page.

var logsOption = stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs, Section: "actions", Number: 4, Hotkey: "g"}
