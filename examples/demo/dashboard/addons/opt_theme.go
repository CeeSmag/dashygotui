package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [20] Color Theme: the library's ready-made palette picker.

func init() { stdaddons.Theme(dashboard.Registry, "appearance", 20) }
