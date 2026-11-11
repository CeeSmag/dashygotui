// Package layout arranges rendered panels on a page and insets content from
// the edges of the space it is given.
//
// It is pure: it knows nothing about Bubble Tea, themes or application state,
// only about rows of text and how many cells are available, so it can be used
// on its own. The dashygotui package re-exports [Mode] as Layout and [Spacing]
// as Spacing; applications normally use those.
package layout

import (
	"strings"

	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// Hard floors and breakpoints, in cells.
const (
	// MinCols and MinRows are the smallest terminal a dashboard draws in.
	// Below them it shows a notice instead of a frame nothing can be read from.
	MinCols = 40
	MinRows = 10

	// StatusPanelH is the default height of a widget panel: nine rows of
	// content plus the two borders.
	StatusPanelH = 11
	// WideCols is the width from which four panels may sit side by side.
	WideCols = 100
	// TightCols is the width from which two or three panels may sit side by
	// side when stacking them would not fit.
	TightCols = 80
	// MenuTwoColCols is the width a menu needs to be laid out in two columns.
	MenuTwoColCols = 76
)

// Spacing holds the four configurable gaps of the dashboard, each applied on
// every side (top, bottom, left and right) except SectionGap, which is the
// horizontal gap between side-by-side panels.
type Spacing struct {
	TerminalPage  int // terminal edge to page          0..6, default 0
	PageSection   int // page edge to its panels         0..6, default 0
	Section       int // between side-by-side panels     0..10, default 2
	SectionOption int // panel border to its options     0..6, default 0
}

// Bounds of each gap.
const (
	TerminalPageMax  = 6
	PageSectionMax   = 6
	SectionMax       = 10
	SectionOptionMax = 6
)

// DefaultSpacing is the spacing a dashboard starts with.
func DefaultSpacing() Spacing {
	return Spacing{TerminalPage: 0, PageSection: 0, Section: 2, SectionOption: 0}
}

// Clamp returns s with every gap forced into its valid range.
func (s Spacing) Clamp() Spacing {
	return Spacing{
		TerminalPage:  clamp(s.TerminalPage, TerminalPageMax),
		PageSection:   clamp(s.PageSection, PageSectionMax),
		Section:       clamp(s.Section, SectionMax),
		SectionOption: clamp(s.SectionOption, SectionOptionMax),
	}
}

// clamp forces v into [0, hi].
func clamp(v, hi int) int { return min(max(v, 0), hi) }

// Inset renders content inside a cols x rows area with gap cells of margin on
// every side, and returns exactly rows lines of exactly cols cells.
//
// The margin shrinks whenever it would leave the content less than minW x minH,
// so a generous setting never overflows a small terminal, even at 1x1.
func Inset(cols, rows, gap, minW, minH int, content func(w, h int) []string) []string {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	gap = max(gap, 0)
	hPad, vPad := gap, gap
	if cols-2*hPad < minW {
		hPad = max((cols-minW)/2, 0)
	}
	if rows-2*vPad < minH {
		vPad = max((rows-minH)/2, 0)
	}
	innerW := max(cols-2*hPad, 1)
	innerH := max(rows-2*vPad, 1)

	lines := panel.Frame(content(innerW, innerH), innerW, innerH)
	if hPad > 0 {
		margin := strings.Repeat(" ", hPad)
		for i, line := range lines {
			lines[i] = text.Fit(margin+line+margin, cols)
		}
	}
	if vPad > 0 {
		blank := strings.Repeat(" ", cols)
		out := make([]string, 0, rows)
		for range vPad {
			out = append(out, blank)
		}
		out = append(out, lines...)
		for range vPad {
			out = append(out, blank)
		}
		lines = out
	}
	return panel.Frame(lines, cols, rows)
}
