// Package stdaddons provides ready-made addons that every dashboard may want:
// a colour-theme picker, the four spacing sliders, a refresh-rate picker, a
// display-mode (Unicode or TTY) picker, a scrollable log view, and one-line
// settings (Toggle, Number, Pick, Text).
//
// They are opt-in. An application adds one by declaring it as a value in a
// one-file addon, with the section to appear in and the option number:
//
//	// dashboard/addons/opt_theme.go
//	var theme = stdaddons.ThemePicker[dashboard.State]{Section: "appearance", Number: 20}
//
// Each expands into ordinary [dashygotui.Option] values, so it takes part in
// number collision checks like any other. The functions that take a
// [dashygotui.Registry] (Theme, Spacing and the rest) are the v0.3 form of the
// same addons.
package stdaddons

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/theme"
)

// Setup tunes how a standard addon is registered. It is optional.
type Setup struct {
	// Hotkey runs the option from any page, like [dashygotui.Option].Hotkey.
	Hotkey string
	// Order sorts the option within its section, like [dashygotui.Option].Order.
	Order int
}

// Theme registers an option that opens a colour palette picker. Moving the
// highlight previews each palette live; Enter keeps one and saves the choice
// and Esc restores the previous palette.
func Theme[S any](reg *dashygotui.Registry[S], section string, number int, setup ...Setup) {
	reg.Add(ThemePicker[S]{Section: section, Number: number}.with(setup))
}

// ThemePicker is an option that opens a colour palette picker. Moving the
// highlight previews each palette live; Enter keeps one and saves the choice
// and Esc restores the previous palette.
//
// It is the value form of [Theme]: declare it as an addon.
type ThemePicker[S any] struct {
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Hotkey runs the option from any page, like [dashygotui.Option].Hotkey. Optional.
	Hotkey string
	// Order sorts the option within its section. Optional.
	Order int
}

func (a ThemePicker[S]) with(setup []Setup) ThemePicker[S] {
	if len(setup) > 0 {
		a.Hotkey, a.Order = setup[0].Hotkey, setup[0].Order
	}
	return a
}

// Addons implements [dashygotui.Addon].
func (a ThemePicker[S]) Addons() []dashygotui.Addon[S] {
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key:     "stdaddons.theme",
		Section: a.Section,
		Number:  a.Number,
		Label:   "Color Theme",
		Badge: func(ctx *dashygotui.Context[S]) string {
			if p, ok := theme.Lookup(ctx.Palettes(), ctx.PaletteID()); ok {
				return ctx.Badge(p.Name)
			}
			return ctx.Badge(ctx.PaletteID())
		},
		Do:     openThemePicker[S],
		Hotkey: a.Hotkey,
		Order:  a.Order,
	}}
}

func openThemePicker[S any](ctx *dashygotui.Context[S]) {
	palettes := ctx.Palettes()
	original := ctx.PaletteID()
	items := make([]modal.Item, len(palettes))
	cursor := 0
	cell := ctx.Theme().G.MeterFill // one cell per colour, in a glyph the terminal has
	for i, p := range palettes {
		items[i] = modal.Item{Label: p.Name, Badge: theme.SwatchWith(p, cell), Detail: p.Vibe}
		if p.ID == original {
			cursor = i
		}
	}
	ctx.OpenModal(&modal.Choice[S]{
		Title:  "Color Theme",
		Help:   "Move to preview a palette. Enter keeps it, Esc goes back.",
		Items:  items,
		Cursor: cursor,
		OnMove: func(c *dashygotui.Context[S], i int) { c.PreviewPalette(palettes[i].ID) },
		OnSelect: func(c *dashygotui.Context[S], i int) tea.Cmd {
			c.SetPalette(palettes[i].ID)
			c.Flash("Color theme: " + palettes[i].Name)
			return nil
		},
		OnCancel: func(c *dashygotui.Context[S]) tea.Cmd { c.PreviewPalette(original); return nil },
	})
}

