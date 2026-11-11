package addons

import (
	"fmt"
	"log"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/panel"
)

// [51] Processes (hotkey p): a view that draws its whole body itself (Body) and
// handles its own keys (Keys): select, sort and kill. Its selection, sort order
// and the killed processes live in closure variables, not package state.

func init() {
	sel, byMem := 0, false
	killed := map[int]bool{}

	rows := func(c *dashboard.Ctx) []dashboard.Process {
		var out []dashboard.Process
		for _, p := range c.State().Processes {
			if !killed[p.PID] {
				out = append(out, p)
			}
		}
		slices.SortFunc(out, func(a, b dashboard.Process) int {
			if byMem {
				return b.MemMB - a.MemMB
			}
			switch {
			case a.CPU > b.CPU:
				return -1
			case a.CPU < b.CPU:
				return 1
			}
			return a.PID - b.PID
		})
		return out
	}

	dashboard.Registry.View(dashboard.View{
		ID: "processes", Title: "Processes", Mode: "PROCS",
		Badge: func(c *dashboard.Ctx) string { return c.Count(len(rows(c)), "running") },
		Body: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			th := c.Theme()
			procs := rows(c)
			sel = min(max(sel, 0), max(len(procs)-1, 0))
			inner := max(size.Rows-3, 1) // borders and the header row
			start := min(max(sel-inner/2, 0), max(len(procs)-inner, 0))

			sortLabel := "cpu"
			if byMem {
				sortLabel = "mem"
			}
			body := []string{th.Muted.Render(fmt.Sprintf("  %-6s %-16s %7s %8s", "PID", "NAME", "CPU %", "MEM MB"))}
			for i := start; i < min(start+inner, len(procs)); i++ {
				p := procs[i]
				line := fmt.Sprintf("%-6d %-16s %6.1f%% %7d", p.PID, p.Name, p.CPU, p.MemMB)
				if i == sel {
					body = append(body, th.Key.Render(th.G.Pointer)+" "+th.White.Render(line))
				} else {
					body = append(body, "  "+line)
				}
			}
			sub := fmt.Sprintf("%s select  c cpu  m mem  k kill  Enter details  sorted by %s", th.G.UpDown, sortLabel)
			return panel.Panel(th, "Processes", c.Count(len(procs), "running"), sub, size.Cols, size.Rows, body)
		},
		Keys: func(c *dashboard.Ctx, msg tea.KeyPressMsg) (tea.Cmd, bool) {
			procs := rows(c)
			switch strings.ToLower(msg.String()) {
			case "up":
				sel--
			case "down", "j":
				sel++
			case "home":
				sel = 0
			case "end":
				sel = len(procs) - 1
			case "c":
				byMem = false
			case "m":
				byMem = true
			case "k": // the view claims k: it kills instead of moving up
				if sel >= 0 && sel < len(procs) {
					p := procs[sel]
					killed[p.PID] = true
					log.Printf("[WARN] killed process %s (pid %d)", p.Name, p.PID)
					c.FlashResult("Killed "+p.Name+".", nil)
				}
			case "enter":
				if sel >= 0 && sel < len(procs) {
					p := procs[sel]
					c.OpenModal(&modal.Notice[dashboard.State]{
						Title: p.Name, Badge: c.Badge(fmt.Sprint("pid ", p.PID)),
						Lines: []string{fmt.Sprintf("CPU     %.1f%%", p.CPU), fmt.Sprintf("Memory  %d MB", p.MemMB)},
					})
				}
			default:
				return nil, false // everything else (Esc, q, ...) is left to the dashboard
			}
			sel = min(max(sel, 0), max(len(procs)-1, 0))
			return nil, true
		},
	})

	dashboard.Registry.Option(dashboard.Option{
		Key: "processes", Section: "views", Number: 51, Label: "Processes", Hotkey: "p",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			if err := c.OpenView("processes"); err != nil {
				c.FlashError(err.Error())
			}
			return nil
		},
	})
}
