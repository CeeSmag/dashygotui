# Field reference

Generated from the doc comments of the types an addon author fills in. Do not edit: change the
comment in the code and run `go test -run TestReferenceIsCurrent -update-api .`.
For when to use what, read `COOKBOOK.md`; for the narrative, `ADDONS.md`.

## Package dashygotui

### `Config`

Config describes a dashboard. Registry, Pages and Sections are required; the rest have sensible defaults.

| Field | Type | What it does |
| --- | --- | --- |
| `Title` | `string` | Title names the application in the header and the terminal window title. |
| `Registry` | `*Registry[S]` | Registry holds the addons. Required. Applications pass their one global registry; tests pass a fresh one. |
| `Pages` | `[]Page[S]` | Pages are the dashboard tabs, in tab order. At least one is required. |
| `Sections` | `[]SectionSpec[S]` | Sections are the option containers on the pages. Options and widgets refer to pages and sections declared here by ID. |
| `Poll` | `func(ctx context.Context) (S, error)` | Poll fetches the application state. It is called at Interval, in the background, and once at start. When nil the state stays the zero value of S and nothing is polled. |
| `Interval` | `time.Duration` | Interval is how often the dashboard refreshes. Default one second. The user can change it at run time; the choice is kept in Store. |
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
| `Keys` | `KeyMap` | Keys overrides the default key bindings. Unset bindings keep their defaults. |
| `Spacing` | `*Spacing` | Spacing overrides the default spacing; the user's saved choice in Store wins over both. |
| `AltScroll` | `bool` | AltScroll asks the terminal to turn the mouse wheel into arrow keys while the dashboard is up, which keeps text selection with the terminal. It is off by default. |
| `Clock` | `func() time.Time` | Clock supplies the current time. Default [time.Now]; tests inject their own. |
| `Timer` | `func(d time.Duration, fn func(time.Time) tea.Msg) tea.Cmd` | Timer creates the command that delivers a message after d. Default [tea.Tick]. It is a test seam: the dashtest package substitutes timers it can fire on demand, so tests never sleep. |

### `Page`

Page declares one tab of the dashboard. Pages appear in the order they are listed in [Config.Pages].

| Field | Type | What it does |
| --- | --- | --- |
| `ID` | `string` | ID is the unique name sections and widgets refer to. Required. |
| `Title` | `string` | Title is the tab label. Required. |
| `ShortTitle` | `string` | ShortTitle is used when the terminal is narrower than 100 columns. |
| `TinyTitle` | `string` | TinyTitle is used when the terminal is narrower than 75 columns. |
| `Layout` | `Layout` | Layout arranges the page's panels. The zero value is [LayoutAuto]. |
| `Badge` | `func(*Context[S]) string` | Badge is shown after the title on the active tab. Optional. |
| `Fallback` | `func(*Context[S], Size) []string` | Fallback draws a compact version of the page for terminals too small for the full layout, such as a one-panel summary. It receives the size of the page body and returns its rows; the dashboard draws them in a panel titled with the page's Title. While it is shown, the page's options cannot be selected (the page is read-only) and number keys skip it. Optional; FallbackBelow says when it applies. |
| `FallbackBelow` | `Size` | FallbackBelow is the page body size under which Fallback is used: when the body is narrower than Cols or shorter than Rows. A zero field is ignored. Fallback needs FallbackBelow or FallbackWhen, not both. |
| `FallbackWhen` | `func(Size) bool` | FallbackWhen decides for any rule FallbackBelow cannot say, such as "narrow and short, or very short". It receives the size of the page body. Fallback needs FallbackBelow or FallbackWhen, not both. |

### `SectionSpec`

SectionSpec declares a section: a bordered panel on a page that holds options. Sections are declared in the dashboard file; addons fill them with options by section ID.

| Field | Type | What it does |
| --- | --- | --- |
| `ID` | `string` | ID is the unique name options refer to. Required. |
| `Page` | `string` | Page is the ID of the page the section appears on. Required. |
| `Order` | `int` | Order sorts the section among the page's panels, ascending. |
| `Title` | `string` | Title is shown in the panel's top border. Required. |
| `Subtitle` | `string` | Subtitle is shown in the panel's bottom border. Optional. |
| `Noun` | `string` | Noun is the word counted in the default badge, "option" by default. The badge reads "[3 options]". |
| `Badge` | `func(*Context[S]) string` | Badge overrides the default count badge. Optional. |

### `Option`

