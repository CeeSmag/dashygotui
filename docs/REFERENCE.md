# Field reference

Generated from the doc comments of the types an addon author fills in. Do not edit: change the
comment in the code and run `go test -run TestReferenceIsCurrent -update-api .`.
For when to use what, read `COOKBOOK.md`; for the narrative, `ADDONS.md`.

## Package dashygotui

### `Config`

Config describes a dashboard. Map is required; the rest have sensible defaults.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title names the application in the header and the terminal window title. |
| `Map` | `Map[S]` | Map is what the dashboard contains and where: its pages, their panels and rows, and the dashboard's shortcuts and hooks. An application writes it in addons/addons.go. Required: it needs at least one page. |
| `Poll` | `func(ctx context.Context) (S, error)` | Poll fetches the application state. It is called at Interval, in the background, and once at start. When nil the state stays Initial and nothing is polled. |
| `Initial` | `S` | Initial is the state before the first poll returns: what the dashboard shows, and what a key pressed or a start view opened in that moment sees. Put there what is known before polling, such as the handles to the services the state carries, so nothing reads them empty. Context.Ready stays false until the first poll succeeds. Optional; the zero value of S by default. |
| `Interval` | `time.Duration` | Interval is how often the dashboard refreshes. Default one second. The user can change it at run time; the choice is kept in Store. |
| `PollTimeout` | `time.Duration` | PollTimeout is the deadline of one Poll call. Default ten seconds; a negative value means no deadline. A poll that runs past it counts as failed, with an error wrapping [context.DeadlineExceeded], and the next poll starts on schedule. Poll should return when its ctx is done; a call that ignores it is abandoned and its result is discarded, so Poll may be called again while an abandoned call is still running. |
| `Header` | `func(*Context[S]) []panel.Segment` | Header returns the segments of the header strip. Lower priorities are dropped first on narrow terminals and the first is never dropped. The default shows Title and a clock. |
| `StartView` | `[]string` | StartView is a stack of view IDs to open at start, bottom first: the last is the one on screen and Back walks down the rest and then to the pages. Each is opened as if the user had, so OnEnter and OnLeave run. Optional. |
| `Store` | `Store` | Store keeps settings between runs. The default is in-memory. |
| `Palettes` | `[]theme.Palette` | Palettes are extra colour palettes offered next to the built-in ones. |
| `Palette` | `string` | Palette is the ID of the palette used when the user has not chosen one. Default [theme.DefaultID]. |
| `Glyphs` | `GlyphMode` | Glyphs selects the symbol set. The default, [GlyphsAuto], uses the TTY set when the environment looks like the Linux console and the Unicode set otherwise; the user's choice, saved in Store, wins over it. |
| `Background` | `BackgroundMode` | Background says how a palette's background colour reaches the terminal. The default sets the terminal's own colours. It is the mode the dashboard starts in; the user's saved choice (see stdaddons.Background) wins over it. Use [BackgroundOff] to never touch the terminal's colours. |
| `DetectTTY` | `func() bool` | DetectTTY decides what [GlyphsAuto] means, replacing the built-in check of the environment. Use it for a test the library cannot make for you, such as asking the operating system whether the terminal is a Linux virtual console. Optional. |
| `Getenv` | `func(string) string` | Getenv reads environment variables for glyph auto-detection. Default [os.Getenv]; tests inject their own. |
| `OnForceQuit` | `func(*Context[S]) tea.Cmd` | OnForceQuit is called instead of quitting when a force-quit key is pressed (ctrl+q and ctrl+c by default), for example to open a "Stop and exit?" confirmation. Return the command to run. Optional. |
| `OnUnboundKey` | `func(*Context[S], string) string` | OnUnboundKey is called when a key on a page or in a view is not bound to anything (not a shortcut, hotkey, navigation key, digit jump, quit, or rate change). Return a non-empty string to show it as a footer hint, or "" to stay silent. Digits that match no option still show the built-in "No option" flash and do not call this. Optional. |
| `Keys` | `KeyMap` | Keys overrides the default key bindings. Unset bindings keep their defaults. |
| `Spacing` | `*Spacing` | Spacing overrides the default spacing; the user's saved choice in Store wins over both. |
| `AltScroll` | `bool` | AltScroll asks the terminal to turn the mouse wheel into arrow keys while the dashboard is up, which keeps text selection with the terminal. It is off by default. |
| `Clock` | `func() time.Time` | Clock supplies the current time. Default [time.Now]; tests inject their own. |
| `Timer` | `func(d time.Duration, fn func(time.Time) tea.Msg) tea.Cmd` | Timer creates the command that delivers a message after d. Default [tea.Tick]. It is a test seam: the dashtest package substitutes timers it can fire on demand, so tests never sleep. |
| `Exec` | `func(cmd tea.ExecCommand, done tea.ExecCallback) tea.Cmd` | Exec creates the command that hands the terminal to cmd and, when it ends, delivers done's message: what [Context.Exec] runs. Default [tea.Exec]. It is a test seam: the dashtest package records the command instead, and the test decides how it ends. |

