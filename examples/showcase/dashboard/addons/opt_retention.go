package addons

import (
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [16] Log Retention: a whole-number setting. stdaddons.Number gives the badge,
// the pop-up with range validation, saving, and the confirmation; OnChange is
// where an application would apply the new value (restart something, say).

func init() {
	dashboard.Registry.Add(stdaddons.Number[dashboard.State]{
		Key: "logs.retention_days", Section: "settings", Number: 16, Label: "Log Retention",
		Unit: "days", Min: 1, Max: 365, Default: 30,
		Help: "How long logs are kept before they are deleted.",
		OnChange: func(_ *dashboard.Ctx, days int) tea.Cmd {
			log.Printf("[INFO] log retention is now %d days", days)
			return nil
		},
	})
}
