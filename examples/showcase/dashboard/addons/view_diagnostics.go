package addons

import (
	"context"
	"fmt"
	"log"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// [50] Diagnostics (hotkey d): a full-screen view with a Status panel above a
// menu of Options. The view has its own options with their own numbers, and a
// view-local Hotkey (t) that works only inside it. The view and the option that
// opens it are one feature, so one file registers both. OptionsTitle and
// OptionsSubtitle name the menu panel, Header adds a "Mode: ON" segment to the
// header and Hints adds a footer hint for the view-local hotkey.

func init() {
	dashboard.Registry.View(dashboard.View{
		ID: "diagnostics", Title: "Diagnostics", Subtitle: "health of the service", Mode: "DIAG",
		Badge: func(c *dashboard.Ctx) string {
			th := c.Theme()
			if c.Err() != nil {
				return th.Red.Render(th.G.Cross + " UNREACHABLE")
			}
			return th.Green.Render(th.G.Bullet + " HEALTHY")
		},
		OptionsTitle:    "Actions",
		OptionsSubtitle: "Enter runs the highlighted one",
		Header: func(c *dashboard.Ctx) []panel.Segment {
			return []panel.Segment{{Text: c.Theme().Green.Render("Mode: ON"), Prio: 85}}
		},
		Hints: func(*dashboard.Ctx) []dashygotui.Hint {
			return []dashygotui.Hint{{Key: "t", Label: "Self-test", Tone: dashygotui.ToneGood, Prio: 58}}
		},
		Status: func(c *dashboard.Ctx, _ dashygotui.Size) []string {
			th, s := c.Theme(), c.State()
			rows := []string{
				panel.Row(th, "Uptime", s.Uptime.Truncate(time.Second).String()),
				panel.Row(th, "Requests", fmt.Sprintf("%d served", s.Requests)),
				panel.Row(th, "Pushed", fmt.Sprintf("%d events (Dashboard.Notify)", s.Events)),
				panel.Row(th, "Traffic", text.Rate(s.NetIn)+" in, "+text.Rate(s.NetOut)+" out"),
			}
			if err := c.Err(); err != nil {
				rows = append(rows, panel.Row(th, "Error", th.Red.Render(err.Error())))
			}
			return rows
		},
		Options: func(*dashboard.Ctx) []dashboard.Option {
			return []dashboard.Option{
				{Number: 1, Label: "Run Self-Test", Hotkey: "t", Run: func(c *dashboard.Ctx) tea.Cmd {
					return c.RunTask("selftest", "Testing...", func(ctx context.Context) (string, error) {
						select {
						case <-time.After(600 * time.Millisecond):
						case <-ctx.Done():
							return "", ctx.Err()
						}
						log.Print("[INFO] self-test passed")
						return "Self-test passed: 12 of 12 checks.", nil
					})
				}},
				{Number: 2, Label: "Open Logs", Run: func(c *dashboard.Ctx) tea.Cmd {
					// A view can open another view by ID.
					if err := c.OpenView("stdaddons.logs"); err != nil {
						c.FlashError(err.Error())
					}
					return nil
				}},
				{Number: 3, Label: "Back to the Page", Run: func(c *dashboard.Ctx) tea.Cmd { c.Back(); return nil }},
			}
		},
	})

	dashboard.Registry.Option(dashboard.Option{
		Key: "diagnostics", Section: "views", Number: 50, Label: "Diagnostics", Hotkey: "d",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			if err := c.OpenView("diagnostics"); err != nil {
				c.FlashError(err.Error())
			}
			return nil
		},
	})
}