### `Map`

Map is what a dashboard contains and where: its pages, the panels on each page, the rows in each panel, and the keys and hooks that belong to the whole dashboard. An application writes it in one file (addons/addons.go), so the map is the only place anything is placed; an addon says only what it does. Give it to [New] in [Config.Map].

| Field | Type | What it does |
| --- | --- | --- |
| `Pages` | `[]Page[S]` | Pages are the dashboard's tabs, in tab order. At least one is required. |
| `Shortcuts` | `[]Shortcut[S]` | Shortcuts are the keys that work on every page, or everywhere. |
| `Hooks` | `[]Hook[S]` | Hooks react to ticks, messages and task results for the whole dashboard. |
| `Views` | `[]View[S]` | Views are views with no row of their own: opened by a shortcut, by an option's Opens, by [Context.OpenView] or by [Config.StartView]. Each needs an ID. A view placed as a row of a panel is not listed here. |

### `Page`

Page is a tab of the dashboard, in [Map.Pages], or a sub-page: a page placed as a row of a panel, which the row opens full-screen. Its Panels are what it shows.

| Field | Type | What it does |
| --- | --- | --- |
| `ID` | `string` | ID is the name [Context.GotoPage] uses for a tab, and [Context.OpenView] and [Config.StartView] for a sub-page. Optional; unique when set. |
| `Title` | `string` | Title is the tab label, and the header of a sub-page. Required for a tab; a sub-page's defaults to its Row's Label. |
| `ShortTitle` | `string` | ShortTitle is used when the terminal is narrower than 100 columns. |
| `TinyTitle` | `string` | TinyTitle is used when the terminal is narrower than 75 columns. |
| `Layout` | `Layout` | Layout arranges the page's panels. The zero value is [LayoutAuto]. |
| `Badge` | `func(*Context[S]) string` | Badge is shown after the title on the active tab. Optional. |
| `Fallback` | `func(*Context[S], Size) []string` | Fallback draws a compact version of the page for terminals too small for the full layout, such as a one-panel summary. It receives the size of the page body and returns its rows; the dashboard draws them in a panel titled with the page's Title. While it is shown, the page's options cannot be selected (the page is read-only) and number keys skip it. Optional; FallbackBelow says when it applies. |
| `FallbackBelow` | `Size` | FallbackBelow is the page body size under which Fallback is used: when the body is narrower than Cols or shorter than Rows. A zero field is ignored. Fallback needs FallbackBelow or FallbackWhen, not both. |
| `FallbackWhen` | `func(Size) bool` | FallbackWhen decides for any rule FallbackBelow cannot say, such as "narrow and short, or very short". It receives the size of the page body. Fallback needs FallbackBelow or FallbackWhen, not both. |
| `Session` | `func(*Context[S]) (end func())` | Session is called when the page becomes the one on screen and returns the function to call when it stops being: when the user moves to another page or opens a view over it. Coming back starts a new session. Use it for work that only matters while the page is visible, such as asking a backend for more detail. Like [View.Session], it is not called when the dashboard quits, and a pop-up over the page does not end it. Optional. |
| `Panels` | `[]Panel[S]` | Panels are what the page shows, in order: each a widget or a list of rows, arranged by Layout. |
| `Row` | `Option[S]` | Row is the row that opens the page when the page is placed as a row of a panel: a sub-page, a window the row opens. It is an ordinary option, often declared in a feature file of its own (diag_opt.go), with a Label, Hotkey, HotkeyHint or Badge; its action is opening the page, so it sets no Run, Do, Opens or Modal. A tab has no row. Optional. |

