package stdaddons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/modal"
)

// Background registers an option that lets the user choose how the palette's
// background reaches the terminal, and turn it off:
//
//   - Terminal: ask the terminal to use the palette's background and foreground.
//     The terminal's own colours come back when the dashboard exits. A
//     transparent terminal usually stays transparent.
//   - Paint: do that, and also paint every cell, for terminals that ignore the
//     request (some multiplexers and older terminals). Painted cells are opaque.
//   - Off (transparent): never touch the terminal's colours, so a transparent or
//     blurred terminal shows through.
//
// Moving the highlight previews each mode live; Enter keeps one and saves it,
// Esc restores the previous mode. Palettes with no background of their own, and
// TTY mode, are unaffected by this setting.
func Background[S any](reg *dashygotui.Registry[S], section string, number int, setup ...Setup) {
	reg.Option(apply(dashygotui.Option[S]{
		Key:     "stdaddons.background",
		Section: section,
		Number:  number,
		Label:   "Background",
		Badge: func(ctx *dashygotui.Context[S]) string {
			return ctx.Badge(ctx.BackgroundMode().String())
		},
		Run: openBackgroundPicker[S],
	}, setup))
}

func openBackgroundPicker[S any](ctx *dashygotui.Context[S]) tea.Cmd {
	modes := []dashygotui.BackgroundMode{dashygotui.BackgroundTerminal, dashygotui.BackgroundPaint, dashygotui.BackgroundOff}
	items := []modal.Item{
		{Label: "Terminal", Detail: "set the terminal's own background and foreground (restored on exit)"},
		{Label: "Paint", Detail: "also paint every cell, for terminals that ignore the request"},
		{Label: "Off (transparent)", Detail: "keep the terminal's own colours, so a transparent or blurred terminal shows through"},
	}
	original := ctx.BackgroundMode()
	cursor := 0
	for i, m := range modes {
		if m == original {
			cursor = i
		}
	}
	ctx.OpenModal(&modal.Choice[S]{
		Title:  "Background",
		Help:   "Whether the theme colours the whole terminal. If you see no background, try Paint. For a see-through terminal, choose Off.",
		Items:  items,
		Cursor: cursor,
		OnMove: func(c *dashygotui.Context[S], i int) { c.PreviewBackgroundMode(modes[i]) },
		OnSelect: func(c *dashygotui.Context[S], i int) tea.Cmd {
			c.SetBackgroundMode(modes[i])
			c.Flash("Background: " + modes[i].String())
			return nil
		},
		OnCancel: func(c *dashygotui.Context[S]) tea.Cmd { c.PreviewBackgroundMode(original); return nil },
	})
	return nil
}

// Transparency registers a one-key switch for a transparent background: on
// leaves the terminal's own background alone (so a transparent, blurred or
// wallpapered terminal shows through), off brings the themed background back in
// the mode that was in use before. It changes the same setting as [Background],
// so the two always agree.
func Transparency[S any](reg *dashygotui.Registry[S], section string, number int, setup ...Setup) {
	const previousKey = "stdaddons.transparency.previous"
	reg.Option(apply(dashygotui.Option[S]{
		Key:     "stdaddons.transparency",
		Section: section,
		Number:  number,
		Label:   "Transparent Background",
		Badge: func(ctx *dashygotui.Context[S]) string {
			return ctx.OnOff(ctx.BackgroundMode() == dashygotui.BackgroundOff)
		},
		Run: func(ctx *dashygotui.Context[S]) tea.Cmd {
			if ctx.BackgroundMode() != dashygotui.BackgroundOff {
				// Remember the themed mode so turning transparency off restores it.
				_ = ctx.Store().Set(previousKey, ctx.BackgroundMode().String()) // best effort: Terminal is the fallback
				ctx.SetBackgroundMode(dashygotui.BackgroundOff)
				ctx.Flash("Transparent background on: the terminal shows through")
				return nil
			}
			restore := dashygotui.BackgroundTerminal
			var prev string
			if ok, _ := ctx.Store().Get(previousKey, &prev); ok && prev == dashygotui.BackgroundPaint.String() {
				restore = dashygotui.BackgroundPaint
			}
			ctx.SetBackgroundMode(restore)
			ctx.Flash("Transparent background off: themed background (" + restore.String() + ")")
			return nil
		},
	}, setup))
}
