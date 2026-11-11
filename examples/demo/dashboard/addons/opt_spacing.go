package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [21] Section Gap: the library's ready-made spacing sliders.

func init() { stdaddons.Spacing(dashboard.Registry, "appearance", 21) }
