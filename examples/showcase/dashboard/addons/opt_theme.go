package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [20] Color Theme (hotkey m): the library's palette picker. It lists the
// built-in palettes, the 16-colour console palette and this app's own Solarized.
// Palettes with a background recolour the terminal.

var themeOption = stdaddons.ThemePicker[dashboard.State]{Section: "appearance", Number: 20, Hotkey: "m"}
