package addons

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [12] API Token: a masked modal.Edit (Secret) that hands the value to Submit.
// The token itself is never stored or shown, only that one was set.

func init() {
	const setKey = "token.length"

	dashboard.Registry.Add(dashboard.Option{
		Key: "token", Section: "settings", Number: 12, Label: "API Token",
		Badge: func(c *dashboard.Ctx) string {
			var n int
			if ok, _ := c.Store().Get(setKey, &n); !ok {
				return c.Badge("not set")
			}
			return c.Badge(fmt.Sprintf("%d characters", n))
		},
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Edit[dashboard.State]{
				Title:  "API Token",
				Help:   "Paste the token. It is hidden as you type and is never stored by this demo.",
				Secret: true,
				Limit:  64,
				Submit: func(c *dashboard.Ctx, value string) tea.Cmd {
					c.FlashResult("Token accepted.", c.Store().Set(setKey, len(value)))
					return nil
				},
			}
		},
	})
}
