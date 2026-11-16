package addons

import "github.com/CeeSmag/dashygotui/examples/showcase/dashboard"

// Diagnostics (hotkey d): the row that opens the Diagnostics window. Only how the
// row looks; opening the window, and what is in it, is the map's
// (diagnostics_map.go). HotkeyHint lists the key in the footer.

var diagnosticsOpt = dashboard.Option{Label: "Diagnostics", Hotkey: "d", HotkeyHint: "Diagnostics"}
