package addons

import (
	"context"
	"log"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
)

// [1] Run Backup: a background task that records when it last finished.

func init() {
	const lastKey = "backup.last"
	dashboard.Registry.Add(dashboard.Option{
		Key: "backup", Section: "actions", Number: 1, Label: "Run Backup",
		Badge: func(c *dashboard.Ctx) string {
			var last time.Time
			if ok, _ := c.Store().Get(lastKey, &last); !ok {
				return c.Badge("never")
			}
			return c.Badge("last " + last.Format("15:04"))
		},
		Run: func(c *dashboard.Ctx) tea.Cmd {
			st := c.Store() // take what the task needs now; the Context is not for other goroutines
			return c.RunTask("backup", "Backing up...", func(ctx context.Context) (string, error) {
				select {
				case <-time.After(1200 * time.Millisecond):
				case <-ctx.Done():
					return "", ctx.Err()
				}
				if err := st.Set(lastKey, time.Now()); err != nil {
					return "", err
				}
				log.Print("[INFO] backup finished")
				return "Backup finished.", nil
			})
		},
	})
}
