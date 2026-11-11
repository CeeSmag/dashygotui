package dashygotui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/layout"
)

// Size is a width and height in terminal cells.
type Size struct {
	Cols, Rows int
}

// Layout tells a page how to arrange its panels when there is room. On narrow
// terminals every layout degrades to vertical stacking. It is the [layout.Mode]
// of the layout package, re-exported so applications need only one import.
type Layout = layout.Mode

// The page layouts. LayoutAuto is the zero value and picks the natural
// arrangement for the panel count: 1 full, 2 side by side, 3 one above two,
// 4 a 2x2 grid, more stacked.
const (
	LayoutAuto            = layout.Auto
	LayoutColumns         = layout.Columns         // all panels side by side
	LayoutRows            = layout.Rows            // all panels stacked
	LayoutOneTopTwoBottom = layout.OneTopTwoBottom // three panels: one above two
	LayoutTwoTopOneBottom = layout.TwoTopOneBottom // three panels: two above one
	LayoutGrid2x2         = layout.Grid2x2         // four panels in a 2x2 grid
)

// Spacing is the four gaps, in cells, that frame a dashboard: terminal to page,
// page to panels, between side-by-side panels, and panel border to options. It
// is the [layout.Spacing] of the layout package, re-exported so applications
// need only one import. Values outside the documented ranges are clamped.
type Spacing = layout.Spacing

// Bounds of the spacing gaps.
const (
	TerminalPageGapMax  = layout.TerminalPageMax
	PageSectionGapMax   = layout.PageSectionMax
	SectionGapMax       = layout.SectionMax
	SectionOptionGapMax = layout.SectionOptionMax
)

// DefaultSpacing is the spacing a dashboard starts with.
func DefaultSpacing() Spacing { return layout.DefaultSpacing() }

// ExitInfo says why and how the dashboard ended. The application decides what
// to do about it after [Dashboard.Run] returns.
type ExitInfo struct {
	// Reason is an application-defined label passed to [Context.Quit]. It is
	// empty for a normal quit.
	Reason string
	// Message is text for the application to print after the terminal is
	// restored, or empty.
	Message string
}

// TaskResult is the outcome of a background task started with
// [Context.RunTask].
type TaskResult struct {
	// Token identifies the task run; it increases with every task.
	Token int
	// Tag is the tag the task was started with.
	Tag string
	// Text is the message the task returned.
	Text string
	// Err is the error the task returned, if any.
	Err error
}

// Flash durations: a hint is a nudge, a result is something the user may want
// to read twice.
const (
	FlashHint   = 2500 * time.Millisecond
	FlashResult = 6 * time.Second
)

// Hint is one key hint in the footer, rendered as "[Key] Label". Lower
// priorities are dropped first when the footer is too narrow.
type Hint struct {
	Key   string
	Label string
	Prio  int
}

// Shortcut is a global key binding on the dashboard pages.
type Shortcut[S any] struct {
	// Key is the key name as Bubble Tea reports it, e.g. "d" or "ctrl+r". It
	// must not collide with a navigation key or a digit.
	Key string
	// Label is the footer hint text. Leave it empty for no hint.
	Label string
	// Prio is the hint's priority; lower is dropped first. Default 30.
	Prio int
	// Run is called when the key is pressed.
	Run func(*Context[S]) tea.Cmd
}
