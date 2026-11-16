package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// One panel of rows and two widgets. The library's ready-made settings are
// placed here directly; they need no file.
func themePage() dashboard.Page {
	type s = dashboard.State
	return dashboard.Page{ID: "theme", Title: "Theme", ShortTitle: "Thm", TinyTitle: "T", Layout: dashygotui.LayoutOneTopTwoBottom,
		Badge: themeBadge,
		Panels: []dashboard.Panel{
			{Title: "Appearance", Subtitle: "colours, spacing, symbols", Number: 20, Items: []dashboard.Addon{
				stdaddons.ThemePicker[s]{Hotkey: "m"},        // [20]
				stdaddons.SpacingMenu[s]{},                   // [21]
				stdaddons.RefreshRatePicker[s]{},             // [22]
				stdaddons.DisplayPicker[s]{},                 // [23]
				stdaddons.BackgroundPicker[s]{},              // [24]
				stdaddons.TransparencyToggle[s]{Hotkey: "t"}, // [25]
			}},
			{Widget: glyphsWidget},
			{Widget: paletteWidget},
		}}
}
