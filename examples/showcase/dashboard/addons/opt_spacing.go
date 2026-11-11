package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [21] Section Gap: the library's four live spacing sliders.

func init() { stdaddons.Spacing(dashboard.Registry, "appearance", 21) }
