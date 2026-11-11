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
// and the killed processes are per-dashboard state (dashygotui.Local), not package state.

// procState is the view's selection, sort order and the processes killed, kept
// per dashboard.
type procState struct {
	sel    int
	byMem  bool
	killed map[int]bool
}

func init() {
	state := func(c *dashboard.Ctx) *procState {
		return dashygotui.LocalWith(c, "processes", func() *procState { return &procState{killed: map[int]bool{}} })
	}

	rows := func(c *dashboard.Ctx) []dashboard.Process {
		st := state(c)
		var out []dashboard.Process
		for _, p := range c.State().Processes {
			if !st.killed[p.PID] {
				out = append(out, p)
			}
		}
		slices.SortFunc(out, func(a, b dashboard.Process) int {
			if st.byMem {
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

	dashboard.Registry.Add(dashboard.View{
		ID: "processes", Title: "Processes", Mode: "PROCS",
		Entry: &dashboard.Entry{Section: "views", Number: 51, Hotkey: "p"},
		Badge: func(c *dashboard.Ctx) string { return c.Count(len(rows(c)), "running") },
		Body: func(c *dashboard.Ctx, size dashygotui.Size) []string {
			st := state(c)
			th := c.Theme()
			procs := rows(c)
			st.sel = min(max(st.sel, 0), max(len(procs)-1, 0))
			inner := max(size.Rows-3, 1) // borders and the header row
			start := min(max(st.sel-inner/2, 0), max(len(procs)-inner, 0))

			sortLabel := "cpu"
			if st.byMem {
				sortLabel = "mem"
			}
			body := []string{th.Muted.Render(fmt.Sprintf("  %-6s %-16s %7s %8s", "PID", "NAME", "CPU %", "MEM MB"))}
			for i := start; i < min(start+inner, len(procs)); i++ {
				p := procs[i]
				line := fmt.Sprintf("%-6d %-16s %6.1f%% %7d", p.PID, p.Name, p.CPU, p.MemMB)
				if i == st.sel {
					body = append(body, th.Key.Render(th.G.Pointer)+" "+th.White.Render(line))
				} else {
					body = append(body, "  "+line)
				}
			}
			sub := fmt.Sprintf("%s select  c cpu  m mem  k kill  Enter details  sorted by %s", th.G.UpDown, sortLabel)
			return panel.Panel(th, "Processes", c.Count(len(procs), "running"), sub, size.Cols, size.Rows, body)
		},
		Keys: func(c *dashboard.Ctx, msg tea.KeyPressMsg) (tea.Cmd, bool) {
			st := state(c)
			procs := rows(c)
			switch strings.ToLower(msg.String()) {
			case "up":
				st.sel--
			case "down", "j":
				st.sel++
			case "home":
				st.sel = 0
			case "end":
				st.sel = len(procs) - 1
			case "c":
				st.byMem = false
			case "m":
				st.byMem = true
			case "k": // the view claims k: it kills instead of moving up
				if st.sel >= 0 && st.sel < len(procs) {
					p := procs[st.sel]
					st.killed[p.PID] = true
					log.Printf("[WARN] killed process %s (pid %d)", p.Name, p.PID)
					c.FlashResult("Killed "+p.Name+".", nil)
				}
			case "enter":
				if st.sel >= 0 && st.sel < len(procs) {
					p := procs[st.sel]
					c.OpenModal(&modal.Notice[dashboard.State]{
						Title: p.Name, Badge: c.Badge(fmt.Sprint("pid ", p.PID)),
						Lines: []string{fmt.Sprintf("CPU     %.1f%%", p.CPU), fmt.Sprintf("Memory  %d MB", p.MemMB)},
					})
				}
			default:
				return nil, false // everything else (Esc, q, ...) is left to the dashboard
			}
			st.sel = min(max(st.sel, 0), max(len(procs)-1, 0))
			return nil, true
		},
	})
}
