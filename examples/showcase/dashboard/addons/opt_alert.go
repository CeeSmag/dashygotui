package addons

import (
	"fmt"
	"log"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [70] Simulate Alert: a command that produces a message of this addon's own
// type, handled by a Registry.OnMsg hook. This is how an addon reacts to events
// that did not come from a key press or a task.

type alertMsg struct {
	level string
	text  string
}

func init() {
	dashboard.Registry.OnMsg(func(c *dashboard.Ctx, msg tea.Msg) tea.Cmd {
		a, ok := msg.(alertMsg)
		if !ok {
			return nil // not ours
		}
		alerts := dashygotui.Local[int](c, "alerts")
		*alerts++
		log.Printf("[%s] alert #%d: %s", a.level, *alerts, a.text)
		if a.level == "ERROR" {
			c.FlashError(fmt.Sprintf("Alert #%d: %s", *alerts, a.text))
		} else {
			c.Flash(fmt.Sprintf("Alert #%d: %s", *alerts, a.text))
		}
		return func() tea.Msg { return nil } // a non-nil command claims the message
	})

	dashboard.Registry.Add(dashboard.Option{
		Key: "alert", Section: "hooks", Number: 70, Label: "Simulate Alert",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			level, text := "WARN", "disk usage above 80%"
			if c.Ticks()%2 == 1 {
				level, text = "ERROR", "replica lag exceeds 30 seconds"
			}
			return func() tea.Msg { return alertMsg{level: level, text: text} }
		},
	})
}
