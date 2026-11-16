// Package addons holds the buildwatch's features, one per file, and the map
// that places them: this file.
package addons

import "github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"

// Map is what the dashboard contains and where.
func Map() dashboard.Map {
	return dashboard.Map{
		Shortcuts: []dashboard.Shortcut{stopallShortcut},
		Pages: []dashboard.Page{
			{Title: "Builds", Panels: []dashboard.Panel{
				{Widget: pipelinesWidget},
				{Title: "Actions", Items: []dashboard.Addon{
					dashboard.Hotkey{Key: "l", Hint: "Build log", Item: buildlogView}, // [1]
					rerunOpt, // [2]
				}},
			}},
			{Title: "Settings", Panels: []dashboard.Panel{
				{Title: "Preferences", Subtitle: "saved between runs", Number: 10, Items: []dashboard.Addon{
					notifyFailedSetting, keepBuildsSetting, branchSetting, // [10] [11] [12]
				}},
			}},
		},
	}
}
