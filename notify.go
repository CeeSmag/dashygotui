package dashygotui

import tea "charm.land/bubbletea/v2"

// Sender delivers a message to a running Bubble Tea program. *[tea.Program]
// implements it.
type Sender interface {
	Send(msg tea.Msg)
}

// Attach connects the dashboard to the program that is running its [Model], so
// that [Dashboard.Notify] can reach it. [Dashboard.Run] does this itself; call
// Attach only when embedding the model in a program of your own. The returned
// function disconnects again.
func (d *Dashboard[S]) Attach(s Sender) (detach func()) {
	d.link.Lock()
	d.out = s
	d.link.Unlock()
	return func() {
		d.link.Lock()
		defer d.link.Unlock()
		if d.out == s {
			d.out = nil
		}
	}
}

// Notify tells the dashboard that something outside it changed, so it polls
// for fresh state now instead of waiting for the next interval. It is how a
// background goroutine, a file watcher or a websocket reader pushes an update
// into the UI.
//
// Notify is safe to call from any goroutine, any number of times: requests that
// arrive while a poll is running are merged into one more poll. It does nothing
// when the dashboard is not running. It waits for the dashboard's update loop
// to take the message, so do not call it from an addon callback (callbacks
// return a [tea.Cmd] for this; see [Context.Refresh]).
func (d *Dashboard[S]) Notify() {
	d.link.Lock()
	out := d.out
	d.link.Unlock()
	if out != nil {
		out.Send(notifyMsg{})
	}
}