// RefreshRate registers an option that opens a refresh-rate picker.
func RefreshRate[S any](reg *dashygotui.Registry[S], section string, number int, setup ...Setup) {
	reg.Add(RefreshRatePicker[S]{Section: section, Number: number}.with(setup))
}

// RefreshRatePicker is an option that opens a refresh-rate picker.
//
// It is the value form of [RefreshRate]: declare it as an addon.
type RefreshRatePicker[S any] struct {
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Hotkey runs the option from any page, like [dashygotui.Option].Hotkey. Optional.
	Hotkey string
	// Order sorts the option within its section. Optional.
	Order int
}

func (a RefreshRatePicker[S]) with(setup []Setup) RefreshRatePicker[S] {
	if len(setup) > 0 {
		a.Hotkey, a.Order = setup[0].Hotkey, setup[0].Order
	}
	return a
}

// Addons implements [dashygotui.Addon].
func (a RefreshRatePicker[S]) Addons() []dashygotui.Addon[S] {
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key:     "stdaddons.refresh",
		Section: a.Section,
		Number:  a.Number,
		Label:   "Refresh Rate",
		Badge:   func(ctx *dashygotui.Context[S]) string { return ctx.Badge(rateLabel(ctx.RefreshInterval())) },
		Do:      openRatePicker[S],
		Hotkey:  a.Hotkey,
		Order:   a.Order,
	}}
}

func openRatePicker[S any](ctx *dashygotui.Context[S]) {
	intervals := dashygotui.RefreshIntervals()
	details := []string{"smoothest animation, highest CPU", "very responsive", "smooth", "balanced (default)", "relaxed", "lowest CPU"}
	items := make([]modal.Item, len(intervals))
	cursor, best := 0, int64(-1)
	for i, d := range intervals {
		detail := ""
		if i < len(details) {
			detail = details[i]
		}
		items[i] = modal.Item{Label: rateLabel(d), Detail: detail}
		diff := int64(d - ctx.RefreshInterval())
		if diff < 0 {
			diff = -diff
		}
		if best < 0 || diff < best {
			cursor, best = i, diff
		}
	}
	ctx.OpenModal(&modal.Choice[S]{
		Title:  "Refresh Rate",
		Help:   "How often the dashboard refreshes its data.",
		Items:  items,
		Cursor: cursor,
		OnSelect: func(c *dashygotui.Context[S], i int) tea.Cmd {
			c.SetRefreshInterval(intervals[i])
			c.Flash("Refresh rate: " + rateLabel(intervals[i]))
			return nil
		},
	})
}

// rateLabel renders an interval as a rate: "1/s (1s)", "10/s (100ms)".
func rateLabel(d time.Duration) string {
	s := d.Seconds()
	if s < 1 {
		return fmt.Sprintf("%.0f/s (%dms)", 1/s, int(s*1000+0.5))
	}
	return fmt.Sprintf("%.3g/s (%gs)", 1/s, s)
}

// gap describes one of the four spacing settings.
type gap struct {
	name, detail string
	max          int
	get          func(dashygotui.Spacing) int
	set          func(*dashygotui.Spacing, int)
}

func gaps(both string) []gap {
	return []gap{
		{"Terminal " + both + " Page", "space between the terminal edge and the page", dashygotui.TerminalPageGapMax,
			func(s dashygotui.Spacing) int { return s.TerminalPage }, func(s *dashygotui.Spacing, v int) { s.TerminalPage = v }},
		{"Page " + both + " Section", "space between the page and its panels", dashygotui.PageSectionGapMax,
			func(s dashygotui.Spacing) int { return s.PageSection }, func(s *dashygotui.Spacing, v int) { s.PageSection = v }},
		{"Section " + both + " Section", "space between side-by-side panels", dashygotui.SectionGapMax,
			func(s dashygotui.Spacing) int { return s.Section }, func(s *dashygotui.Spacing, v int) { s.Section = v }},
		{"Section " + both + " Option", "space between a panel border and its rows", dashygotui.SectionOptionGapMax,
			func(s dashygotui.Spacing) int { return s.SectionOption }, func(s *dashygotui.Spacing, v int) { s.SectionOption = v }},
	}
}