### `Panel`

Panel is one bordered panel of a page: one widget, or a list of numbered rows. Set Widget or Items, not both.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the panel's top border. Required for a list of rows; for a widget it replaces the widget's own Title. |
| `Subtitle` | `string` | Subtitle is shown in the bottom border. For a widget it replaces the widget's own Subtitle. Optional. |
| `Badge` | `func(*Context[S]) string` | Badge is shown in the top border, already styled. A list of rows shows a count, "[3 options]", by default; for a widget it replaces the widget's own Badge. Optional. |
| `Noun` | `string` | Noun is the word the default count badge counts, "option" by default. |
| `Number` | `int` | Number is the number of the first row; the rows after it count up from there. Without it the panel's rows continue from the rows before them, or start at 1. Optional. |
| `Widget` | `Widget[S]` | Widget is the panel's widget. Optional. |
| `Items` | `[]Addon[S]` | Items are the panel's rows, top to bottom: an [Option], a [View] (the row opens it), a [Page] (the row opens it as a sub-page), or a ready-made setting such as one from package stdaddons. |

### `Option`

Option is one numbered row of a panel: what happens when it is opened, and how its row reads. The map places it in a panel's Items; its position there is its number, and the map gives it a hotkey with [Hotkey].

| Field | Type | What it does |
| --- | --- | --- |
| `Label` | `string` | Label is the row text. Required. |
| `Badge` | `func(*Context[S]) string` | Badge returns the row's current state, already styled, such as "[ON ✓]". Optional. |
| `Note` | `string` | Note is fixed, muted text shown in place of a badge, drawn "[Note]". It is ignored when Badge is set. Optional. |
| `Run` | `func(*Context[S]) tea.Cmd` | Run is called when the user opens the row. Return the command to run, or nil. |
| `Do` | `func(*Context[S])` | Do is Run for an action with no command to return. |
| `Opens` | `string` | Opens is the ID of a registered [View] to open. It is checked at startup, so a mistyped ID is an error rather than a message at run time. |
| `Modal` | `func(*Context[S]) Modal[S]` | Modal returns a pop-up to open, such as a [modal.Confirm] or a notice. |

### `Widget`

