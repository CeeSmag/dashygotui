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
// menu. One Add call declares the view and the option that opens it (Entry, with
// its hotkey and footer hint). Items is a static menu: rows are numbered by
// position, and each row says what it does with Opens, Do or Run. A view-local
// Hotkey (t) works only inside the view. OptionsTitle and OptionsSubtitle name
// the menu panel, Tag adds "Mode: ON" to the header, Hints adds a footer hint for
// the view-local hotkey, and Session runs while the view is on screen.

func init() {
	dashboard.Registry.Add(dashboard.View{
		ID: "diagnostics", Title: "Diagnostics", Subtitle: "health of the service",
		Entry: &dashboard.Entry{Section: "views", Number: 50, Hotkey: "d", HotkeyHint: "Diagnostics"},
		Badge: func(c *dashboard.Ctx) string {
			th := c.Theme()
			if c.Err() != nil {
				return th.Red.Render(th.G.Cross + " UNREACHABLE")
			}
			return th.Green.Render(th.G.Bullet + " HEALTHY")
		},
		Tag: func(*dashboard.Ctx) string { return "Mode: ON" },
		Session: func(*dashboard.Ctx) func() {
			log.Print("[INFO] diagnostics opened")
			return func() { log.Print("[INFO] diagnostics closed") }
		},
		OptionsTitle:    "Actions",
		OptionsSubtitle: "Enter runs the highlighted one",
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
		Items: []dashboard.Option{
			{Label: "Run Self-Test", Hotkey: "t", Run: func(c *dashboard.Ctx) tea.Cmd {
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
			{Label: "Open Logs", Opens: "stdaddons.logs"}, // a view can open another view by ID
			{Label: "Back to the Page", Do: func(c *dashboard.Ctx) { c.Back() }},
		},
	})
}
