package addons

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [36] [37] [38] The input-style pop-ups on their own: a Choice with badges and
// details, a Slider with a unit, and an Edit with validation.

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "choice", Section: "standard", Number: 36, Label: "Choice",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			th := c.Theme()
			items := []modal.Item{
				{Label: "Low", Badge: th.Green.Render("[ok]"), Detail: "digest once a day"},
				{Label: "Normal", Badge: th.Amber.Render("[default]"), Detail: "digest every hour"},
				{Label: "Urgent", Badge: th.Red.Render("[loud]"), Detail: "every event, immediately"},
			}
			return &modal.Choice[dashboard.State]{
				Title: "Alert level", Help: "Up and down move, a digit jumps, Enter selects, Esc cancels.",
				Items: items, Cursor: 1,
				OnSelect: func(c *dashboard.Ctx, i int) tea.Cmd { c.Flash("Level: " + items[i].Label); return nil },
			}
		},
	})

	dashboard.Registry.Add(dashboard.Option{
		Key: "slider", Section: "standard", Number: 37, Label: "Slider",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Slider[dashboard.State]{
				Title: "Retention", Help: "Left and right adjust, Home and End jump, Esc restores.",
				Unit: "days", Min: 1, Max: 30, Value: 14,
				OnDone: func(c *dashboard.Ctx, v int) tea.Cmd { c.Flash(fmt.Sprintf("Keeping %d days", v)); return nil },
			}
		},
	})

	dashboard.Registry.Add(dashboard.Option{
		Key: "edit", Section: "standard", Number: 38, Label: "Edit (with validation)",
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Edit[dashboard.State]{
				Title: "Contact", Help: "Where alerts are sent. Press Enter on an empty field to cancel.",
				Validate: func(v string) error {
					if !strings.Contains(v, "@") {
						return fmt.Errorf("%q does not look like an email address", v)
					}
					return nil
				},
				Submit: func(c *dashboard.Ctx, v string) tea.Cmd { c.Flash("Alerts go to " + v); return nil },
			}
		},
	})
}
