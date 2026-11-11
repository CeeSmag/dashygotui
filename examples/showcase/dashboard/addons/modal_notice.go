package addons

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [30] [31] [32] The three faces of modal.Notice: static text, text recomputed
// on every frame with Content, and a long scrollable report.

func init() {
	dashboard.Registry.Option(dashboard.Option{
		Key: "notice", Section: "standard", Number: 30, Label: "Notice",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Notice[dashboard.State]{
				Title: "About", Badge: c.Badge("v1"),
				Lines: []string{
					"A Notice is a read-only card. Long lines wrap to fit the width.",
					"",
					"It closes on Enter, Esc, Space, b or q.",
				},
			})
			return nil
		},
	})

	dashboard.Registry.Option(dashboard.Option{
		Key: "notice_live", Section: "standard", Number: 31, Label: "Notice (live content)",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			c.OpenModal(&modal.Notice[dashboard.State]{
				Width: 60,
				// Content runs on every frame, so this card updates while it is open.
				Content: func(c *dashboard.Ctx, inner int) (string, string, []string) {
					s := c.State()
					return "Live Status", c.Badge(fmt.Sprintf("%d wide", inner)), []string{
						"Time      " + c.Now().Format("15:04:05"),
						"Uptime    " + s.Uptime.Truncate(time.Second).String(),
						fmt.Sprintf("CPU       %.1f%%", s.CPU),
						fmt.Sprintf("Requests  %d", s.Requests),
					}
				},
			})
			return nil
		},
	})

	dashboard.Registry.Option(dashboard.Option{
		Key: "notice_scroll", Section: "standard", Number: 32, Label: "Notice (scrolls)",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			var lines []string
			for i := 1; i <= 60; i++ {
				lines = append(lines, fmt.Sprintf("v1.%02d  %s", i, strings.Repeat("change ", 1+i%5)))
			}
			c.OpenModal(&modal.Notice[dashboard.State]{Title: "Changelog", Badge: c.Count(len(lines), "entries"), Lines: lines})
			return nil
		},
	})
}
