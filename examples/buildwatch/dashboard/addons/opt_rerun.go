package addons

import (
	"fmt"

	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
)

// Re-run the failed builds: an action with nothing to return, so Do.

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "rerun", Section: "actions", Number: 2, Label: "Re-run Failed Builds",
		Badge: func(c *dashboard.Ctx) string { return c.Count(c.State().Failed(), "failed") },
		Do: func(c *dashboard.Ctx) {
			c.Flash(fmt.Sprintf("Re-running %d failed build(s)", c.State().Failed()))
		},
	})
}
