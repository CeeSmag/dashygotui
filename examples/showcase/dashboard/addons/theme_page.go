package addons

import "github.com/CeeSmag/dashygotui/examples/showcase/dashboard"

// The Theme tab's badge: the palette in use.

func themeBadge(c *dashboard.Ctx) string { return c.Theme().Muted.Render(c.PaletteID()) }
