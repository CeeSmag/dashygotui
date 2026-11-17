package dashygotui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/panel"
)

// Option is one numbered row of a panel: what happens when it is opened, and how
// its row reads. The map places it in a panel's Items; its position there is its
// number, and the map gives it a hotkey with [Hotkey].
type Option[S any] struct {
	// Label is the row text. Required.
	Label string
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

	// number is the row's number, given by its position in the map, and hot its
	// hotkey, given by the map with [Hotkey].
	number int
	hot    Hotkey[S]
}

// actions counts how many of Run, Do, Opens and Modal are set.
func (o Option[S]) actions() int { return actionCount(o.Run, o.Do, o.Opens, o.Modal) }

// run performs the option's action.
func (o Option[S]) run(c *Context[S]) tea.Cmd { return runAction(c, o.Run, o.Do, o.Opens, o.Modal) }

// Widget is a read-only panel: a chart, a gauge, a status readout. The map
// places it as a panel of a page ([Panel].Widget).
type Widget[S any] struct {
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
	// Tick is called on every refresh tick, for a widget that samples data over
	// time (a graph's history, say): the [Hook] Tick of the widget itself, so the
	// widget is one value. It runs on the update loop and must not block.
	// Optional.
	Tick func(*Context[S], time.Time) tea.Cmd
}

// View is a full-screen body opened from a row, a shortcut or
// [Context.OpenView], and left with Back or Quit (Esc/`b`/`q` by default): a
// log, a long text, a table with keys of its own. The map places it as a row of
// a panel (the row opens it), or in [Map.Views] when it has no row. For a full
// page of panels opened from a row, place a [Page] as the row instead: a
// sub-page.
//
// A view draws a header, its Body and a footer.
type View[S any] struct {
	// ID is the name [Context.OpenView], an option's Opens and
	// [Config.StartView] use. A view in [Map.Views] needs one; a view placed as
	// a row is named by its place in the map when it has none.
	ID string
	// Title names the view in the header and its panel. Required.
	Title string
	// Badge is shown in the view's panel (Follow, Framed, Panel). Optional.
	Badge func(*Context[S]) string
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
	// Row is the row that opens the view, when the view is placed as a row of a
	// panel: its Label (the view's Title by default), Hotkey, HotkeyHint, Badge or
	// Note. Its action is opening the view, so it sets no Run, Do, Opens or
	// Modal. Optional.
	Row Option[S]
	// Body draws the view. It returns rows for the given body size. Required.
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

	// sub is the page this view draws, when the view is a page placed as a row
	// of a [Map]: a sub-page.
	sub *pageDef[S]
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

// Page is a tab of the dashboard, in [Map.Pages], or a sub-page: a page placed
// as a row of a panel, which the row opens full-screen. Its Panels are what it
// shows.
type Page[S any] struct {
	// ID is the name [Context.GotoPage] uses for a tab, and [Context.OpenView]
	// and [Config.StartView] for a sub-page. Optional; unique when set.
	ID string
	// Title is the tab label, and the header of a sub-page. Required for a tab;
	// a sub-page's defaults to its Row's Label.
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
	// Session is called when the page becomes the one on screen and returns the
	// function to call when it stops being: when the user moves to another page
	// or opens a view over it. Coming back starts a new session. Use it for work
	// that only matters while the page is visible, such as asking a backend for
	// more detail. Like [View.Session], it is not called when the dashboard
	// quits, and a pop-up over the page does not end it. Optional.
	Session func(*Context[S]) (end func())
	// Panels are what the page shows, in order: each a widget or a list of rows,
	// arranged by Layout.
	Panels []Panel[S]
	// Row is the row that opens the page when the page is placed as a row of a
	// panel: a sub-page, a window the row opens. It is an ordinary option, often
	// declared in a feature file of its own (diag_opt.go), with a Label, Hotkey,
	// HotkeyHint or Badge; its action is opening the page, so it sets no Run, Do,
	// Opens or Modal. A tab has no row. Optional.
	Row Option[S]
}
