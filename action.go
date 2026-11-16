package dashygotui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Addon is anything a panel's Items take in the map ([Panel].Items). An
// [Option], a [View] and a [Page] are rows; so is any type of yours that returns
// options from Addons, along with any view, shortcut or hook they need: a
// "setting" that is an option plus a pop-up, say. The ready-made addons in
// package stdaddons are written this way.
//
// An addon that returns other addons is expanded in place, so what it returns is
// placed and checked like anything else. [Option], [Shortcut], [View], [Widget],
// [Hook] and [Page] return themselves.
type Addon[S any] interface {
	// Addons returns what the addon is made of.
	Addons() []Addon[S]
}

// Addons implements [Addon].
func (o Option[S]) Addons() []Addon[S] { return []Addon[S]{o} }

// Addons implements [Addon].
func (s Shortcut[S]) Addons() []Addon[S] { return []Addon[S]{s} }

// Addons implements [Addon].
func (v View[S]) Addons() []Addon[S] { return []Addon[S]{v} }

// Addons implements [Addon].
func (w Widget[S]) Addons() []Addon[S] { return []Addon[S]{w} }

// Addons implements [Addon].
func (h Hook[S]) Addons() []Addon[S] { return []Addon[S]{h} }

// Hook is an addon that reacts to the dashboard's events rather than drawing
// anything: an addon that samples data on every tick, handles the messages of
// its own commands, or claims the results of its own tasks. Set at least one
// field.
type Hook[S any] struct {
	// Tick is called on every refresh tick, for addons that sample data over
	// time. It runs on the update loop and must not block.
	Tick func(*Context[S], time.Time) tea.Cmd
	// Msg is called with every message the dashboard does not recognise, so an
	// addon can handle the messages of its own commands. Every hook sees every
	// such message; return nil to ignore one.
	Msg func(*Context[S], tea.Msg) tea.Cmd
	// Result sees the result of every background task. Return handled == true
	// to stop the other result hooks and the default footer message.
	Result func(*Context[S], TaskResult) (handled bool, cmd tea.Cmd)
}

// Invalid is an addon that records a mistake instead of placing anything. A
// composite addon returns it from Addons when its own settings make no sense (a
// number range that is empty, a list with nothing to pick), so the mistake is
// reported at startup with its place in the map, like any other.
type Invalid[S any] struct {
	// Reason says what is wrong, in a lower-case sentence fragment.
	Reason string
}

// Addons implements [Addon].
func (i Invalid[S]) Addons() []Addon[S] { return []Addon[S]{i} }

// maxAddonDepth stops an addon that returns itself from expanding forever.
const maxAddonDepth = 8

// actionCount counts how many of the four ways of acting are set.
func actionCount[S any](run func(*Context[S]) tea.Cmd, do func(*Context[S]), opens string, modal func(*Context[S]) Modal[S]) int {
	n := 0
	if run != nil {
		n++
	}
	if do != nil {
		n++
	}
	if opens != "" {
		n++
	}
	if modal != nil {
		n++
	}
	return n
}

// runAction performs whichever way of acting is set.
func runAction[S any](c *Context[S], run func(*Context[S]) tea.Cmd, do func(*Context[S]), opens string, modal func(*Context[S]) Modal[S]) tea.Cmd {
	switch {
	case run != nil:
		return run(c)
	case do != nil:
		do(c)
	case opens != "":
		if err := c.OpenView(opens); err != nil {
			c.FlashError(err.Error())
		}
	case modal != nil:
		c.OpenModal(modal(c))
	}
	return nil
}
