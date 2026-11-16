package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [22] Refresh Rate: the library's refresh-rate picker. The + and - keys step
// through the same rates.

var refreshOption = stdaddons.RefreshRatePicker[dashboard.State]{Section: "appearance", Number: 22}
