package addons

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [10] Storage Limit (hotkey s). This is the raw modal API on show, to demonstrate
// Edit's Validate, Save and OnSaved. For an ordinary number setting do not write
// this: use stdaddons.Number (see [16] Log Retention, which is five lines).
//
// modal.Edit with Validate and a background
// Save. A failed save or validation shows inline and keeps the editor open. On
// success OnSaved chains a follow-up task, here pretending to reload the
// service with the new limit, the way an application restarts a server after
// saving one of its settings.

func init() {
	const limitKey = "storage.limit_gb"
	const defaultGB = 50

	limit := func(c *dashboard.Ctx) int {
		var gb int
		if ok, _ := c.Store().Get(limitKey, &gb); !ok {
			return defaultGB
		}
		return gb
	}

	dashboard.Registry.Add(dashboard.Option{
		Key: "storage", Section: "settings", Number: 10, Label: "Storage Limit", Hotkey: "s",
		Badge: func(c *dashboard.Ctx) string { return c.Badge(fmt.Sprintf("%d GB", limit(c))) },
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			st := c.Store()
			return &modal.Edit[dashboard.State]{
				Title:   "Storage Limit",
				Help:    "How much disk space the service may use, in gigabytes (1 to 1000).",
				Current: fmt.Sprintf("%d GB", limit(c)),
				Pending: "Saving the limit...",
				Validate: func(v string) error {
					n, err := strconv.Atoi(v)
					if err != nil || n < 1 || n > 1000 {
						return fmt.Errorf("%q is not a number from 1 to 1000", v)
					}
					return nil
				},
				Save: func(_ context.Context, v string) (string, error) {
					n, _ := strconv.Atoi(v) // already validated
					if err := st.Set(limitKey, n); err != nil {
						return "", err
					}
					log.Printf("[INFO] storage limit set to %d GB", n)
					return fmt.Sprintf("Storage limit set to %d GB.", n), nil
				},
				OnSaved: func(c *dashboard.Ctx, saved string) tea.Cmd {
					return c.RunTask("storage.reload", "Reloading the service...", func(ctx context.Context) (string, error) {
						select {
						case <-time.After(400 * time.Millisecond):
						case <-ctx.Done():
							return "", ctx.Err()
						}
						log.Print("[INFO] service reloaded with the new storage limit")
						return saved + " Service reloaded.", nil
					})
				},
			}
		},
	})
}
