package addons

import (
	"log"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Write Log Entries: writes through the standard logger and through slog,
// at every level, per-subsystem entries in their own categories, plus a long wrapping line and one containing terminal escape
// sequences, which the log view shows harmlessly. Open Logs (g) to see them.

var logtestOption = dashboard.Option{
	Label: "Write Log Entries",
	Do: func(c *dashboard.Ctx) {
		slog, s := c.State().Logger(""), c.State()
		log.Print("[DEBUG] standard logger, debug")
		log.Print("[INFO] standard logger, info")
		log.Print("[WARN] standard logger, warning")
		log.Print("[ERROR] standard logger, error")
		slog.Debug("slog debug", "component", "cache")
		slog.Info("slog info", "user", "ada", "ms", 12)
		slog.Warn("slog warning", "queue", "mail", "depth", 940)
		slog.Error("slog error", "err", "connection reset by peer")
		log.Print("[INFO] a long line that wraps onto several rows so you can see how continuation lines are indented under the message text, not under the time")
		//dashygotui:allow A6 this line exists to prove the log view neutralises escape sequences
		log.Print("[WARN] control characters \x1b[2J\x1b[31m are replaced, never obeyed")
		// Per-subsystem loggers file their entries under a category tab.
		s.Logger("net").Info("listening", "addr", ":8080")
		s.Logger("net").Warn("slow upstream", "ms", 1800)
		s.Logger("disk").Info("volume mounted", "path", "/data")
		s.Logger("disk").Error("write failed", "err", "no space left on device")
		s.Logger("auth").Info("login", "user", "ada")
		s.Logger("auth").Warn("failed login", "user", "mallory", "attempts", 3)
		c.FlashResult("Wrote 16 log entries. Press g to read them, then c to switch tabs.", nil)
	},
}
