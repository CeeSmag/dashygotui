package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [22] Refresh Rate: the library's ready-made refresh-rate picker.

var refreshOption = stdaddons.RefreshRatePicker[dashboard.State]{Section: "appearance", Number: 22}
