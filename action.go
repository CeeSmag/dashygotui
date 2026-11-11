package dashygotui

import tea "charm.land/bubbletea/v2"

// Addon is anything [Registry.Add] takes. [Option], [Shortcut], [View] and
// [Widget] are addons, and so is any type of yours that returns some of them
// from Addons: a "setting" that is an option plus a pop-up, say. The ready-made
// settings in package stdaddons are written this way.
//
// An addon that returns other addons is expanded in place, so what it returns
// is registered and checked like anything else.
type Addon[S any] interface {
	// Addons returns what to register. The four basic types return themselves.
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

// Invalid is an addon that records a registration error instead of registering
// anything. A composite addon returns it from Addons when its own settings make
// no sense (a number range that is empty, a list with nothing to pick), so the
// mistake is reported at startup with the call site, like any other.
type Invalid[S any] struct {
	// Reason says what is wrong, in a lower-case sentence fragment.
	Reason string
}

// Addons implements [Addon].
func (i Invalid[S]) Addons() []Addon[S] { return []Addon[S]{i} }

// maxAddonDepth stops an addon that returns itself from expanding forever.
const maxAddonDepth = 8

// registerAddon registers it, expanding composite addons.
func (r *Registry[S]) registerAddon(it Addon[S], site string, depth int) {
	switch v := it.(type) {
	case nil:
		r.mu.Lock()
		r.fail(site, "Add was given a nil addon")
		r.mu.Unlock()
	case Option[S]:
		v.register(r, site)
	case Shortcut[S]:
		v.register(r, site)
	case View[S]:
		v.register(r, site)
	case Widget[S]:
		v.register(r, site)
	case Invalid[S]:
		r.mu.Lock()
		r.fail(site, "%s", v.Reason)
		r.mu.Unlock()
	default:
		if depth >= maxAddonDepth {
			r.mu.Lock()
			r.fail(site, "addon %T expands into addons more than %d levels deep", it, maxAddonDepth)
			r.mu.Unlock()
			return
		}
		for _, sub := range it.Addons() {
			r.registerAddon(sub, site, depth+1)
		}
	}
}

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