Option is one numbered, selectable row in a section: the thing an addon adds to a menu. Register it with [Registry.Option].

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key is a unique, stable identifier such as "storage". Required. It must not start with "dashygotui.", which the library keeps for its own settings. |
| `Section` | `string` | Section is the ID of the section the option appears in, as declared in [Config.Sections]. Required. |
| `Number` | `int` | Number is the number that selects the option from the keyboard: typing it jumps straight to the row, on any page. It must be positive and unique across the whole dashboard. Required. |
| `Order` | `int` | Order sorts the option within its section, ascending. When zero the Number is used. |
| `Label` | `string` | Label is the row text. Required. |
| `Hotkey` | `string` | Hotkey is an optional key that runs the option directly from any dashboard page, without selecting it first: "l" to open the logs, say. It is shown after the label as "(l)". It must not clash with a navigation key, a digit, a shortcut or another option's hotkey. Inside a view's own options it works only in that view. |
| `HotkeyDisplay` | `string` | HotkeyDisplay is how the hotkey is written after the label, such as "Ctrl+L". It defaults to Hotkey. |
| `HotkeyTone` | `Tone` | HotkeyTone colours the hotkey after the label. The default is the label's own colour. |
| `HotkeyHint` | `string` | HotkeyHint, when set, also lists the hotkey in the footer with this wording, written as HotkeyDisplay (or Hotkey) and coloured by HotkeyTone, so a hotkey needs no separate [Shortcut] to be advertised. It is shown on the dashboard pages. Optional. |
| `Badge` | `func(*Context[S]) string` | Badge returns the row's current state, already styled, such as "[ON ✓]". Optional. |
| `Note` | `string` | Note is fixed, muted text shown in place of a badge, drawn "[Note]". It is ignored when Badge is set. Optional. |
| `Run` | `func(*Context[S]) tea.Cmd` | Run is called when the user opens the row. Return the command to run, or nil. |
| `Do` | `func(*Context[S])` | Do is Run for an action with no command to return. |
| `Opens` | `string` | Opens is the ID of a registered [View] to open. It is checked at startup, so a mistyped ID is an error rather than a message at run time. |
| `Modal` | `func(*Context[S]) Modal[S]` | Modal returns a pop-up to open, such as a [modal.Confirm] or a notice. |

### `Entry`

Entry describes the option that opens a view; see [View.Entry].

| Field | Type | What it does |
| --- | --- | --- |
| `Section` | `string` | Section is the ID of the section the option appears in. Required. |
| `Number` | `int` | Number is the option's number, unique across the dashboard. Required. |
| `Order` | `int` | Order sorts the option within its section. When zero the Number is used. |
| `Label` | `string` | Label is the row text. It defaults to the view's Title. |
| `Hotkey` | `string` | Hotkey, HotkeyDisplay, HotkeyTone and HotkeyHint are those of [Option]. |
| `HotkeyDisplay` | `string` |  |
| `HotkeyTone` | `Tone` |  |
| `HotkeyHint` | `string` |  |
| `Badge` | `func(*Context[S]) string` | Badge and Note are those of [Option]. |
| `Note` | `string` |  |

### `Widget`

Widget is a read-only panel on a page: a chart, a gauge, a status readout. Register it with [Registry.Widget].

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key is a unique, stable identifier such as "disk_io". Required. |
| `Page` | `string` | Page is the ID of the page the widget appears on, as declared in [Config.Pages]. Required. |
| `Order` | `int` | Order sorts the widget among the page's panels (sections and widgets together), ascending. |
| `Title` | `string` | Title is shown in the panel's top border. Required. |
| `Subtitle` | `string` | Subtitle is shown in the panel's bottom border. Optional. |
| `MinHeight` | `int` | MinHeight is the height in rows, borders included, the widget would like when panels are stacked. It defaults to 11. |
| `Badge` | `func(*Context[S]) string` | Badge returns the state shown in the top border, already styled. Optional. |
| `Render` | `func(*Context[S], Size) []string` | Render returns the panel's body rows for the given inner size. Rows wider than the width are truncated and missing rows are left blank, but returning exactly the right size is faster and clearer. Required. |

### `View`

View is a full-screen page opened from an option or a shortcut with [Context.OpenView] and left with the Back key. Register it with [Registry.View]. A view draws a header, a body and a footer. The body is, in order of precedence, Body when set, otherwise a status panel (Status) above a menu of numbered rows (Options), either of which may be absent.

