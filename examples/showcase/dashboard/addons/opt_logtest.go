package addons

import (
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [73] Write Log Entries: writes through the standard logger and through slog,
// at every level, plus a long wrapping line and one containing terminal escape
// sequences, which the log view shows harmlessly. Open [52] Logs to see them.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "logtest", Section: "hooks", Number: 73, Label: "Write Log Entries",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			log.Print("[DEBUG] standard logger, debug")
			log.Print("[INFO] standard logger, info")
			log.Print("[WARN] standard logger, warning")
			log.Print("[ERROR] standard logger, error")
			dashboard.Slog.Debug("slog debug", "component", "cache")
			dashboard.Slog.Info("slog info", "user", "ada", "ms", 12)
			dashboard.Slog.Warn("slog warning", "queue", "mail", "depth", 940)
			dashboard.Slog.Error("slog error", "err", "connection reset by peer")
			log.Print("[INFO] a long line that wraps onto several rows so you can see how continuation lines are indented under the message text, not under the time")
			log.Print("[WARN] control characters \x1b[2J\x1b[31m are replaced, never obeyed")
			c.FlashResult("Wrote 10 log entries. Press g to read them.", nil)
			return nil
		},
	})
}
