package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [24] Background: turn the theme's background on or off, live.

var backgroundOption = stdaddons.BackgroundPicker[dashboard.State]{Section: "appearance", Number: 24}
