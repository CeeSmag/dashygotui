package addons

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [3] Export Report: a task whose result is handled by a Registry.OnResult
// hook, which opens the report in a scrollable modal.Notice instead of the
// default footer message.

func init() {
	const tag = "export"

	dashboard.Registry.Add(dashboard.Option{
		Key: "export", Section: "ops", Number: 3, Label: "Export Report",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			s := c.State() // a copy for the task to read
			return c.RunTask(tag, "Building the report...", func(ctx context.Context) (string, error) {
				select {
				case <-time.After(700 * time.Millisecond):
				case <-ctx.Done():
					return "", ctx.Err()
				}
				var b strings.Builder
				fmt.Fprintf(&b, "Report generated at %s\n\n", time.Now().Format("15:04:05"))
				fmt.Fprintf(&b, "CPU       %5.1f%%\nMemory    %5.1f%%\nDisk      %5.1f%%\n", s.CPU, s.Mem, s.Disk)
				fmt.Fprintf(&b, "Requests  %d served\nUptime    %s\n\nTop processes:\n", s.Requests, s.Uptime.Truncate(time.Second))
				for i, p := range s.Processes {
					if i == 5 {
						break
					}
					fmt.Fprintf(&b, "  %-14s %5.1f%% %5d MB\n", p.Name, p.CPU, p.MemMB)
				}
				return b.String(), nil
			})
		},
	})

	dashboard.Registry.OnResult(func(c *dashboard.Ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
		if res.Tag != tag || res.Err != nil {
			return false, nil // not ours, or failed: let the default footer message show it
		}
		c.OpenModal(&modal.Notice[dashboard.State]{
			Title: "Report", Badge: c.Badge("exported"), Lines: strings.Split(strings.TrimRight(res.Text, "\n"), "\n"),
		})
		return true, nil
	})
}
