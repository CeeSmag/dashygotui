package stdaddons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/modal"
)

// DisplayPicker is an option that opens a display-mode picker: Auto (detect
// the Linux console), Unicode, or TTY. Moving the highlight previews each mode
// live; Enter keeps one and saves it, Esc restores the previous mode.
//
// Declare it as an addon.
type DisplayPicker[S any] struct {
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Hotkey runs the option from any page, like [dashygotui.Option].Hotkey. Optional.
	Hotkey string
	// Order sorts the option within its section. Optional.
	Order int
}

// Addons implements [dashygotui.Addon].
func (a DisplayPicker[S]) Addons() []dashygotui.Addon[S] {
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key:     "stdaddons.display",
		Section: a.Section,
		Number:  a.Number,
		Label:   "Display Mode",
		Badge: func(ctx *dashygotui.Context[S]) string {
			label := "Unicode"
			if ctx.TTY() {
				label = "TTY"
			}
			if ctx.GlyphMode() == dashygotui.GlyphsAuto {
				label += ", auto"
			}
			return ctx.Badge(label)
		},
		Do:     openDisplayPicker[S],
		Hotkey: a.Hotkey,
		Order:  a.Order,
	}}
}

func openDisplayPicker[S any](ctx *dashygotui.Context[S]) {
	modes := []dashygotui.GlyphMode{dashygotui.GlyphsAuto, dashygotui.GlyphsUnicode, dashygotui.GlyphsTTY}
	items := []modal.Item{
		{Label: "Auto", Detail: "use the TTY set when the terminal looks like the Linux console"},
		{Label: "Unicode", Detail: "rich symbols: rounded borders, braille graphs, checkmarks"},
		{Label: "TTY", Detail: "ASCII and straight lines, shade-block graphs, 16 colours"},
	}
	original := ctx.GlyphMode()
	cursor := 0
	for i, m := range modes {
		if m == original {
			cursor = i
		}
	}
	ctx.OpenModal(&modal.Choice[S]{
		Title:  "Display Mode",
		Help:   "Pick the symbol set. Use TTY if some symbols look wrong, as on the Linux console.",
		Items:  items,
		Cursor: cursor,
		OnMove: func(c *dashygotui.Context[S], i int) { c.PreviewGlyphMode(modes[i]) },
		OnSelect: func(c *dashygotui.Context[S], i int) tea.Cmd {
			c.SetGlyphMode(modes[i])
			c.Flash("Display mode: " + modes[i].String())
			return nil
		},
		OnCancel: func(c *dashygotui.Context[S]) tea.Cmd { c.PreviewGlyphMode(original); return nil },
	})
}
