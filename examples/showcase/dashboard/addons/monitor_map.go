package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Four widgets in a 2x2 grid. The tab shows whether data is flowing. On a short or
// narrow terminal it falls back to a one-panel summary.
func monitorPage() dashboard.Page {
	return dashboard.Page{ID: "monitor", Title: "Monitor", ShortTitle: "Mon", TinyTitle: "M", Layout: dashygotui.LayoutGrid2x2,
		Badge: monitorBadge, Fallback: monitorSummary, FallbackBelow: dashygotui.Size{Cols: 60, Rows: 15},
		Panels: []dashboard.Panel{
			{Widget: systemWidget},
			{Widget: networkWidget},
			{Widget: activityWidget},
			{Widget: wavesWidget},
		}}
}
