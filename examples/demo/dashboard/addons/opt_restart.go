package addons

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [2] Restart Service: a yes/no confirmation in front of a background task.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "restart", Section: "actions", Number: 2, Label: "Restart Service",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Confirm[dashboard.State]{
				Title:  "Restart the service?",
				Lines:  []string{"Connections are dropped for a moment while it restarts."},
				Accept: "Restart",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd {
					return c.RunTask("restart", "Restarting…", func(ctx context.Context) (string, error) {
						select {
						case <-time.After(800 * time.Millisecond):
						case <-ctx.Done():
							return "", ctx.Err()
						}
						return "Service restarted.", nil
					})
				},
			})
			return nil
		},
	})
}
