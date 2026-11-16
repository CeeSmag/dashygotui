package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// Two panels above one.
func viewsPage() dashboard.Page {
	return dashboard.Page{ID: "views", Title: "Views & Hooks", ShortTitle: "View", TinyTitle: "V", Layout: dashygotui.LayoutTwoTopOneBottom,
		Panels: []dashboard.Panel{
			{Title: "Full-screen Views", Subtitle: "Esc, b or Backspace returns", Number: 50, Items: []dashboard.Addon{
				diagnosticsPage(), // [50] a window: diagnostics_opt.go opens it, diagnostics_map.go fills it
				processesView,     // [51]
				// [52] the library's log view, a ready-made addon placed with its settings.
				stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs, Hotkey: "g",
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
		}}
}
