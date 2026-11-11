package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [24] Background: turn the theme's background on or off, live.

func init() { stdaddons.Background(dashboard.Registry, "appearance", 24) }
