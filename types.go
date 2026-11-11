package dashygotui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/CeeSmag/dashygotui/layout"
	"github.com/CeeSmag/dashygotui/theme"
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

// GlyphMode selects the symbol set the dashboard draws with.
type GlyphMode int

// The glyph modes. GlyphsAuto is the zero value: use the TTY set when the
// environment looks like the Linux console (see [theme.DetectTTY]), otherwise
// the Unicode set.
const (
	GlyphsAuto GlyphMode = iota
	// GlyphsUnicode forces the rich Unicode symbol set.
	GlyphsUnicode
	// GlyphsTTY forces the conservative set for terminals with a small font:
	// ASCII plus straight line-drawing, shade-block graphs and the 16-colour
	// palette.
	GlyphsTTY
)

// String returns "auto", "unicode" or "tty".
func (g GlyphMode) String() string {
	switch g {
	case GlyphsUnicode:
		return "unicode"
	case GlyphsTTY:
		return "tty"
	default:
		return "auto"
	}
}

func parseGlyphMode(s string) GlyphMode {
	switch s {
	case "unicode":
		return GlyphsUnicode
	case "tty":
		return GlyphsTTY
	default:
		return GlyphsAuto
	}
}

// BackgroundMode says how a palette's background colour reaches the terminal.
type BackgroundMode int

// The background modes. BackgroundTerminal is the zero value.
const (
	// BackgroundTerminal sets the terminal's own background and foreground
	// colours (restored when the dashboard exits) when the active palette has
	// them. This also colours the window padding, which painting cannot reach.
	// Terminals that ignore the request keep their own colours; use
	// [BackgroundPaint] for those.
	BackgroundTerminal BackgroundMode = iota
	// BackgroundPaint does that and also paints every cell with the palette
	// background, so it shows even where the request is ignored (the Linux
	// console excepted: it has no true colour). It costs more output.
	BackgroundPaint
	// BackgroundOff never applies a background; the terminal keeps its colours.
	// This is the transparent mode: nothing covers the terminal's own
	// background, so a transparent, blurred or wallpapered terminal shows
	// through. See [BackgroundTransparent].
	BackgroundOff
)

// BackgroundTransparent is [BackgroundOff] under the name people look for: the
// dashboard leaves the background alone so a transparent terminal shows through.
//
// Whether [BackgroundTerminal] stays transparent too depends on the terminal:
// Alacritty applies its window opacity to the default background, which is what
// that mode changes, but not to cells with an explicit background colour, which
// is what [BackgroundPaint] writes.
const BackgroundTransparent = BackgroundOff

// String returns "terminal", "paint" or "off".
func (b BackgroundMode) String() string {
	switch b {
	case BackgroundPaint:
		return "paint"
	case BackgroundOff:
		return "off"
	default:
		return "terminal"
	}
}

func parseBackgroundMode(s string) BackgroundMode {
	switch s {
	case "paint":
		return BackgroundPaint
	case "off", "transparent":
		return BackgroundOff
	default:
		return BackgroundTerminal
	}
}

// Scope says where a [Shortcut] is active.
type Scope int

// The shortcut scopes. ScopePages is the zero value.
const (
	// ScopePages makes the shortcut work on the dashboard pages only.
	ScopePages Scope = iota
	// ScopeEverywhere makes the shortcut work on pages, in views and over
	// pop-ups, taking precedence over everything else, including the force-quit
	// keys.
	ScopeEverywhere
)

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

// Tone is a semantic colour for a footer hint or a hotkey. The theme decides
// what each tone looks like, so tones follow the palette.
type Tone int

// The tones. ToneDefault is the zero value: the ordinary key style.
const (
	ToneDefault Tone = iota
	// ToneDanger is for actions that stop or destroy something.
	ToneDanger
	// ToneWarn is for actions that interrupt, such as a restart.
	ToneWarn
	// ToneInfo is for actions that open something, such as a web page. It is
	// drawn in the palette's frame colour, which differs from the key style.
	ToneInfo
	// ToneGood is for actions that are safe and positive.
	ToneGood
	// ToneMuted is for actions that should not draw the eye.
	ToneMuted
)

// style is the theme's style for the tone.
func (t Tone) style(th *theme.Theme) lipgloss.Style {
	switch t {
	case ToneDanger:
		return th.Red
	case ToneWarn:
		return th.Orange.Bold(true)
	case ToneInfo:
		return th.Border.Bold(true)
	case ToneGood:
		return th.Green
	case ToneMuted:
		return th.Muted
	default:
		return th.Key
	}
}

// Hint is one entry of the footer's key strip: "[Key] Label".
type Hint struct {
	// Key is the key as it is written, such as "Ctrl+Q" or "F5".
	Key string
	// Label says what the key does.
	Label string
	// Tone colours the hint. The default is the plain key style.
	Tone Tone
	// Prio is the priority; lower is dropped first when the strip does not fit.
	// Default 30.
	Prio int
}

// Shortcut is a global key binding on the dashboard pages.
type Shortcut[S any] struct {
	// Key is the key name as Bubble Tea reports it, e.g. "d" or "ctrl+r". It
	// must not collide with a navigation key or a digit.
	Key string
	// Label is the footer hint text. Leave it empty for no hint.
	Label string
	// Display is how the key is written in the footer hint, such as "Ctrl+Q"
	// for the key "ctrl+q". It defaults to Key.
	Display string
	// Tone colours the hint, key and label together, so a dangerous shortcut can
	// look dangerous. The default is the plain key style.
	Tone Tone
	// Prio is the hint's priority; lower is dropped first. Default 30.
	Prio int
	// Scope says where the shortcut is active. The default is the dashboard
	// pages. An [ScopeEverywhere] shortcut may use a force-quit key such as
	// "ctrl+q" to replace the default quit; make sure the application still has
	// a way out.
	Scope Scope
	// Run is called when the key is pressed.
	Run func(*Context[S]) tea.Cmd
}
