package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [20] Color Theme: the library's ready-made palette picker.

var themeOption = stdaddons.ThemePicker[dashboard.State]{Section: "appearance", Number: 20}
