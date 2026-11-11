package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [25] Transparent Background (hotkey t): a one-key switch. On leaves the
// terminal's own background alone, so a transparent or blurred terminal shows
// through; off brings the themed background back in the mode it had. It changes
// the same setting as [24] Background.

func init() {
	stdaddons.Transparency(dashboard.Registry, "appearance", 25, stdaddons.Setup{Hotkey: "t"})
}
