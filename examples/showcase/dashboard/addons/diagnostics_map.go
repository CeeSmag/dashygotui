package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// The Diagnostics subpage: what is inside it, and the hotkeys of its rows, which
// work only inside it. The subpage itself is diagnostics_subpage.go; its title is
// its option's label, and its rows are numbered from 1.
func diagnosticsPage() dashboard.Page {
	return dashboard.Page{ID: "diagnostics", Row: diagnosticsSubpage, Badge: diagnosticsBadge, Session: diagnosticsSession,
		Panels: []dashboard.Panel{
			{Widget: diagnosticsHealthWidget},
			{Title: "Actions", Subtitle: "Enter runs the highlighted one", Items: []dashboard.Addon{
				dashboard.Hotkey{Key: "t", Hint: "Self-test", Tone: dashygotui.ToneGood, Item: diagnosticsSelfTestOpt}, // [1]
				diagnosticsLogsOpt, // [2]
				diagnosticsBackOpt, // [3]
			}},
		}}
}
