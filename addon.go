package dashygotui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/panel"
)

// Option is one numbered, selectable row in a section: the thing an addon adds
// to a menu. Register it with [Registry.Option].
type Option[S any] struct {
	// Key is a unique, stable identifier such as "storage". Required. It must not
	// start with "dashygotui.", which the library keeps for its own settings.
	Key string
	// Section is the ID of the section the option appears in, as declared in
	// [Config.Sections]. Required.
	Section string
	// Number is the number that selects the option from the keyboard: typing it
	// jumps straight to the row, on any page. It must be positive and unique
	// across the whole dashboard. Required.
	Number int
	// Order sorts the option within its section, ascending. When zero the
	// Number is used.
	Order int
	// Label is the row text. Required.
	Label string
	// Hotkey is an optional key that runs the option directly from any
	// dashboard page, without selecting it first: "l" to open the logs, say. It
	// is shown after the label as "(l)". It must not clash with a navigation key,
	// a digit, a shortcut or another option's hotkey. Inside a view's own
	// options it works only in that view.
	Hotkey string
	// HotkeyDisplay is how the hotkey is written after the label, such as
	// "Ctrl+L". It defaults to Hotkey.
	HotkeyDisplay string
	// HotkeyTone colours the hotkey after the label. The default is the label's
	// own colour.
	HotkeyTone Tone
	// HotkeyHint, when set, also lists the hotkey in the footer with this
	// wording, written as HotkeyDisplay (or Hotkey) and coloured by HotkeyTone,
	// so a hotkey needs no separate [Shortcut] to be advertised. It is shown on
	// the dashboard pages. Optional.
	HotkeyHint string
	// Badge returns the row's current state, already styled, such as "[ON ✓]".
	// Optional.
	Badge func(*Context[S]) string
	// Note is fixed, muted text shown in place of a badge, drawn "[Note]". It is
	// ignored when Badge is set. Optional.
	Note string

	// What the row does when opened. Set exactly one of Run, Do, Opens and
	// Modal; anything else is a startup error.

	// Run is called when the user opens the row. Return the command to run, or
	// nil.
	Run func(*Context[S]) tea.Cmd
	// Do is Run for an action with no command to return.
	Do func(*Context[S])
	// Opens is the ID of a registered [View] to open. It is checked at startup,
	// so a mistyped ID is an error rather than a message at run time.
	Opens string
	// Modal returns a pop-up to open, such as a [modal.Confirm] or a notice.
	Modal func(*Context[S]) Modal[S]
}

// actions counts how many of Run, Do, Opens and Modal are set.
func (o Option[S]) actions() int { return actionCount(o.Run, o.Do, o.Opens, o.Modal) }

// run performs the option's action.
func (o Option[S]) run(c *Context[S]) tea.Cmd { return runAction(c, o.Run, o.Do, o.Opens, o.Modal) }

func (o Option[S]) order() int {
	if o.Order != 0 {
		return o.Order
	}
	return o.Number
}

// Widget is a read-only panel on a page: a chart, a gauge, a status readout.
// Register it with [Registry.Widget].
type Widget[S any] struct {
	// Key is a unique, stable identifier such as "disk_io". Required.
	Key string
	// Page is the ID of the page the widget appears on, as declared in
	// [Config.Pages]. Required.
	Page string
	// Order sorts the widget among the page's panels (sections and widgets
	// together), ascending.
	Order int
	// Title is shown in the panel's top border. Required.
	Title string
	// Subtitle is shown in the panel's bottom border. Optional.
	Subtitle string
	// MinHeight is the height in rows, borders included, the widget would like
	// when panels are stacked. It defaults to 11.
	MinHeight int
	// Badge returns the state shown in the top border, already styled.
	// Optional.
	Badge func(*Context[S]) string
	// Render returns the panel's body rows for the given inner size. Rows wider
	// than the width are truncated and missing rows are left blank, but
	// returning exactly the right size is faster and clearer. Required.
	Render func(*Context[S], Size) []string
}

