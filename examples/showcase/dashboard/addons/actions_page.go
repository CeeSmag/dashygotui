package addons

import "github.com/CeeSmag/dashygotui/examples/showcase/dashboard"

// The Settings panel's badge, which replaces the default count.

func persistedBadge(c *dashboard.Ctx) string { return c.Theme().Muted.Render("[persisted]") }
