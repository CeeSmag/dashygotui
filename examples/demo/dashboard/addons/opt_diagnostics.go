package addons

import (
	"context"
	"fmt"
	"log"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// [3] Diagnostics: a full-screen view and the option that opens it, declared
// together. Entry registers the option; Items is the menu inside the view.

func init() {
	dashboard.Registry.Add(dashboard.View{
		ID: "diagnostics", Title: "Diagnostics", Subtitle: "health of the demo service",
		Entry: &dashboard.Entry{Section: "actions", Number: 3},
		Badge: func(c *dashboard.Ctx) string { return c.Theme().Green.Render(c.Theme().G.Bullet + " HEALTHY") },
		Status: func(c *dashboard.Ctx, _ dashygotui.Size) []string {
			th, s := c.Theme(), c.State()
			return []string{
				panel.Row(th, "Uptime", s.Uptime.Truncate(time.Second).String()),
				panel.Row(th, "Requests", fmt.Sprintf("%d served", s.Requests)),
				panel.Row(th, "Traffic", text.Rate(s.NetIn)+" in, "+text.Rate(s.NetOut)+" out"),
			}
		},
		Items: []dashboard.Option{
			{Label: "Run Self-Test", Run: func(c *dashboard.Ctx) tea.Cmd {
				return c.RunTask("selftest", "Testing...", func(ctx context.Context) (string, error) {
					select {
					case <-time.After(600 * time.Millisecond):
					case <-ctx.Done():
						return "", ctx.Err()
					}
					log.Print("[INFO] self-test passed")
					return "Self-test passed: 12 of 12 checks.", nil
				})
			}},
		},
	})
}
