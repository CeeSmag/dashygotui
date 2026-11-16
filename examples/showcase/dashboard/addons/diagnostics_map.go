package addons

import "github.com/CeeSmag/dashygotui/examples/showcase/dashboard"

// The Diagnostics window: what is inside it. The row that opens it is a plain
// option (diagnostics_opt.go), and what it does while open is diagnostics_page.go.
// Its title is the option's label; its rows are numbered from 1.
func diagnosticsPage() dashboard.Page {
	return dashboard.Page{ID: "diagnostics", Row: diagnosticsOpt, Badge: diagnosticsBadge, Session: diagnosticsSession,
		Panels: []dashboard.Panel{
			{Widget: diagnosticsHealthWidget},
			{Title: "Actions", Subtitle: "Enter runs the highlighted one", Items: []dashboard.Addon{
				diagnosticsSelfTestOpt, diagnosticsLogsOpt, diagnosticsBackOpt, // [1] to [3]
			}},
		}}
}
