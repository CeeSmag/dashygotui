package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Four charts stacked. FallbackWhen says when to show the compact summary with
// any rule; FallbackBelow is for the simple "narrower or shorter".
func chartsPage() dashboard.Page {
	return dashboard.Page{ID: "charts", Title: "Charts", ShortTitle: "Chrt", TinyTitle: "C", Layout: dashygotui.LayoutRows,
		Fallback: chartsSummary, FallbackWhen: chartsTooSmall,
		Panels: []dashboard.Panel{
			{Widget: chartAreaWidget},
			{Widget: chartBarsWidget},
			{Widget: chartTraceWidget},
			{Widget: chartStreamsWidget},
		}}
}
