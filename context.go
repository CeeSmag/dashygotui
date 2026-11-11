package dashygotui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/theme"
)

// Context is the handle an addon callback receives. It exposes the current
// state and the actions an addon may take, and is how addons talk to the
// dashboard.
//
// A Context is valid only inside the callback it was passed to. Do not keep it,
// and do not use it from another goroutine: the dashboard's state is changed
// only on the Bubble Tea update loop. Work that takes time belongs in
// [Context.RunTask].
type Context[S any] struct {
	m *model[S]
}

// State returns the latest state from the configured Poll function. Before the
// first successful poll it is the zero value; see [Context.Ready].
func (c *Context[S]) State() S { return c.m.state }

// Ready reports whether Poll has succeeded at least once.
func (c *Context[S]) Ready() bool { return c.m.ready }

// Err returns the error of the most recent poll, or nil when it succeeded.
func (c *Context[S]) Err() error { return c.m.stateErr }

// Size returns the terminal size.
func (c *Context[S]) Size() Size { return Size{Cols: c.m.cols, Rows: c.m.rows} }

// Theme returns the active theme, for colouring output and calling the chart
// and panel helpers.
func (c *Context[S]) Theme() *theme.Theme { return c.m.th }

// Now returns the current time from the configured clock.
func (c *Context[S]) Now() time.Time { return c.m.now() }

// Ticks counts the refresh ticks since the dashboard started. It drives
// animations such as the footer spinner.
func (c *Context[S]) Ticks() int { return c.m.ticks }

// Store returns the settings store. Namespace keys with the addon's own Key;
// the "dashygotui." prefix belongs to the library.
func (c *Context[S]) Store() Store { return c.m.store }

// Flash shows a short hint message in the footer.
func (c *Context[S]) Flash(msg string) { c.m.setFlash(msg, false, FlashHint) }

// FlashError shows an error message in the footer.
func (c *Context[S]) FlashError(msg string) {
	c.m.setFlash(c.m.th.G.Cross+" "+msg, true, FlashResult)
}

// FlashResult shows the outcome of an action in the footer: a tick and the
// text on success, a cross and the reason on failure.
func (c *Context[S]) FlashResult(msg string, err error) { c.m.flashResult(msg, err) }

// OpenModal shows modal over the current page, replacing any open one.
func (c *Context[S]) OpenModal(mod Modal[S]) {
	c.m.modal = mod
	c.m.modalScroll = 0
}

// CloseModal dismisses the open modal, if any.
func (c *Context[S]) CloseModal() {
	c.m.modal = nil
	c.m.modalScroll = 0
}

// ModalScroll is the scroll offset of the open modal.
func (c *Context[S]) ModalScroll() int { return c.m.modalScroll }

// SetModalScroll sets the scroll offset of the open modal.
func (c *Context[S]) SetModalScroll(v int) { c.m.modalScroll = max(v, 0) }

// OpenView opens the registered full-screen view with the given ID. The Back
// key returns to where the user came from.
func (c *Context[S]) OpenView(id string) error {
	if _, ok := c.m.d.views[id]; !ok {
		return fmt.Errorf("no view %q is registered", id)
	}
	c.m.openView(id)
	return nil
}

// Back leaves the current view, returning to the previous view or the page.
func (c *Context[S]) Back() { c.m.back() }

// Page returns the ID of the dashboard page the user is on.
func (c *Context[S]) Page() string { return c.m.d.pages[c.m.page].page.ID }

// GotoPage switches to the page with the given ID and reports whether it
// exists. It leaves any open view.
func (c *Context[S]) GotoPage(id string) bool {
	for i, p := range c.m.d.pages {
		if p.page.ID == id {
			c.m.view, c.m.stack = "", nil
			c.m.gotoPage(i, -1, 0)
			return true
		}
	}
	return false
}

// Quit ends the dashboard. reason and message come back to the application in
// [ExitInfo]; use them to ask for post-exit work such as an uninstall.
func (c *Context[S]) Quit(reason, message string) tea.Cmd {
	return c.m.quit(ExitInfo{Reason: reason, Message: message})
}

// RunTask runs fn in the background and reports its result through the
// registered [Registry.OnResult] hooks, the open modal, or, by default, the
// footer. pending, if not empty, is shown in the footer meanwhile. fn must
// return when ctx is cancelled.
func (c *Context[S]) RunTask(tag, pending string, fn func(ctx context.Context) (string, error)) tea.Cmd {
	return c.m.runTask(tag, pending, fn)
}

// Refresh polls for fresh state now rather than waiting for the next interval.
// It returns nil when nothing is configured to poll or a poll is under way.
func (c *Context[S]) Refresh() tea.Cmd { return c.m.pollCmd() }

// Spacing returns the active spacing.
func (c *Context[S]) Spacing() Spacing { return c.m.spacing }

