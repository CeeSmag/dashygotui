package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [4] Open Web UI (hotkey Ctrl+O): an option whose hotkey is a key combination.
// HotkeyDisplay writes it as "Ctrl+O" after the label instead of "ctrl+o", and
// HotkeyTone draws it in the theme's info colour. The key itself is still
// Hotkey, the name Bubble Tea reports.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "web", Section: "ops", Number: 4, Label: "Open Web UI",
		Hotkey: "ctrl+o", HotkeyDisplay: "Ctrl+O", HotkeyTone: dashygotui.ToneInfo,
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.Flash("Would open http://localhost:8080 in your browser")
			return nil
		},
	})
}
