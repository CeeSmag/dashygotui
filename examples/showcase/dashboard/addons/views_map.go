package addons

import (
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// Two panels above one.
func viewsPage() dashboard.Page {
	return dashboard.Page{ID: "views", Title: "Views & Hooks", ShortTitle: "View", TinyTitle: "V", Layout: dashygotui.LayoutTwoTopOneBottom,
		Panels: []dashboard.Panel{
			{Title: "Full-screen Views", Subtitle: "Esc, b or Backspace returns", Number: 50, Items: []dashboard.Addon{
				// [50] a subpage: diagnostics_subpage.go opens it, diagnostics_map.go fills it.
				dashboard.Hotkey{Key: "d", Hint: "Diagnostics", Item: diagnosticsPage()},
				dashboard.Hotkey{Key: "p", Item: processesView}, // [51]
				// [52] the library's log view, a ready-made addon placed with its settings.
				dashboard.Hotkey{Key: "g", Item: stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs,
					Categories: []string{"net", "disk", "auth"}, CategoryKey: "c"}},
				dashboard.Hotkey{Key: "n", Item: notesView},  // [53]
				dashboard.Hotkey{Key: "e", Item: streamView}, // [54]
				dashboard.Hotkey{Key: "f", Item: stdaddons.DirBrowser[dashboard.State]{
					Label: "Pick a Folder",
					Start: func(*dashboard.Ctx) string { return os.TempDir() },
					OnSelect: func(c *dashboard.Ctx, path string) tea.Cmd {
						c.Flash("Selected " + path)
						return nil
					},
				}}, // [55]
			}},
			{Title: "Navigation", Subtitle: "Context.GotoPage", Number: 60, Items: []dashboard.Addon{
				gotoChartsOption, gotoThemeOption, gotoMonitorOption, // [60] to [62]
			}},
			{Title: "Hooks & Tasks", Subtitle: "messages, results, processes", Number: 70, Items: []dashboard.Addon{
				alertOption, outageOption, execOption, logtestOption, // [70] to [73]
			}},
		}}
}
