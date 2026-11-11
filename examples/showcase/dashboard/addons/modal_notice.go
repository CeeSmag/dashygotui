package addons

import (
	"fmt"
	"strings"
	"time"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [30] [31] [32] The three faces of modal.Notice: static text, text recomputed
// on every frame with Content, and a long scrollable report.

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "notice", Section: "standard", Number: 30, Label: "Notice",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Notice[dashboard.State]{
				Title: "About", Badge: c.Badge("v1"),
				Lines: []string{
					"A Notice is a read-only card. Long lines wrap to fit the width.",
					"",
					"It closes on Enter, Esc, Space, b or q.",
				},
			}
		},
	})

	dashboard.Registry.Add(dashboard.Option{
		Key: "notice_live", Section: "standard", Number: 31, Label: "Notice (live content)",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Notice[dashboard.State]{
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
			}
		},
	})

	dashboard.Registry.Add(dashboard.Option{
		Key: "notice_scroll", Section: "standard", Number: 32, Label: "Notice (scrolls)",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			var lines []string
			for i := 1; i <= 60; i++ {
				lines = append(lines, fmt.Sprintf("v1.%02d  %s", i, strings.Repeat("change ", 1+i%5)))
			}
			return &modal.Notice[dashboard.State]{Title: "Changelog", Badge: c.Count(len(lines), "entries"), Lines: lines}
		},
	})
}