// View is a full-screen page opened from an option or a shortcut with
// [Context.OpenView] and left with the Back key. Register it with
// [Registry.View].
//
// A view draws a header, a body and a footer. The body is, in order of
// precedence, Body when set, otherwise a status panel (Status) above a menu of
// numbered rows (Options), either of which may be absent.
type View[S any] struct {
	// ID is the unique name passed to [Context.OpenView]. Required.
	ID string
	// Title names the view in the header and above the status panel. Required.
	Title string
	// Badge is shown beside the title. Optional.
	Badge func(*Context[S]) string
	// Subtitle is shown in the bottom border of the status panel. Optional.
	Subtitle string
	// OptionsTitle names the options panel. It defaults to "Options".
	OptionsTitle string
	// OptionsSubtitle is shown in the bottom border of the options panel.
	// Optional.
	OptionsSubtitle string
	// Header returns extra header segments, shown after the view title and
	// before the clock, such as "Mode: ON". They are dropped before the title
	// when the terminal is narrow. Optional.
	Header func(*Context[S]) []panel.Segment
	// Hints returns extra footer hints, shown with the view's own. Optional.
	Hints func(*Context[S]) []Hint
	// Mode is the footer's left-hand label, such as "LIVE". It defaults to the
	// Title in capitals.
	Mode string
	// Tag returns a short muted note for the header, such as "Mode: ON". Empty
	// shows nothing. Header is the way to add several or styled segments.
	// Optional.
	Tag func(*Context[S]) string
	// Status returns the rows of the status panel. Optional.
	Status func(*Context[S], Size) []string
	// Options returns the menu rows, called afresh on every frame so badges and
	// availability can change. Numbers need only be unique within the view; a
	// row with no Number is numbered by its position, from 1.
	Options func(*Context[S]) []Option[S]
	// Items is Options for a list that never changes: the rows themselves, with
	// badges that may still change. Set Items or Options, not both.
	Items []Option[S]
	// Entry makes the view reachable from a menu: it registers an option that
	// opens this view, with the hotkey and footer hint the entry asks for, so a
	// view and the way into it are one declaration. Optional.
	Entry *Entry[S]
	// Body draws the whole body itself, replacing Status and Options. It
	// returns rows for the given body size.
	Body func(*Context[S], Size) []string
	// Follow makes the view a log: Scroll and Tail, drawn in a panel. It needs
	// Body.
	Follow bool
	// Framed draws the body in a bordered panel with the view's Title, as Panel
	// does with no settings. It needs Body.
	Framed bool
	// Panel draws the body inside a bordered panel, as the status panel and the
	// menu are, instead of edge to edge. Body is then given the panel's inner
	// size, and a scrolling body shows its position in the panel's bottom
	// border. It needs Body. Optional.
	Panel *ViewPanel[S]
	// Scroll makes the engine scroll the body. Body may then return more rows
	// than fit: the dashboard shows a window of them, scrolls it with the
	// Up/Down, PageUp/PageDown, Home and End keys, and draws a scroll bar down
	// the right-hand edge (Body is given one column less to leave room for it,
	// but only when there is something to scroll). It needs Body. The scroll
	// position starts at the top each time the view is opened.
	Scroll bool
	// Tail makes a scrolling body behave like a log being written at the bottom:
	// the view starts at the newest rows (the bottom), Up goes back in time, End
	// returns to the live bottom, and a view scrolled back holds its place while
	// rows are appended instead of drifting. Without it the view starts at the
	// top and scrolls from there. It needs Scroll.
	Tail bool
	// Appended, for a Tail view whose body is a bounded history that drops its
	// oldest rows, returns how many rows have ever been appended (a count that
	// only rises). A view scrolled back uses it to hold its place: without it the
	// engine can only compare the body's length, which does not change once the
	// history is full. It needs Tail. Optional.
	Appended func(*Context[S]) int
	// Keys sees every key press first. Return handled == true to stop the
	// dashboard from acting on it. Optional.
	Keys func(ctx *Context[S], msg tea.KeyPressMsg) (cmd tea.Cmd, handled bool)
	// OnEnter is called when the view becomes the one on screen: when it is
	// opened, and when the user comes back to it from a view opened over it. Use
	// it to start work that only matters while the view is visible; return the
	// command to run, or nil. Optional.
	OnEnter func(*Context[S]) tea.Cmd
	// OnLeave is called when the view stops being the one on screen: when the
	// user goes back, opens another view over it, or moves to a page. It is not
	// called when the dashboard quits; stop long-running work with the context
	// given to [Dashboard.Model] or [Dashboard.Run]. Optional.
	OnLeave func(*Context[S]) tea.Cmd
	// Session is OnEnter and OnLeave in one: it is called when the view becomes
	// the one on screen and returns the function to call when it stops being. Use
	// it for work that starts and stops together, such as asking a backend for
	// more detail only while the view is open. Like OnLeave, it is not called when
	// the dashboard quits. Optional.
	Session func(*Context[S]) (end func())
}