// Spacing registers an option that opens the spacing menu: the four gaps
// that frame the dashboard, each adjusted with a live slider, plus a reset.
func Spacing[S any](reg *dashygotui.Registry[S], section string, number int, setup ...Setup) {
	reg.Add(SpacingMenu[S]{Section: section, Number: number}.with(setup))
}

// SpacingMenu is an option that opens the spacing menu: the four gaps that
// frame the dashboard, each adjusted with a live slider, plus a reset.
//
// It is the value form of [Spacing]: declare it as an addon.
type SpacingMenu[S any] struct {
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Hotkey runs the option from any page, like [dashygotui.Option].Hotkey. Optional.
	Hotkey string
	// Order sorts the option within its section. Optional.
	Order int
}

func (a SpacingMenu[S]) with(setup []Setup) SpacingMenu[S] {
	if len(setup) > 0 {
		a.Hotkey, a.Order = setup[0].Hotkey, setup[0].Order
	}
	return a
}

// Addons implements [dashygotui.Addon].
func (a SpacingMenu[S]) Addons() []dashygotui.Addon[S] {
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key:     "stdaddons.spacing",
		Section: a.Section,
		Number:  a.Number,
		Label:   "Section Gap",
		Badge: func(ctx *dashygotui.Context[S]) string {
			sp := ctx.Spacing()
			d := ctx.Theme().G.Dot
			return ctx.Badge(fmt.Sprintf("%d%s%d%s%d%s%d", sp.TerminalPage, d, sp.PageSection, d, sp.Section, d, sp.SectionOption))
		},
		Run:    openSpacingMenu[S],
		Hotkey: a.Hotkey,
		Order:  a.Order,
	}}
}

func openSpacingMenu[S any](ctx *dashygotui.Context[S]) tea.Cmd {
	all := gaps(ctx.Theme().G.Both)
	cur := ctx.Spacing()
	items := make([]modal.Item, 0, len(all)+1)
	for _, g := range all {
		items = append(items, modal.Item{Label: g.name, Badge: ctx.Badge(fmt.Sprintf("%d", g.get(cur))), Detail: g.detail})
	}
	items = append(items, modal.Item{Label: "Reset to defaults", Detail: "restore the original spacing"})

	ctx.OpenModal(&modal.Choice[S]{
		Title: "Section Gap",
		Help:  "Choose a gap to adjust.",
		Items: items,
		OnSelect: func(c *dashygotui.Context[S], i int) tea.Cmd {
			if i == len(all) {
				c.SetSpacing(dashygotui.DefaultSpacing())
				c.Flash("Spacing reset to defaults")
				return nil
			}
			return openGapSlider(c, all[i])
		},
	})
	return nil
}

func openGapSlider[S any](ctx *dashygotui.Context[S], g gap) tea.Cmd {
	original := ctx.Spacing()
	ctx.OpenModal(&modal.Slider[S]{
		Title: g.name,
		Help:  g.detail + ".",
		Unit:  "cells",
		Min:   0,
		Max:   g.max,
		Value: g.get(original),
		OnChange: func(c *dashygotui.Context[S], v int) {
			sp := original
			g.set(&sp, v)
			c.PreviewSpacing(sp)
		},
		OnDone: func(c *dashygotui.Context[S], v int) tea.Cmd {
			sp := original
			g.set(&sp, v)
			c.SetSpacing(sp)
			c.Flash(fmt.Sprintf("%s: %d", g.name, v))
			return openSpacingMenu(c)
		},
		OnCancel: openSpacingMenu[S],
	})
	return nil
}
