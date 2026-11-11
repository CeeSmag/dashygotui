// Package stdaddons provides ready-made addons that every dashboard may want:
// a colour-theme picker, the four spacing sliders, and a refresh-rate picker.
//
// They are opt-in. An application adds one by calling its function from a
// one-file addon, passing the registry, the section to appear in, and the
// option number:
//
//	// dashboard/addons/opt_theme.go
//	func init() { stdaddons.Theme(dashboard.Registry, "appearance", 20) }
//
// Each registers an ordinary [dashygotui.Option], so it takes part in number
// collision checks like any other.
package stdaddons

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/theme"
)

// Theme registers an option that opens a colour palette picker. Moving the
// highlight previews each palette live; Enter keeps one and saves the choice
// and Esc restores the previous palette.
func Theme[S any](reg *dashygotui.Registry[S], section string, number int) {
	reg.Option(dashygotui.Option[S]{
		Key:     "stdaddons.theme",
		Section: section,
		Number:  number,
		Label:   "Color Theme",
		Badge: func(ctx *dashygotui.Context[S]) string {
			if p, ok := theme.Lookup(ctx.Palettes(), ctx.PaletteID()); ok {
				return ctx.Badge(p.Name)
			}
			return ctx.Badge(ctx.PaletteID())
		},
		Run: openThemePicker[S],
	})
}

func openThemePicker[S any](ctx *dashygotui.Context[S]) tea.Cmd {
	palettes := ctx.Palettes()
	original := ctx.PaletteID()
	items := make([]modal.Item, len(palettes))
	cursor := 0
	for i, p := range palettes {
		items[i] = modal.Item{Label: p.Name, Badge: theme.Swatch(p), Detail: p.Vibe}
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
	return nil
}

// RefreshRate registers an option that opens a refresh-rate picker.
func RefreshRate[S any](reg *dashygotui.Registry[S], section string, number int) {
	reg.Option(dashygotui.Option[S]{
		Key:     "stdaddons.refresh",
		Section: section,
		Number:  number,
		Label:   "Refresh Rate",
		Badge:   func(ctx *dashygotui.Context[S]) string { return ctx.Badge(rateLabel(ctx.RefreshInterval())) },
		Run:     openRatePicker[S],
	})
}

func openRatePicker[S any](ctx *dashygotui.Context[S]) tea.Cmd {
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
	return nil
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

func gaps() []gap {
	return []gap{
		{"Terminal ↔ Page", "space between the terminal edge and the page", dashygotui.TerminalPageGapMax,
			func(s dashygotui.Spacing) int { return s.TerminalPage }, func(s *dashygotui.Spacing, v int) { s.TerminalPage = v }},
		{"Page ↔ Section", "space between the page and its panels", dashygotui.PageSectionGapMax,
			func(s dashygotui.Spacing) int { return s.PageSection }, func(s *dashygotui.Spacing, v int) { s.PageSection = v }},
		{"Section ↔ Section", "space between side-by-side panels", dashygotui.SectionGapMax,
			func(s dashygotui.Spacing) int { return s.Section }, func(s *dashygotui.Spacing, v int) { s.Section = v }},
		{"Section ↔ Option", "space between a panel border and its rows", dashygotui.SectionOptionGapMax,
			func(s dashygotui.Spacing) int { return s.SectionOption }, func(s *dashygotui.Spacing, v int) { s.SectionOption = v }},
	}
}

// Spacing registers an option that opens the spacing menu: the four gaps
// that frame the dashboard, each adjusted with a live slider, plus a reset.
func Spacing[S any](reg *dashygotui.Registry[S], section string, number int) {
	reg.Option(dashygotui.Option[S]{
		Key:     "stdaddons.spacing",
		Section: section,
		Number:  number,
		Label:   "Section Gap",
		Badge: func(ctx *dashygotui.Context[S]) string {
			sp := ctx.Spacing()
			return ctx.Badge(fmt.Sprintf("%d·%d·%d·%d", sp.TerminalPage, sp.PageSection, sp.Section, sp.SectionOption))
		},
		Run: openSpacingMenu[S],
	})
}

func openSpacingMenu[S any](ctx *dashygotui.Context[S]) tea.Cmd {
	all := gaps()
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
