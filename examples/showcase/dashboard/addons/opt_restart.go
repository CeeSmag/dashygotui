package addons

import (
	"context"
	"log"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [2] Restart Service: a modal.Confirm with both OnAccept and OnCancel in
// front of a background task.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "restart", Section: "ops", Number: 2, Label: "Restart Service",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Confirm[dashboard.State]{
				Title:  "Restart the service?",
				Lines:  []string{"Connections are dropped for a moment while it restarts."},
				Accept: "Restart",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd {
					return c.RunTask("restart", "Restarting...", func(ctx context.Context) (string, error) {
						select {
						case <-time.After(800 * time.Millisecond):
						case <-ctx.Done():
							return "", ctx.Err()
						}
						log.Print("[WARN] service restarted by the operator")
						return "Service restarted.", nil
					})
				},
				OnCancel: func(c *dashboard.Ctx) tea.Cmd {
					c.Flash("Restart cancelled")
					return nil
				},
			})
			return nil
		},
	})
}
