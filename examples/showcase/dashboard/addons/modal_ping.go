package addons

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// [41] Ping Host: a custom pop-up that is also a dashygotui.ResultModal. It
// starts a background task with RunTask and receives that task's result in
// OnResult, while it stays open, instead of the footer showing it.

type pingModal struct {
	host    string
	tag     string // identifies this pop-up's task among all tasks
	running bool
	result  string
}

// HandleKey pings on Enter and closes on Esc.
func (m *pingModal) HandleKey(c *dashboard.Ctx, msg tea.KeyPressMsg) tea.Cmd {
	switch strings.ToLower(msg.String()) {
	case "enter":
		if m.running {
			return nil
		}
		m.running, m.result = true, ""
		return c.RunTask(m.tag, "", func(ctx context.Context) (string, error) {
			select {
			case <-time.After(600 * time.Millisecond):
			case <-ctx.Done():
				return "", ctx.Err()
			}
			return fmt.Sprintf("reply in %d ms", 8+rand.IntN(40)), nil
		})
	case "esc", "q":
		c.CloseModal()
	}
	return nil
}

// OnResult takes the result of this pop-up's task and nobody else's.
func (m *pingModal) OnResult(_ *dashboard.Ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
	if res.Tag != m.tag {
		return false, nil
	}
	m.running = false
	m.result = res.Text
	if res.Err != nil {
		m.result = "failed: " + res.Err.Error()
	}
	return true, nil
}

// Render draws the box.
func (m *pingModal) Render(c *dashboard.Ctx, size dashygotui.Size) []string {
	th := c.Theme()
	status := th.Muted.Render("press Enter to ping")
	switch {
	case m.running:
		status = th.Amber.Render("pinging" + strings.Repeat(".", 1+c.Ticks()%3))
	case m.result != "":
		status = th.Green.Render(th.G.Check + " " + m.result)
	}
	body := []string{panel.Row(th, "Host", m.host), panel.Row(th, "Status", status), "",
		th.Muted.Render("Enter") + " ping    " + th.Muted.Render("Esc") + " close"}
	return panel.Panel(th, "Ping", "", "", min(50, max(size.Cols, 1)), len(body)+2, body)
}

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "ping", Section: "custom", Number: 41, Label: "Ping Host",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			m := &pingModal{host: "example.local"}
			m.tag = fmt.Sprintf("ping.%p", m)
			return m
		},
	})
}
