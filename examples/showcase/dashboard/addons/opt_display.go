package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [23] Display Mode: switch between the Unicode and TTY symbol sets, live. Auto
// picks the TTY set on the Linux console.

var displayOption = stdaddons.DisplayPicker[dashboard.State]{Section: "appearance", Number: 23}
