package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [22] Refresh Rate: the library's refresh-rate picker. The + and - keys step
// through the same rates.

func init() { stdaddons.RefreshRate(dashboard.Registry, "appearance", 22) }