Widget is a read-only panel: a chart, a gauge, a status readout. The map places it as a panel of a page ([Panel].Widget).

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the panel's top border. Required. |
| `Subtitle` | `string` | Subtitle is shown in the panel's bottom border. Optional. |
| `MinHeight` | `int` | MinHeight is the height in rows, borders included, the widget would like when panels are stacked. It defaults to 11. |
| `Badge` | `func(*Context[S]) string` | Badge returns the state shown in the top border, already styled. Optional. |
| `Render` | `func(*Context[S], Size) []string` | Render returns the panel's body rows for the given inner size. Rows wider than the width are truncated and missing rows are left blank, but returning exactly the right size is faster and clearer. Required. |
| `Tick` | `func(*Context[S], time.Time) tea.Cmd` | Tick is called on every refresh tick, for a widget that samples data over time (a graph's history, say): the [Hook] Tick of the widget itself, so the widget is one value. It runs on the update loop and must not block. Optional. |

### `View`

View is a full-screen body opened from a row, a shortcut or [Context.OpenView], and left with Back or Quit (Esc/`b`/`q` by default): a log, a long text, a table with keys of its own. The map places it as a row of a panel (the row opens it), or in [Map.Views] when it has no row. For a full page of panels opened from a row, place a [Page] as the row instead: a sub-page. A view draws a header, its Body and a footer.

| Field | Type | What it does |
| --- | --- | --- |
| `ID` | `string` | ID is the name [Context.OpenView], an option's Opens and [Config.StartView] use. A view in [Map.Views] needs one; a view placed as a row is named by its place in the map when it has none. |
| `Title` | `string` | Title names the view in the header and its panel. Required. |
| `Badge` | `func(*Context[S]) string` | Badge is shown in the view's panel (Follow, Framed, Panel). Optional. |
| `Header` | `func(*Context[S]) []panel.Segment` | Header returns extra header segments, shown after the view title and before the clock, such as "Mode: ON". They are dropped before the title when the terminal is narrow. Optional. |
| `Hints` | `func(*Context[S]) []Hint` | Hints returns extra footer hints, shown with the view's own. Optional. |
| `Mode` | `string` | Mode is the footer's left-hand label, such as "LIVE". It defaults to the Title in capitals. |
| `Tag` | `func(*Context[S]) string` | Tag returns a short muted note for the header, such as "Mode: ON". Empty shows nothing. Header is the way to add several or styled segments. Optional. |
| `Row` | `Option[S]` | Row is the row that opens the view, when the view is placed as a row of a panel: its Label (the view's Title by default), Hotkey, HotkeyHint, Badge or Note. Its action is opening the view, so it sets no Run, Do, Opens or Modal. Optional. |
| `Body` | `func(*Context[S], Size) []string` | Body draws the view. It returns rows for the given body size. Required. |
| `Follow` | `bool` | Follow makes the view a log: Scroll and Tail, drawn in a panel. It needs Body. |
| `Framed` | `bool` | Framed draws the body in a bordered panel with the view's Title, as Panel does with no settings. It needs Body. |
| `Panel` | `*ViewPanel[S]` | Panel draws the body inside a bordered panel, as the status panel and the menu are, instead of edge to edge. Body is then given the panel's inner size, and a scrolling body shows its position in the panel's bottom border. It needs Body. Optional. |
| `Scroll` | `bool` | Scroll makes the engine scroll the body. Body may then return more rows than fit: the dashboard shows a window of them, scrolls it with the Up/Down, PageUp/PageDown, Home and End keys, and draws a scroll bar down the right-hand edge (Body is given one column less to leave room for it, but only when there is something to scroll). It needs Body. The scroll position starts at the top each time the view is opened. |
| `Tail` | `bool` | Tail makes a scrolling body behave like a log being written at the bottom: the view starts at the newest rows (the bottom), Up goes back in time, End returns to the live bottom, and a view scrolled back holds its place while rows are appended instead of drifting. Without it the view starts at the top and scrolls from there. It needs Scroll. |
| `Appended` | `func(*Context[S]) int` | Appended, for a Tail view whose body is a bounded history that drops its oldest rows, returns how many rows have ever been appended (a count that only rises). A view scrolled back uses it to hold its place: without it the engine can only compare the body's length, which does not change once the history is full. It needs Tail. Optional. |
| `Keys` | `func(ctx *Context[S], msg tea.KeyPressMsg) (cmd tea.Cmd, handled bool)` | Keys sees every key press first. Return handled == true to stop the dashboard from acting on it. Optional. |
| `OnEnter` | `func(*Context[S]) tea.Cmd` | OnEnter is called when the view becomes the one on screen: when it is opened, and when the user comes back to it from a view opened over it. Use it to start work that only matters while the view is visible; return the command to run, or nil. Optional. |
| `OnLeave` | `func(*Context[S]) tea.Cmd` | OnLeave is called when the view stops being the one on screen: when the user goes back, opens another view over it, or moves to a page. It is not called when the dashboard quits; stop long-running work with the context given to [Dashboard.Model] or [Dashboard.Run]. Optional. |
| `Session` | `func(*Context[S]) (end func())` | Session is OnEnter and OnLeave in one: it is called when the view becomes the one on screen and returns the function to call when it stops being. Use it for work that starts and stops together, such as asking a backend for more detail only while the view is open. Like OnLeave, it is not called when the dashboard quits. Optional. |

### `ViewPanel`

ViewPanel is the frame of a view whose Body is drawn in a panel. See [View.Panel].

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the panel's top border. It defaults to the view's Title. |
| `Badge` | `func(*Context[S]) string` | Badge is shown in the top border, already styled. It defaults to the view's Badge. |
| `Subtitle` | `string` | Subtitle is shown in the bottom border, before the scroll position when the body scrolls. Optional. |

### `Shortcut`

Shortcut is a global key binding on the dashboard pages.

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key is the key name as Bubble Tea reports it, e.g. "d" or "ctrl+r". It must not collide with a navigation key or a digit. |
| `Label` | `string` | Label is the footer hint text. Leave it empty for no hint. |
| `Display` | `string` | Display is how the key is written in the footer hint, such as "Ctrl+Q" for the key "ctrl+q". It defaults to Key. |
| `Tone` | `Tone` | Tone colours the hint, key and label together, so a dangerous shortcut can look dangerous. The default is the plain key style. |
| `Prio` | `int` | Prio is the hint's priority; lower is dropped first. Default 30. |
| `Scope` | `Scope` | Scope says where the shortcut is active. The default is the dashboard pages. An [ScopeEverywhere] shortcut may use a force-quit key such as "ctrl+q" to replace the default quit; make sure the application still has a way out. |
| `Run` | `func(*Context[S]) tea.Cmd` | Run is called when the key is pressed. |
| `Do` | `func(*Context[S])` | Do is Run for an action with no command to return. |
| `Opens` | `string` | Opens is the ID of a registered [View] to open. |
| `Modal` | `func(*Context[S]) Modal[S]` | Modal returns a pop-up to open. |

### `Hook`

Hook is an addon that reacts to the dashboard's events rather than drawing anything: an addon that samples data on every tick, handles the messages of its own commands, or claims the results of its own tasks. Set at least one field.

| Field | Type | What it does |
| --- | --- | --- |
| `Tick` | `func(*Context[S], time.Time) tea.Cmd` | Tick is called on every refresh tick, for addons that sample data over time. It runs on the update loop and must not block. |
| `Msg` | `func(*Context[S], tea.Msg) tea.Cmd` | Msg is called with every message the dashboard does not recognise, so an addon can handle the messages of its own commands. Every hook sees every such message; return nil to ignore one. |
| `Result` | `func(*Context[S], TaskResult) (handled bool, cmd tea.Cmd)` | Result sees the result of every background task. Return handled == true to stop the other result hooks and the default footer message. |

## Package modal

### `Edit`

Edit is a one-line text editor, optionally masked for passwords. On Enter the trimmed text is checked with Validate, then handed to Submit (which runs on the update loop and closes the modal first) or to Save (which runs in the background while the modal stays open, showing any error inline and closing on success). Set at most one of Submit and Save; with neither, Enter just closes the modal.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the pop-up's top border. |
| `Help` | `string` | Help is explanatory text shown above the input. |
| `Current` | `string` | Current is the present value, shown for reference unless Secret. |
| `Initial` | `string` | Initial prefills the input. |
| `Secret` | `bool` | Secret masks the input and hides Current. |
| `Limit` | `int` | Limit caps the input length. Default 200. |
| `AllowEmpty` | `bool` | AllowEmpty lets an empty value through to Submit or Save. By default an empty value closes the modal with "Nothing was changed." |
| `Pending` | `string` | Pending is the footer message while Save runs. |
| `Validate` | `func(value string) error` | Validate rejects a value with an error shown inline. Optional. |
| `Submit` | `func(ctx *dashygotui.Context[S], value string) tea.Cmd` | Submit receives the accepted value. The modal is already closed. |
| `Save` | `func(ctx context.Context, value string) (message string, err error)` | Save stores the accepted value in the background and returns a message for the footer, or an error to show in the editor. |
| `OnSaved` | `func(ctx *dashygotui.Context[S], message string) tea.Cmd` | OnSaved is called, with the modal already closed, after Save succeeded, with the message Save returned. Return the command to run next, for example one that restarts a server with the new setting; the dashboard refreshes and shows the message as well. Optional. |

### `Confirm`

Confirm is a yes/no confirmation. Enter or y accepts; Esc, n or q cancels.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the pop-up's top border. |
| `Lines` | `[]string` | Lines describe what will happen; they are wrapped to fit. |
| `Accept` | `string` | Accept labels the accepting key, e.g. "Delete". Default "Confirm". The special label "Back" makes the modal an acknowledgement: Enter and Esc both just close it. |
| `OnAccept` | `func(ctx *dashygotui.Context[S]) tea.Cmd` | OnAccept runs, with the modal already closed, when the user accepts. |
| `OnCancel` | `func(ctx *dashygotui.Context[S]) tea.Cmd` | OnCancel runs, with the modal already closed, when the user cancels. Optional. |

### `Typed`

Typed asks the user to type a word out before an irreversible action, so it cannot be confirmed by a stray Enter.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the pop-up's top border. |
| `Lines` | `[]string` | Lines describe what is about to happen. |
| `Word` | `string` | Word is what must be typed, exactly. |
| `Prompt` | `string` | Prompt asks for it, e.g. "Type RESET to continue". |
| `OnConfirm` | `func(ctx *dashygotui.Context[S]) tea.Cmd` | OnConfirm runs, with the modal already closed, when Word was typed. |
| `OnEmpty` | `func(ctx *dashygotui.Context[S]) tea.Cmd` | OnEmpty runs, with the modal already closed, when Enter is pressed on an empty input. Optional; by default an empty input is an error like any other wrong word. |

### `Notice`

Notice is a read-only, scrollable pop-up for a report or a card. It closes on Enter, Esc, Space, b or q and scrolls with the arrow, page, home and end keys. Give it static Title, Badge and Lines, or a Content function when the text depends on the width or on live state.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title and Badge are shown in the top border. |
| `Badge` | `string` |  |
| `Lines` | `[]string` | Lines are the body text; they are wrapped to fit. |
| `Width` | `int` | Width is the desired width. Default 68, or 96 with Content. |
| `Content` | `func(ctx *dashygotui.Context[S], inner int) (title, badge string, lines []string)` | Content, when set, produces the title, badge and lines afresh on every frame for an inner width, and replaces Title, Badge and Lines. |

### `Choice`

Choice is a numbered list the user picks one entry from. Up and down (or j and k) move, a digit jumps to that entry, Enter selects, Esc cancels.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the pop-up's top border. |
| `Help` | `string` | Help is explanatory text shown above the list. |
| `Badge` | `func(ctx *dashygotui.Context[S]) string` | Badge is shown in the pop-up's top border, already styled, such as the current value. Optional. |
| `Intro` | `func(ctx *dashygotui.Context[S], inner int) []string` | Intro returns explanatory lines shown above the list, after Help, for the given inner width. It is called on every frame, so the lines can show live state. Optional. |
| `Items` | `[]Item` | Items are the entries. At least one is needed, from here or from Rows. |
| `Rows` | `func(ctx *dashygotui.Context[S]) []Item` | Rows returns the entries instead of Items, called on every frame and key press, so badges can show live state. Optional. |
| `FooterSelect` | `string` | FooterSelect words the Enter hint; the default is "Select". Optional. |
| `FooterCancel` | `string` | FooterCancel words the Esc hint; the default is "Cancel". Optional. |
| `Cursor` | `int` | Cursor is the highlighted entry; set it to the current value. |
| `OnMove` | `func(ctx *dashygotui.Context[S], index int)` | OnMove is called whenever the highlight moves, for live previews. Optional. |
| `OnSelect` | `func(ctx *dashygotui.Context[S], index int) tea.Cmd` | OnSelect is called, with the modal already closed, when an entry is chosen. |
| `OnCancel` | `func(ctx *dashygotui.Context[S]) tea.Cmd` | OnCancel is called, with the modal already closed, on Esc, for example to undo a live preview. Optional. |

### `Slider`

Slider lets the user pick an integer between Min and Max. Left and down decrease, right and up increase, Home and End jump to the ends, Enter keeps the value and Esc restores the original.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title is shown in the pop-up's top border. |
| `Help` | `string` | Help is explanatory text shown above the slider. |
| `Unit` | `string` | Unit is appended to the value, e.g. "cells". |
| `Min` | `int` | Min, Max and Value are the range and the starting value. |
| `Max` | `int` | Min, Max and Value are the range and the starting value. |
| `Value` | `int` | Min, Max and Value are the range and the starting value. |
| `OnChange` | `func(ctx *dashygotui.Context[S], value int)` | OnChange is called on every change, so the effect shows live. On Esc it is called once more with the original value. Optional. |
| `OnDone` | `func(ctx *dashygotui.Context[S], value int) tea.Cmd` | OnDone is called, with the modal already closed, when Enter keeps the value. Optional. |
| `OnCancel` | `func(ctx *dashygotui.Context[S]) tea.Cmd` | OnCancel is called, with the modal already closed, after Esc restored the original value. Optional. |

## Package stdaddons

### `Toggle`

Toggle is an on/off setting. Opening the row flips it.

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key names the setting in the Store. Required and unique. |
| `Label` | `string` | Label is the row text. Required. |
| `Default` | `bool` | Default is the value until the user changes it. The zero value is off. |
| `OnChange` | `func(ctx *dashygotui.Context[S], on bool) tea.Cmd` | OnChange is called with the new value after it was saved. Optional. |

### `Number`

Number is a whole-number setting, edited in a pop-up that validates the range.

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key names the setting in the Store. Required and unique. |
| `Label` | `string` | Label is the row text. Required. |
| `Unit` | `string` | Unit follows the value in the badge and messages, such as "GB". Optional. |
| `Min` | `int` | Min and Max bound the value, inclusive. Max must be greater than Min. |
| `Max` | `int` | Min and Max bound the value, inclusive. Max must be greater than Min. |
| `Default` | `int` | Default is the value until the user changes it. It should be within the bounds. |
| `Help` | `string` | Help explains the setting in the pop-up. It defaults to a sentence naming the bounds. |
| `OnChange` | `func(ctx *dashygotui.Context[S], n int) tea.Cmd` | OnChange is called with the new value after it was saved. Optional. |

### `Pick`

Pick is a setting chosen from a fixed list, shown in a pop-up with the current choice marked. The Store keeps the chosen item's Label.

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key names the setting in the Store. Required and unique. |
| `Label` | `string` | Label is the row text. Required. |
| `Items` | `[]modal.Item` | Items are the choices; at least one, with distinct labels. An item's Detail explains it under its label. |
| `Default` | `string` | Default is the Label chosen until the user picks. It defaults to the first item. |
| `Help` | `string` | Help explains the setting above the list. Optional. |
| `OnChange` | `func(ctx *dashygotui.Context[S], label string) tea.Cmd` | OnChange is called with the chosen item's Label after it was saved. Optional. |

### `Text`

Text is a text setting, edited in a one-line pop-up. Secret masks the input and hides the value, for tokens and passwords.

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key names the setting in the Store. Required and unique. |
| `Label` | `string` | Label is the row text. Required. |
| `Default` | `string` | Default is the value until the user changes it. Optional. |
| `Help` | `string` | Help explains the setting in the pop-up. Optional. |
| `Secret` | `bool` | Secret masks the input, and shows only "set" or "not set" as the badge. |
| `Limit` | `int` | Limit is the longest value accepted, in characters. The default is 200. |
| `AllowEmpty` | `bool` | AllowEmpty lets the user save an empty value, which clears the setting back to Default. |
| `Validate` | `func(value string) error` | Validate checks a value before it is saved. Optional. |
| `OnChange` | `func(ctx *dashygotui.Context[S], value string) tea.Cmd` | OnChange is called with the new value after it was saved. Optional. |

### `ThemePicker`

ThemePicker is an option that opens a colour palette picker. Moving the highlight previews each palette live; Enter keeps one and saves the choice and Esc restores the previous palette.

| Field | Type | What it does |
| --- | --- | --- |

### `RefreshRatePicker`

RefreshRatePicker is an option that opens a refresh-rate picker. Declare it as an addon.

| Field | Type | What it does |
| --- | --- | --- |

### `SpacingMenu`

SpacingMenu is an option that opens the spacing menu: the four gaps that frame the dashboard, each adjusted with a live slider, plus a reset. Declare it as an addon.

| Field | Type | What it does |
| --- | --- | --- |

### `DisplayPicker`

DisplayPicker is an option that opens a display-mode picker: Auto (detect the Linux console), Unicode, or TTY. Moving the highlight previews each mode live; Enter keeps one and saves it, Esc restores the previous mode. Declare it as an addon.

| Field | Type | What it does |
| --- | --- | --- |

### `BackgroundPicker`

Background registers an option that lets the user choose how the palette's background reaches the terminal, and turn it off: - Terminal: ask the terminal to use the palette's background and foreground. The terminal's own colours come back when the dashboard exits. A transparent terminal usually stays transparent. - Paint: do that, and also paint every cell, for terminals that ignore the request (some multiplexers and older terminals). Painted cells are opaque. - Off (transparent): never touch the terminal's colours, so a transparent or blurred terminal shows through. BackgroundPicker is an option that lets the user choose how the palette's background reaches the terminal, and turn it off: - Terminal: ask the terminal to use the palette's background and foreground. The terminal's own colours come back when the dashboard exits. A transparent terminal usually stays transparent. - Paint: do that, and also paint every cell, for terminals that ignore the request (some multiplexers and older terminals). Painted cells are opaque. - Off (transparent): never touch the terminal's colours, so a transparent or blurred terminal shows through. Moving the highlight previews each mode live; Enter keeps one and saves it, Esc restores the previous mode. Palettes with no background of their own, and TTY mode, are unaffected by this setting. Declare it as an addon.

| Field | Type | What it does |
| --- | --- | --- |

### `TransparencyToggle`

TransparencyToggle is a one-key switch for a transparent background: on leaves the terminal's own background alone (so a transparent, blurred or wallpapered terminal shows through), off brings the themed background back in the mode that was in use before. It changes the same setting as [BackgroundPicker], so the two always agree. Declare it as an addon.

| Field | Type | What it does |
| --- | --- | --- |

### `LogView`

LogView is a full-screen log view of Buffer and the row that opens it. Place it in a panel's Items in the map: stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs} The view shows the newest entry on top and scrolls with the arrow keys (or j and k), PgUp and PgDn; Home returns to the live top and End jumps to the oldest entry. Scrolled away from the top, it keeps the line being read while new entries arrive. The f key cycles the minimum level shown. Entries filed under a category with [logs.Buffer.In] get a strip of tabs along the top, "All" first; CategoryKey moves along them, each tab keeps its own scroll position, and a tab you are not on shows how many entries arrived since you left it. Divert the application's logger into Buffer before the dashboard starts: restore := buf.Divert(log.Default()) defer restore() The option's badge counts the entries, and the errors among them.

| Field | Type | What it does |
| --- | --- | --- |
| `Buffer` | `*logs.Buffer` | Buffer is the log buffer shown. Required. |
| `Categories` | `[]string` | Categories are tabs always shown, in this order, ahead of any category the buffer meets later. Optional. |
| `CategoryKey` | `string` | CategoryKey is the key that selects the next category tab; the same letter in upper case selects the previous one. Without it the tabs cannot be switched. Optional. |

### `DirBrowser`

DirBrowser is a full-screen directory picker and the row that opens it. Place it in a panel's Items in the map: stdaddons.DirBrowser[dashboard.State]{ Label: "Choose Folder", Start: func(c *dashboard.Ctx) string { return c.State().Home }, OnSelect: func(c *dashboard.Ctx, path string) tea.Cmd { return c.RunTask("folder", "Saving...", func(ctx context.Context) (string, error) { return c.State().SetFolder(ctx, path) }) }, } The view starts at Start (or the user home), lists directories with item count, size and modified time in a left panel (~75%), with key hints on the right (~25%). Enter opens, Ctrl+Enter selects the highlighted directory, Ctrl+B goes to the parent, Ctrl+S searches, Ctrl+H toggles dot-directories, letter and digit keys jump, and Ctrl+D creates a directory. Dot-directories are hidden by default.

| Field | Type | What it does |
| --- | --- | --- |
| `Label` | `string` | Label is the row text. Required. |
| `Title` | `string` | Title is the view title. It defaults to Label. |
| `ID` | `string` | ID names the view for Opens. It defaults to "stdaddons.dirbrowser". |
| `Badge` | `func(*dashygotui.Context[S]) string` | Badge is the row badge. Optional. |
| `Start` | `func(*dashygotui.Context[S]) string` | Start returns the directory to open when the view is entered. Empty or unset uses the user home directory. |
| `OnSelect` | `func(ctx *dashygotui.Context[S], path string) tea.Cmd` | OnSelect runs after the user confirms a directory with Ctrl+Enter. The view has already closed. path is the absolute path of the highlighted directory. Optional. |
| `Pending` | `string` | Pending is the footer message while a create-directory Save runs. Defaults to "Creating the folder...". |

