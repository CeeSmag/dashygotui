package addons

import (
	"fmt"
	"math/rand/v2"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
)

// [40] Dice: a custom pop-up that implements dashygotui.Modal itself. All of its
// state lives in the modal value, never in a package variable.

type diceModal struct{ history []int }

// HandleKey rolls on Space or Enter, clears on r, and closes on Esc or q.
func (d *diceModal) HandleKey(c *dashboard.Ctx, msg tea.KeyPressMsg) tea.Cmd {
	switch strings.ToLower(msg.String()) {
	case "space", " ", "enter":
		d.history = append(d.history, rand.IntN(6)+1)
		if len(d.history) > 10 {
			d.history = d.history[1:]
		}
	case "r":
		d.history = nil
	case "esc", "q":
		c.CloseModal()
	}
	return nil
}

// Render draws the box only; the dashboard centres and clips it.
func (d *diceModal) Render(c *dashboard.Ctx, size dashygotui.Size) []string {
	th := c.Theme()
	last := th.Muted.Render("press space")
	if n := len(d.history); n > 0 {
		last = th.White.Render(fmt.Sprintf("[ %d ]", d.history[n-1]))
	}
	rolls := th.Muted.Render("no rolls yet")
	if len(d.history) > 0 {
		parts := make([]string, len(d.history))
		for i, h := range d.history {
			parts[i] = fmt.Sprint(h)
		}
		rolls = strings.Join(parts, " ")
	}
	body := []string{
		"Roll", "", "  " + last, "", panel.Row(th, "History", rolls), "",
		th.Muted.Render("Space") + " roll    " + th.Muted.Render("r") + " clear    " + th.Muted.Render("Esc") + " close",
	}
	return panel.Panel(th, "Dice", "", "", min(48, max(size.Cols, 1)), len(body)+2, body)
}

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "dice", Section: "custom", Number: 40, Label: "Dice Roller",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &diceModal{}
		},
	})
}
