// Package addons holds the showcase's features, one per file, and the map that
// places them: this file. A feature file says what the feature does; only the
// map says where it goes. Rows are numbered by position; a panel's Number sets
// where its count starts.
package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// Map is what the showcase's dashboard contains and where.
func Map() dashboard.Map {
	type s = dashboard.State // the library's ready-made addons take the state type
	return dashboard.Map{
		Shortcuts: []dashboard.Shortcut{refreshShortcut, stopShortcut, clearlogsShortcut, helpShortcut},
		Hooks:     []dashboard.Hook{alertHook, exportResult, execResult, streamTick},
		Pages: []dashboard.Page{
			// Four widgets in a 2x2 grid. The tab shows whether data is flowing. On a
			// short or narrow terminal it falls back to a one-panel summary.
			{ID: "monitor", Title: "Monitor", ShortTitle: "Mon", TinyTitle: "M", Layout: dashygotui.LayoutGrid2x2,
				Badge: monitorBadge, Fallback: monitorSummary, FallbackBelow: dashygotui.Size{Cols: 60, Rows: 15},
				Panels: []dashboard.Panel{
					{Widget: systemWidget},
					{Widget: networkWidget},
					{Widget: activityWidget},
					{Widget: wavesWidget},
				}},
			// Four charts stacked. FallbackWhen says when to show the compact summary
			// with any rule; FallbackBelow is for the simple "narrower or shorter".
			{ID: "charts", Title: "Charts", ShortTitle: "Chrt", TinyTitle: "C", Layout: dashygotui.LayoutRows,
				Fallback: chartsSummary, FallbackWhen: chartsTooSmall,
				Panels: []dashboard.Panel{
					{Widget: chartAreaWidget},
					{Widget: chartBarsWidget},
					{Widget: chartTraceWidget},
					{Widget: chartStreamsWidget},
				}},
			// Three panels of rows: Auto puts one above two.
			{ID: "actions", Title: "Actions", ShortTitle: "Act", TinyTitle: "A", Layout: dashygotui.LayoutAuto,
				Panels: []dashboard.Panel{
					// Noun changes the default "[4 options]" badge to "[4 actions]".
					{Title: "Quick Actions", Subtitle: "things to do", Noun: "action", Number: 1, Items: []dashboard.Addon{
						backupOption, restartOption, exportOption, webOption, // [1] to [4]
					}},
					// Badge replaces the count badge entirely.
					{Title: "Settings", Subtitle: "saved between runs", Number: 10, Badge: persistedBadge, Items: []dashboard.Addon{
						storageOption, notificationsSetting, tokenOption, regionOption, volumeOption,
						hostnameOption, retentionSetting, loglevelSetting, displayNameSetting, // [10] to [18]
					}},
					{Title: "Danger Zone", Subtitle: "think twice", Number: 90, Items: []dashboard.Addon{resetOption}},
				}},
			// Two panels side by side.
			{ID: "popups", Title: "Pop-ups", ShortTitle: "Pop", TinyTitle: "P", Layout: dashygotui.LayoutColumns,
				Panels: []dashboard.Panel{
					{Title: "Standard Pop-ups", Subtitle: "from the modal package", Number: 30, Items: []dashboard.Addon{
						noticeOption, noticeLiveOption, noticeScrollOption, confirmOption, acknowledgeOption,
						typedOption, choiceOption, sliderOption, editOption, // [30] to [38]
					}},
					{Title: "Custom Pop-ups", Subtitle: "written for this app", Number: 40, Items: []dashboard.Addon{
						diceOption, pingOption, chainOption, // [40] to [42]
					}},
				}},
			// Two above one.
			{ID: "views", Title: "Views & Hooks", ShortTitle: "View", TinyTitle: "V", Layout: dashygotui.LayoutTwoTopOneBottom,
				Panels: []dashboard.Panel{
					{Title: "Full-screen Views", Subtitle: "Esc, b or Backspace returns", Number: 50, Items: []dashboard.Addon{
						// [50] a sub-page: a Page as a row, with panels of its own.
						dashboard.Page{ID: "diagnostics", Title: "Diagnostics", Badge: diagnosticsBadge, Session: diagnosticsSession,
							Entry: &dashboard.Entry{Hotkey: "d", HotkeyHint: "Diagnostics"},
							Panels: []dashboard.Panel{
								{Widget: diagnosticsHealthWidget},
								{Title: "Actions", Subtitle: "Enter runs the highlighted one", Items: []dashboard.Addon{
									diagnosticsSelfTestOpt, diagnosticsLogsOpt, diagnosticsBackOpt, // [1] to [3]
								}},
							}},
						processesView, // [51]
						// [52] the library's log view, a ready-made addon placed with its settings.
						stdaddons.LogView[s]{Buffer: dashboard.Logs, Hotkey: "g",
							Categories: []string{"net", "disk", "auth"}, CategoryKey: "c"},
						notesView,  // [53]
						streamView, // [54]
					}},
					{Title: "Navigation", Subtitle: "Context.GotoPage", Number: 60, Items: []dashboard.Addon{
						gotoChartsOption, gotoThemeOption, gotoMonitorOption, // [60] to [62]
					}},
					{Title: "Hooks & Tasks", Subtitle: "messages, results, processes", Number: 70, Items: []dashboard.Addon{
						alertOption, outageOption, execOption, logtestOption, // [70] to [73]
					}},
				}},
			// One panel of rows and two widgets.
			{ID: "theme", Title: "Theme", ShortTitle: "Thm", TinyTitle: "T", Layout: dashygotui.LayoutOneTopTwoBottom,
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
				}},
		},
	}
}
