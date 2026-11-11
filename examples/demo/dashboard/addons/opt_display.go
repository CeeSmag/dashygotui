package addons

import (
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [23] Display Mode: switch between the Unicode and TTY symbol sets. Auto picks
// the TTY set on the Linux console.

func init() { stdaddons.Display(dashboard.Registry, "appearance", 23) }