| Field | Type | What it does |
| --- | --- | --- |
| `ID` | `string` | ID is the unique name passed to [Context.OpenView]. Required. |
| `Title` | `string` | Title names the view in the header and above the status panel. Required. |
| `Badge` | `func(*Context[S]) string` | Badge is shown beside the title. Optional. |
| `Subtitle` | `string` | Subtitle is shown in the bottom border of the status panel. Optional. |
| `OptionsTitle` | `string` | OptionsTitle names the options panel. It defaults to "Options". |
| `OptionsSubtitle` | `string` | OptionsSubtitle is shown in the bottom border of the options panel. Optional. |
| `Header` | `func(*Context[S]) []panel.Segment` | Header returns extra header segments, shown after the view title and before the clock, such as "Mode: ON". They are dropped before the title when the terminal is narrow. Optional. |
| `Hints` | `func(*Context[S]) []Hint` | Hints returns extra footer hints, shown with the view's own. Optional. |
| `Mode` | `string` | Mode is the footer's left-hand label, such as "LIVE". It defaults to the Title in capitals. |
| `Tag` | `func(*Context[S]) string` | Tag returns a short muted note for the header, such as "Mode: ON". Empty shows nothing. Header is the way to add several or styled segments. Optional. |
| `Status` | `func(*Context[S], Size) []string` | Status returns the rows of the status panel. Optional. |
| `Options` | `func(*Context[S]) []Option[S]` | Options returns the menu rows, called afresh on every frame so badges and availability can change. Numbers need only be unique within the view; a row with no Number is numbered by its position, from 1. |
| `Items` | `[]Option[S]` | Items is Options for a list that never changes: the rows themselves, with badges that may still change. Set Items or Options, not both. |
| `Entry` | `*Entry[S]` | Entry makes the view reachable from a menu: it registers an option that opens this view, with the hotkey and footer hint the entry asks for, so a view and the way into it are one declaration. Optional. |
| `Body` | `func(*Context[S], Size) []string` | Body draws the whole body itself, replacing Status and Options. It returns rows for the given body size. |
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
| `Key` | `string` | Key names the setting in the Store and the option. Required and unique. |
| `Section` | `string` | Section is the section the option appears in. Required. |
| `Number` | `int` | Number is the option's number. Required and unique. |
| `Order` | `int` | Order sorts the option within its section. Optional. |
| `Label` | `string` | Label is the row text. Required. |
| `Hotkey` | `string` | Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option]. |
| `HotkeyDisplay` | `string` |  |
| `HotkeyHint` | `string` |  |
| `Default` | `bool` | Default is the value until the user changes it. The zero value is off. |
| `OnChange` | `func(ctx *dashygotui.Context[S], on bool) tea.Cmd` | OnChange is called with the new value after it was saved. Optional. |

### `Number`

Number is a whole-number setting, edited in a pop-up that validates the range.

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key names the setting in the Store and the option. Required and unique. |
| `Section` | `string` | Section is the section the option appears in. Required. |
| `Number` | `int` | Number is the option's number. Required and unique. |
| `Order` | `int` | Order sorts the option within its section. Optional. |
| `Label` | `string` | Label is the row text. Required. |
| `Hotkey` | `string` | Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option]. |
| `HotkeyDisplay` | `string` |  |
| `HotkeyHint` | `string` |  |
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
| `Key` | `string` | Key names the setting in the Store and the option. Required and unique. |
| `Section` | `string` | Section is the section the option appears in. Required. |
| `Number` | `int` | Number is the option's number. Required and unique. |
| `Order` | `int` | Order sorts the option within its section. Optional. |
| `Label` | `string` | Label is the row text. Required. |
| `Hotkey` | `string` | Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option]. |
| `HotkeyDisplay` | `string` |  |
| `HotkeyHint` | `string` |  |
| `Items` | `[]modal.Item` | Items are the choices; at least one, with distinct labels. An item's Detail explains it under its label. |
| `Default` | `string` | Default is the Label chosen until the user picks. It defaults to the first item. |
| `Help` | `string` | Help explains the setting above the list. Optional. |
| `OnChange` | `func(ctx *dashygotui.Context[S], label string) tea.Cmd` | OnChange is called with the chosen item's Label after it was saved. Optional. |

### `Text`

Text is a text setting, edited in a one-line pop-up. Secret masks the input and hides the value, for tokens and passwords.

| Field | Type | What it does |
| --- | --- | --- |
| `Key` | `string` | Key names the setting in the Store and the option. Required and unique. |
| `Section` | `string` | Section is the section the option appears in. Required. |
| `Number` | `int` | Number is the option's number. Required and unique. |
| `Order` | `int` | Order sorts the option within its section. Optional. |
| `Label` | `string` | Label is the row text. Required. |
| `Hotkey` | `string` | Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option]. |
| `HotkeyDisplay` | `string` |  |
| `HotkeyHint` | `string` |  |
| `Default` | `string` | Default is the value until the user changes it. Optional. |
| `Help` | `string` | Help explains the setting in the pop-up. Optional. |
| `Secret` | `bool` | Secret masks the input, and shows only "set" or "not set" as the badge. |
| `Limit` | `int` | Limit is the longest value accepted, in characters. The default is 200. |
| `AllowEmpty` | `bool` | AllowEmpty lets the user save an empty value, which clears the setting back to Default. |
| `Validate` | `func(value string) error` | Validate checks a value before it is saved. Optional. |
| `OnChange` | `func(ctx *dashygotui.Context[S], value string) tea.Cmd` | OnChange is called with the new value after it was saved. Optional. |

