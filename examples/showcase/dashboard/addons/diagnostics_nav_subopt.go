package addons

import "github.com/CeeSmag/dashygotui/examples/showcase/dashboard"

// Two more rows of the Diagnostics subpage: one opens another view by its ID,
// one goes back.

var diagnosticsLogsOpt = dashboard.Option{Label: "Open Logs", Opens: "stdaddons.logs"}

var diagnosticsBackOpt = dashboard.Option{Label: "Back to the Page", Do: func(c *dashboard.Ctx) { c.Back() }}