// SetSpacing applies sp immediately, clamped to its valid ranges, and saves it
// in the store.
func (c *Context[S]) SetSpacing(sp Spacing) {
	c.m.spacing = sp.Clamp()
	c.m.persist(storeKeySpacing, c.m.spacing)
}

// PreviewSpacing applies sp for this run without saving it, so a slider can
// show its effect live. Follow it with [Context.SetSpacing] to keep the value,
// or with PreviewSpacing of the original to undo.
func (c *Context[S]) PreviewSpacing(sp Spacing) { c.m.spacing = sp.Clamp() }

// PreviewPalette applies the palette with the given ID for this run without
// saving it, so a picker can show its effect live. It reports whether the
// palette exists.
func (c *Context[S]) PreviewPalette(id string) bool {
	if _, ok := theme.Lookup(c.m.d.palettes, id); !ok {
		return false
	}
	c.m.applyPalette(id)
	return true
}

// GlyphMode returns the chosen glyph mode: auto, Unicode or TTY.
func (c *Context[S]) GlyphMode() GlyphMode { return c.m.glyphMode }

// TTY reports whether the TTY glyph set is in use, whether chosen or detected.
func (c *Context[S]) TTY() bool { return c.m.tty }

// SetGlyphMode switches the symbol set immediately and saves the choice. On a
// TTY, with no palette chosen, the 16-colour palette follows along.
func (c *Context[S]) SetGlyphMode(mode GlyphMode) {
	c.m.glyphMode = mode
	c.m.rebuildTheme()
	c.m.persist(storeKeyGlyphs, mode.String())
}

// PreviewGlyphMode applies mode for this run without saving it, so a picker can
// show its effect live. Follow it with [Context.SetGlyphMode] to keep the
// choice, or with PreviewGlyphMode of the original to undo.
func (c *Context[S]) PreviewGlyphMode(mode GlyphMode) {
	c.m.glyphMode = mode
	c.m.rebuildTheme()
}

// BackgroundMode returns how the palette's background reaches the terminal.
func (c *Context[S]) BackgroundMode() BackgroundMode { return c.m.bgMode }

// SetBackgroundMode switches how the palette's background is applied (the
// terminal's own colours, painted cells, or none) immediately and saves the
// choice.
func (c *Context[S]) SetBackgroundMode(mode BackgroundMode) {
	c.m.bgMode = mode
	c.m.persist(storeKeyBG, mode.String())
}

// PreviewBackgroundMode applies mode for this run without saving it, so a
// picker can show its effect live. Follow it with [Context.SetBackgroundMode] to
// keep the choice, or with PreviewBackgroundMode of the original to undo.
func (c *Context[S]) PreviewBackgroundMode(mode BackgroundMode) { c.m.bgMode = mode }

// Palettes lists the colour palettes the user can choose from.
func (c *Context[S]) Palettes() []theme.Palette {
	return append([]theme.Palette(nil), c.m.d.palettes...)
}

// PaletteID returns the ID of the active palette.
func (c *Context[S]) PaletteID() string { return c.m.th.Palette.ID }

// SetPalette applies the palette with the given ID immediately and saves the
// choice. It reports whether the palette exists; if it does not, nothing
// changes.
func (c *Context[S]) SetPalette(id string) bool {
	if _, ok := theme.Lookup(c.m.d.palettes, id); !ok {
		return false
	}
	c.m.paletteChosen = true
	c.m.applyPalette(id)
	c.m.persist(storeKeyPalette, c.m.palette)
	return true
}

// RefreshInterval returns the current refresh interval.
func (c *Context[S]) RefreshInterval() time.Duration { return c.m.interval }

// SetRefreshInterval changes how often the dashboard refreshes, saves the
// choice, and takes effect from the next tick. Non-positive values are ignored.
func (c *Context[S]) SetRefreshInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	c.m.interval = d
	c.m.persist(storeKeyInterval, d.Milliseconds())
}

// OnOff returns a styled "[ON ✓]" or "[OFF ✗]" badge.
func (c *Context[S]) OnOff(on bool) string {
	if on {
		return c.m.th.Green.Render("[ON " + c.m.th.G.Check + "]")
	}
	return c.m.th.Red.Render("[OFF " + c.m.th.G.Cross + "]")
}

// Badge returns a plain styled state badge: "[text]".
func (c *Context[S]) Badge(text string) string { return c.m.th.White.Render("[" + text + "]") }

// Count returns a muted count badge: "[3 files]".
func (c *Context[S]) Count(n int, what string) string {
	return c.m.th.Muted.Render(fmt.Sprintf("[%d %s]", n, what))
}

// persist saves a library setting and reports a failure in the footer; a
// setting that cannot be saved is still applied for this run.
func (m *model[S]) persist(key string, v any) {
	if err := m.store.Set(key, v); err != nil {
		m.setFlash(m.th.G.Cross+" setting not saved: "+err.Error(), true, FlashResult)
	}
}
