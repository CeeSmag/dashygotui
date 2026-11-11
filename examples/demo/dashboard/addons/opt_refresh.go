package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [22] Refresh Rate: the library's ready-made refresh-rate picker.

func init() { stdaddons.RefreshRate(dashboard.Registry, "appearance", 22) }
