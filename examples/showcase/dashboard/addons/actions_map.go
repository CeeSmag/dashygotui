package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Three panels of rows: Auto puts one above two.
func actionsPage() dashboard.Page {
	return dashboard.Page{ID: "actions", Title: "Actions", ShortTitle: "Act", TinyTitle: "A", Layout: dashygotui.LayoutAuto,
		Panels: []dashboard.Panel{
			// Noun changes the default "[4 options]" badge to "[4 actions]".
			{Title: "Quick Actions", Subtitle: "things to do", Noun: "action", Number: 1, Items: []dashboard.Addon{
				backupOption, restartOption, exportOption, // [1] to [3]
				// [4] Display writes the key as "Ctrl+O" instead of "ctrl+o", and Tone draws it
				// in the theme's info colour.
				dashboard.Hotkey{Key: "ctrl+o", Display: "Ctrl+O", Tone: dashygotui.ToneInfo, Item: webOption},
			}},
			// Badge replaces the count badge entirely.
			{Title: "Settings", Subtitle: "saved between runs", Number: 10, Badge: persistedBadge, Items: []dashboard.Addon{
				dashboard.Hotkey{Key: "s", Item: storageOption}, notificationsSetting, tokenOption, regionOption, volumeOption,
				hostnameOption, retentionSetting, loglevelSetting, displayNameSetting, // [10] to [18]
			}},
			{Title: "Danger Zone", Subtitle: "think twice", Number: 90, Items: []dashboard.Addon{resetOption}},
		}}
}
