package addons

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [54] Event Stream (hotkey e): a Tail view, like `tail -f`. Rows are appended
// at the bottom as they happen; the view starts at the newest, Up goes back in
// time, and a view scrolled back holds its place while new rows arrive (watch
// the row numbers stay put). End returns to the live bottom. Tail needs Scroll,
// which does the scrolling and the scroll bar. Panel draws the body in a
// bordered panel, with the scroll position in its bottom border. The events live
// in a closure.

func init() {
	var events []string

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			events = append(events, fmt.Sprintf("%s  #%04d  cpu %4.1f%%  mem %4.1f%%",
				now.Format("15:04:05"), len(events)+1, c.State().CPU, c.State().Mem))
			if len(events) > 2000 {
				events = events[len(events)-2000:]
			}
		}
		return nil
	})

	dashboard.Registry.View(dashboard.View{
		ID: "stream", Title: "Event Stream", Mode: "STREAM", Scroll: true, Tail: true,
		// Panel frames the body like a status panel, and puts the scroll
		// position in its bottom border.
		Panel: &dashygotui.ViewPanel[dashboard.State]{Title: "Events", Subtitle: "newest at the bottom"},
		Badge: func(c *dashboard.Ctx) string { return c.Count(len(events), "events") },
		Body: func(c *dashboard.Ctx, _ dashygotui.Size) []string {
			if len(events) == 0 {
				return []string{c.Theme().Muted.Render("waiting for events...")}
			}
			return append([]string(nil), events...)
		},
	})

	dashboard.Registry.Option(dashboard.Option{
		Key: "stream", Section: "views", Number: 54, Label: "Event Stream", Hotkey: "e",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			if err := c.OpenView("stream"); err != nil {
				c.FlashError(err.Error())
			}
			return nil
		},
	})
}