// hasOptions reports whether the view has a menu of rows.
func (v View[S]) hasOptions() bool { return v.Options != nil || len(v.Items) > 0 }

// Entry describes the option that opens a view; see [View.Entry].
type Entry[S any] struct {
	// Section is the ID of the section the option appears in. Required.
	Section string
	// Number is the option's number, unique across the dashboard. Required.
	Number int
	// Order sorts the option within its section. When zero the Number is used.
	Order int
	// Label is the row text. It defaults to the view's Title.
	Label string
	// Hotkey, HotkeyDisplay, HotkeyTone and HotkeyHint are those of [Option].
	Hotkey        string
	HotkeyDisplay string
	HotkeyTone    Tone
	HotkeyHint    string
	// Badge and Note are those of [Option].
	Badge func(*Context[S]) string
	Note  string
}

// ViewPanel is the frame of a view whose Body is drawn in a panel. See
// [View.Panel].
type ViewPanel[S any] struct {
	// Title is shown in the panel's top border. It defaults to the view's Title.
	Title string
	// Badge is shown in the top border, already styled. It defaults to the
	// view's Badge.
	Badge func(*Context[S]) string
	// Subtitle is shown in the bottom border, before the scroll position when
	// the body scrolls. Optional.
	Subtitle string
}

// SectionSpec declares a section: a bordered panel on a page that holds
// options. Sections are declared in the dashboard file; addons fill them with
// options by section ID.
type SectionSpec[S any] struct {
	// ID is the unique name options refer to. Required.
	ID string
	// Page is the ID of the page the section appears on. Required.
	Page string
	// Order sorts the section among the page's panels, ascending.
	Order int
	// Title is shown in the panel's top border. Required.
	Title string
	// Subtitle is shown in the panel's bottom border. Optional.
	Subtitle string
	// Noun is the word counted in the default badge, "option" by default. The
	// badge reads "[3 options]".
	Noun string
	// Badge overrides the default count badge. Optional.
	Badge func(*Context[S]) string
}

// Page declares one tab of the dashboard. Pages appear in the order they are
// listed in [Config.Pages].
type Page[S any] struct {
	// ID is the unique name sections and widgets refer to. Required.
	ID string
	// Title is the tab label. Required.
	Title string
	// ShortTitle is used when the terminal is narrower than 100 columns.
	ShortTitle string
	// TinyTitle is used when the terminal is narrower than 75 columns.
	TinyTitle string
	// Layout arranges the page's panels. The zero value is [LayoutAuto].
	Layout Layout
	// Badge is shown after the title on the active tab. Optional.
	Badge func(*Context[S]) string
	// Fallback draws a compact version of the page for terminals too small for
	// the full layout, such as a one-panel summary. It receives the size of the
	// page body and returns its rows; the dashboard draws them in a panel titled
	// with the page's Title. While it is shown, the page's options cannot be
	// selected (the page is read-only) and number keys skip it. Optional;
	// FallbackBelow says when it applies.
	Fallback func(*Context[S], Size) []string
	// FallbackBelow is the page body size under which Fallback is used: when
	// the body is narrower than Cols or shorter than Rows. A zero field is
	// ignored. Fallback needs FallbackBelow or FallbackWhen, not both.
	FallbackBelow Size
	// FallbackWhen decides for any rule FallbackBelow cannot say, such as
	// "narrow and short, or very short". It receives the size of the page body.
	// Fallback needs FallbackBelow or FallbackWhen, not both.
	FallbackWhen func(Size) bool
}
