package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [24] Background: turn the theme's background on or off, live. Terminal asks the
// terminal to use the palette's colours; Paint also paints every cell, for
// terminals that ignore the request; Off keeps the terminal's own colours. The
// developer sets the starting mode with Config.Background (or -background); the
// user's saved choice wins.

func init() { stdaddons.Background(dashboard.Registry, "appearance", 24) }
