// Package addons holds the demo's features, one per file, and the map that
// places them on the pages: this file. A feature file says what the feature
// does; only the map says where it goes. The demo is small enough for one map
// file; a larger application splits it by page into <name>_map.go files.
package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// Map is what the demo's dashboard contains and where. Rows are numbered by
// position; a panel's Number sets where its count starts.
func Map() dashboard.Map {
	return dashboard.Map{
		Shortcuts: []dashboard.Shortcut{refreshShortcut, stopShortcut},
		Pages: []dashboard.Page{
			{Title: "Monitor", Layout: dashygotui.LayoutAuto, Panels: []dashboard.Panel{
				{Widget: systemWidget},
				{Widget: networkWidget},
				{Widget: activityWidget},
				{Widget: wavesWidget},
			}},
			{Title: "Control", ShortTitle: "Ctl", Panels: []dashboard.Panel{
				{Title: "Quick Actions", Subtitle: "things to do", Items: []dashboard.Addon{
					backupOpt,  // [1]
					restartOpt, // [2]
					dashboard.Page{Row: diagnosticsOpt, Panels: []dashboard.Panel{ // [3] opens a window
						{Widget: diagnosticsHealthWidget},
						{Title: "Tools", Items: []dashboard.Addon{diagnosticsSelfTestOpt}},
					}},
					stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs, Hotkey: "g"}, // [4]
				}},
				{Title: "Settings", Subtitle: "saved between runs", Number: 10, Items: []dashboard.Addon{
					storageOpt,       // [10]
					notificationsOpt, // [11]
					resetOpt,         // [12]
				}},
			}},
			{Title: "Theme", Panels: []dashboard.Panel{
				{Title: "Appearance", Subtitle: "colours and spacing", Number: 20, Items: []dashboard.Addon{
					stdaddons.ThemePicker[dashboard.State]{},       // [20]
					stdaddons.SpacingMenu[dashboard.State]{},       // [21]
					stdaddons.RefreshRatePicker[dashboard.State]{}, // [22]
					stdaddons.DisplayPicker[dashboard.State]{},     // [23]
					stdaddons.BackgroundPicker[dashboard.State]{},  // [24]
				}},
			}},
		},
	}
}
