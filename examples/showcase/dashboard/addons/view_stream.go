package addons

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [54] Event Stream (hotkey e): a Follow view (Scroll + Tail), like `tail -f`. Rows are appended
// at the bottom as they happen; the view starts at the newest, Up goes back in
// time, and a view scrolled back holds its place while new rows arrive (watch
// the row numbers stay put). End returns to the live bottom. Tail needs Scroll,
// which does the scrolling and the scroll bar. Panel draws the body in a
// bordered panel, with the scroll position in its bottom border. The events are per-dashboard
// state (dashygotui.Local).

// streamState is the view's history, kept per dashboard.
type streamState struct {
	events []string
	total  int // events ever recorded: numbers keep rising after old ones are dropped
}

func init() {
	history := func(c *dashboard.Ctx) *streamState { return dashygotui.Local[streamState](c, "stream") }

	dashboard.Registry.OnTick(func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		if c.Ready() && c.Err() == nil {
			h := history(c)
			h.total++
			h.events = append(h.events, fmt.Sprintf("%s  #%04d  cpu %4.1f%%  mem %4.1f%%",
				now.Format("15:04:05"), h.total, c.State().CPU, c.State().Mem))
			if len(h.events) > 2000 {
				h.events = h.events[len(h.events)-2000:]
			}
		}
		return nil
	})

	dashboard.Registry.Add(dashboard.View{
		ID: "stream", Title: "Event Stream", Mode: "STREAM",
		Entry: &dashboard.Entry{Section: "views", Number: 54, Hotkey: "e"},
		// Follow is Scroll, Tail and a panel in one word. Setting Panel as well
		// gives the panel its own title and subtitle; the scroll position goes in
		// its bottom border.
		Follow: true,
		// The history is capped, so tell the engine how many rows were ever added;
		// a view scrolled back then holds its place even when the list is full.
		Appended: func(c *dashboard.Ctx) int { return history(c).total },
		Panel:    &dashygotui.ViewPanel[dashboard.State]{Title: "Events", Subtitle: "newest at the bottom"},
		Badge:    func(c *dashboard.Ctx) string { return c.Count(len(history(c).events), "events") },
		Body: func(c *dashboard.Ctx, _ dashygotui.Size) []string {
			events := history(c).events
			if len(events) == 0 {
				return []string{c.Theme().Muted.Render("waiting for events...")}
			}
			return append([]string(nil), events...)
		},
	})
}
