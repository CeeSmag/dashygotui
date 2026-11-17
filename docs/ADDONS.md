# Writing features

A feature is **one file** in the application's `addons/` directory, named
`<name>_<kind>.go`. It declares its addons as unexported top-level values and says
only what they do. The **map**, `addons/addons.go`, places them: it is the only file
that says where anything goes. Adding a feature is its file and its line in the
map; no other file is edited.

`dashygotui new <kind> <name>` writes a starter from [`templates/`](../templates/)
and prints the line to add to the map. The [`examples/showcase`](../examples/showcase/)
app shows every kind and every feature ([`examples/README.md`](../examples/README.md)
maps each to its file); [`examples/demo`](../examples/demo/) is the small
quickstart.

| You want | File | Declare | Place it in the map |
| --- | --- | --- | --- |
| a numbered row | `<name>_opt.go` | an `Option` | in a panel's `Items` |
| a saved setting | `<name>_opt.go` | a `stdaddons.Toggle`, `Number`, `Pick` or `Text` | in a panel's `Items` |
| a read-only panel | `<name>_widget.go` | a `Widget` | as a panel: `{Widget: x}` |
| a full-screen view | `<name>_view.go` | a `View` | in a panel's `Items` (the row opens it), or `Map.Views` |
| a window: panels opened from a row | `<name>_subpage.go` | the `Option` that opens it (no action, no hotkey), and its `Session` and `Badge` | a `Page` with that `Row`, in a panel's `Items`, with its `Panels` |
| a row, widget, view or pop-up inside a window | `<subpage>_<name>_subopt.go`, `_subwidget.go`, `_subview.go`, `_submodal.go` | as its plain kind | inside its subpage's window, and nowhere else |
| what a tab does (badge, fallback, session) | `<name>_page.go` | functions | on the tab in the map |
| the map of one page or window | `<name>_map.go` | a function returning the `Page` | in `Map.Pages`, or a panel's `Items` |
| a custom pop-up | `<name>_modal.go` | an `Option` with a `Modal` | in a panel's `Items` |
| a key that is not a row | `<name>_shortcut.go` | a `Shortcut` | `Map.Shortcuts` |
| a reaction to ticks, messages, results | `<name>_hook.go` | a `Hook` | `Map.Hooks` |

Declare each as `var <name> = <literal>` at the top of the file, one addon per
`var`, with an unexported name (`backupOpt`, `systemWidget`). Before writing
anything, check `COOKBOOK.md`: most features are a few lines.

Helpers a file needs (a function, a constant, a type) are package-level too, so
they share one namespace with every other feature file: name them after the file
(`networkHistory`, not `history`) to keep two files from clashing. A clash is a
compile error, never a silent one.

**A feature is self-contained.** It uses the library and, from the `dashboard`
package, only types: `State` with its fields and methods, and the short type names
(`Ctx`, `Option`...). What the application knows and does (its backend, its
loggers, a computed figure) is a field or method of `State`. Nothing of a sibling
feature file, ever.

## The map

```go
// addons/addons.go
package addons

func Map() dashboard.Map {
	return dashboard.Map{
		Shortcuts: []dashboard.Shortcut{stopShortcut},
		Hooks:     []dashboard.Hook{exportDone},
		Pages: []dashboard.Page{
			{Title: "Monitor", Layout: dashygotui.LayoutAuto, Panels: []dashboard.Panel{
				{Widget: systemWidget},
				{Widget: networkWidget},
			}},
			{Title: "Control", Panels: []dashboard.Panel{
				{Title: "Quick Actions", Subtitle: "things to do", Items: []dashboard.Addon{
					backupOpt,  // [1]
					restartOpt, // [2]
					logsView,   // [3] a row that opens the view
				}},
				{Title: "Settings", Number: 10, Items: []dashboard.Addon{storageOpt, notifyOpt}}, // [10] [11]
			}},
		},
	}
}
```

- **Pages** are the tabs, in order. A page's **panels** are arranged by its
  `Layout`, in order; a panel is one widget or a list of rows.
- **Rows are numbered by position**, across the tab pages in map order. A panel's
  `Number` sets where its count starts; typing `1` then `0` jumps to `[10]`. Two
  rows with one number is a startup error.
- **Hotkeys are the map's**, like numbers: `dashboard.Hotkey{Key: "d", Hint:
  "Diagnostics", Item: …}` gives the row it wraps a key that runs it without
  selecting it (from every tab for a row of a tab, only inside the window for a row
  of a window), shown as `Label (d)`. `Display` writes the key (`"Ctrl+O"`), `Tone`
  colours it and `Hint` lists it in the footer. A feature never has a hotkey.
- A **`View`** placed as a row is opened by that row; its `Row` (an `Option` with no
  action) says how the row reads (`Label`, `Note`, `Badge`).
- A **`Page`** placed as a row is a **window**, opened from a **subpage**: the row
  opens it full-screen with panels of its own, numbered from 1, and Back returns.
  Its `Row` is the subpage's option, a plain option in `<name>_subpage.go`; its
  title defaults to that option's label. Its `Session` and `Badge` work as a tab's
  do. Everything inside it is named after it, `<name>_<element>_sub<kind>.go`, and
  the map places those inside it and nowhere else (rule A14).
- **The map can be split by page.** `addons.go` holds `Map()` and lists the tabs;
  a page or window that is longer than a screen gets its own `<name>_map.go` with
  a function returning it (`func setupPage() dashboard.Page`). Map files place
  features and nothing else: no variables, no behaviour.
- A ready-made addon from `stdaddons` is placed directly, with no file.
- Each feature is placed once. A second way into a view is an option with `Opens`.
- Panel `Title`, `Subtitle`, `Badge` and `Noun` dress a list of rows; on a widget
  panel they replace the widget's own.

Every mistake names its place: `Control > Quick Actions > row 3: option "Trash"
needs exactly one of Run, Do, Opens and Modal, has 0`.

## Options

```go
var storageOpt = dashboard.Option{
	Label: "Storage Limit",
	Badge: func(c *dashboard.Ctx) string { return c.Badge("50 GB") },
	Modal: func(c *dashboard.Ctx) dashboard.Modal { return &modal.Edit[State]{...} },
}
```

(A setting like this one is a single `stdaddons.Number`; see "Ready-made settings".)

- `Badge` runs on every frame; keep it cheap. Helpers: `c.OnOff(bool)`,
  `c.Badge(text)`, `c.Count(n, "files")`.
- Its number and its hotkey are the map's (`dashboard.Hotkey{Key: "s", Item:
  storageOpt}`). A hotkey may not be a navigation key, a digit, or the same as a
  shortcut or another row's hotkey; a clash is a startup error naming both places.

## What an option does

An option and a shortcut say what happens when they are used by setting
**exactly one** of:

| Field | Use it for |
| --- | --- |
| `Do func(*Ctx)` | an action with nothing to return |
| `Opens string` | opening a registered view by ID (checked at startup) |
| `Modal func(*Ctx) Modal` | opening a pop-up |
| `Run func(*Ctx) tea.Cmd` | when you must return a command |

`Note: "text"` draws fixed muted text where a badge would go.

Anything that implements `Addon` (an `Addons()` method returning some) is an
addon too: the ready-made settings in `stdaddons` are written that way, and a file
that declares several alike rows can define a small type of its own that returns
them (see `goto_opt.go` in the showcase).

## Ready-made settings

`stdaddons.Toggle`, `Number`, `Pick` and `Text` are options that edit one value,
remembered in the `Store` under their `Key`, with a badge, a pop-up, validation and
a confirmation; `OnChange` runs after a change is saved:

```go
var storageLimitSetting = stdaddons.Number[dashboard.State]{
	Key: "storage.limit_gb", Label: "Storage Limit",
	Unit: "GB", Min: 1, Max: 1000, Default: 50,
}
```

A setting that makes no sense (an empty number range, a `Pick` with no items or
duplicate labels, a `Key` starting with `dashygotui.`, which the library keeps for its
own settings, or no `Key` at all) is a startup error naming its place in the map,
like any other mistake.

## Widgets

```go
var cpuWidget = dashboard.Widget{
	Title: "CPU", MinHeight: 6,
	Render: func(c *dashboard.Ctx, size dashygotui.Size) []string { ... },
}
```

`Render` returns the **body rows** for the inner size and runs on every frame,
so read from `c.State()`. The map places it as a panel, `{Widget: cpuWidget}`,
among the page's other panels in the order written.

A widget that **samples over time** keeps its history with `dashygotui.Local` (state
that belongs to the dashboard; see "State that outlives a poll"), fed by its own
`Tick`, which runs on every refresh:

```go
var networkWidget = dashboard.Widget{
	Title: "Network",
	Tick: func(c *dashboard.Ctx, now time.Time) tea.Cmd {
		h := networkHistory(c) // dashygotui.Local, see below
		h.add(c.State().NetIn, c.State().NetOut)
		return nil
	},
	Render: func(c *dashboard.Ctx, size dashygotui.Size) []string { ... },
}
```

See `examples/demo/dashboard/addons/network_widget.go`. The only package-level
variables in a feature file are its addon values, and nothing assigns to them: a
variable that changes would be shared by every dashboard in the process.

## The Context

Every callback receives a `*Context[S]`. It is valid **only during that
callback**: do not keep it or use it from another goroutine.

| Need | Use |
| --- | --- |
| the data | `c.State()`, `c.Ready()`, `c.Err()` |
| colours | `c.Theme()` (pass to `panel`, `chart`) |
| a footer message | `c.Flash`, `c.FlashError`, `c.FlashResult(msg, err)` |
| a pop-up | `c.OpenModal(m)`, `c.CloseModal()` |
| another screen | `c.OpenView(id)`, `c.Back()`, `c.GotoPage(id)` |
| work that takes time | `c.RunTask(tag, pending, fn)` |
| fresh data now | `c.Refresh()` (from outside the UI: `Dashboard.Notify()`) |
| a terminal program | `c.Exec(cmd, tag)` |
| saved settings | `c.Store().Get/Set` (namespace with your `Key`) |
| end the program | `c.Quit(reason, message)` |

### Tasks

`RunTask` runs `fn` in the background and reports in the footer (`✓ text` or
`✗ error`), then refreshes the data. Take what the task needs **before**
starting it; the Context is not for other goroutines:

```go
st := c.Store()
return c.RunTask("backup", "Backing up…", func(ctx context.Context) (string, error) {
	return "done", st.Set("backup.last", time.Now())
})
```

To handle results yourself, declare a `Hook` with a `Result` function and return
`handled == true` for your task's tag.

### Updates from outside the UI

Polling covers data you can fetch on a timer. When something else learns of a
change first (a file watcher, a socket reader, another service), have it call
`Dashboard.Notify()`. It is safe from any goroutine and any number of times; the
dashboard polls at once, and requests that arrive during a poll are merged into
one more poll.

```go
d, _ := dashygotui.New(cfg)
go func() { // you own this goroutine: stop it with your context
	for range changes {
		d.Notify()
	}
}()
d.Run(ctx)
```

`Notify` does nothing when the dashboard is not running, and it waits for the
update loop to take the message, so never call it from an addon callback; return
`c.Refresh()` there instead. When you embed `Model()` in a program of your own,
connect it with `Dashboard.Attach(program)`; `dashtest.Driver.Notify` does that
for tests.

### Ending the program

`c.Quit("reset", "message")` ends the dashboard; `Run` returns `ExitInfo`. The
application's `main` then does what the reason asks (uninstall, restart, print),
with the terminal back to normal. The library has no exit-handler registry.

## Persisted settings

Use `c.Store()` with a key of the feature's own, named after it:

```go
var on bool
_, _ = c.Store().Get("notifications.enabled", &on)
err := c.Store().Set("notifications.enabled", !on)
```

The `dashygotui.` key prefix belongs to the library (theme, spacing, refresh
rate).

## Views

A view is a full-screen body: a log, a long text, a table with keys of its own. The
map places it as a row (the row opens it) or in `Map.Views` (opened by a shortcut,
`Opens`, `c.OpenView(id)` or `StartView`); Back leaves it. For a page of panels
opened from a row, use a subpage instead (a `Page` placed as a row).

- **`Scroll: true`** makes the dashboard scroll a `Body` that returns more rows
  than fit: arrows, PgUp/PgDn, Home and End, a scroll bar down the right-hand
  edge and a "Scroll" hint. Your `Body` is given one column less when there is
  something to scroll, so the bar has room. The position starts at the top each
  time the view opens.
- **`Tail: true`** (with `Scroll`) is for a body that grows at the bottom, like
  `tail -f`: the view starts at the newest rows, Up goes back in time, End returns
  to live, and a view scrolled back **holds its place** while rows are appended.
  A `Body` that is called on every message in this mode should be cheap.
- **`OnEnter` / `OnLeave`** run when the view becomes visible and when it stops
  being: opened, left with Back, covered by another view, uncovered again, or
  abandoned by `GotoPage`. They return a `tea.Cmd`, so a view can start a probe
  when it appears and stop it when it goes. They do not run when the dashboard
  quits; stop long work with the context you gave `Run`.
- **`Keys`** sees every key first.
- **`Row`** is how the row that opens the view looks, when the view is placed as
  a row: an `Option` with its hotkey and footer hint, a note or badge, and no
  action. `Label` defaults to the view's title.
- **`Follow`** is a log: `Scroll`, `Tail` and a panel. **`Framed`** draws any `Body` in
  a panel with the view's title. Both need `Body`.
- **`Tag`** adds a short muted note to the header (`"Mode: ON"`).
- **`Session`** is `OnEnter` and `OnLeave` in one: it returns the function to call
  when the view stops being on screen.
- **`Mode`** defaults to the title in capitals.
- **`Appended`** is for a `Tail` view whose body is a bounded history that drops its
  oldest rows: it returns how many rows were ever appended, so a view scrolled back
  holds its place even when the list is full (its length no longer changes).
- **`Panel`** draws a `Body` in a bordered panel instead of edge to edge: `Panel: &dashygotui.ViewPanel[S]{Title:
  "Live Logs", Subtitle: "newest first"}`. `Body` is then given the panel's inner
  size, and a scrolling body shows `rows 12-40 of 310` in the bottom border.
- **`Header`** returns extra header segments (`panel.Segment`) shown after the
  view's title and before the clock, such as `Mode: ON`. On a narrow terminal the
  lowest `Prio` goes first; the app title and view title stay.
- **`Hints`** returns extra footer hints (`dashygotui.Hint{Key, Label, Tone,
  Prio}`) for the view's own keys.

To open at a view, or a stack of them, set `Config.StartView` (bottom first); the
last is on screen and Back walks down the rest. Each is opened as if the user had
navigated, so `OnEnter` and `OnLeave` run.

### State that outlives a poll

`State` is replaced by every poll, so it is the wrong place for UI state such as a
pending restart, a selection or a sampled history. Use `dashygotui.Local`:

```go
cpu := dashygotui.Local[[]float64](c, "cpu_trace")   // zero value on first use
*cpu = append(*cpu, c.State().CPU)

graph := dashygotui.LocalWith(c, "load", func() *chart.Stream { return chart.NewStream(chart.StreamOpts{}) })
```

The value belongs to the dashboard, not to the process. That is the difference from a
package-level variable, which every dashboard in the process shares:
two dashboards (a test, an embedding) then corrupt each other, and tests depend on
their order. The same key with another type is a different value, so name keys after
the addon. Call `Local` on the update loop (hooks, `Render`, `Body`, `Keys`); a function
started with `RunTask` runs elsewhere, so read the value first and hand the task what
it needs.

## Live graphs

[`chart.Stream`](../chart/stream.go) and `chart.DualStream` are graphs that keep
their own history and adaptive scale, so a widget needs no series, resampling or
scaler code. `Push` a sample on each tick (from the widget's `Tick`) and `Render` on
every frame:

```go
io := chart.NewDualStream("Read", "Write", chart.StreamOpts{})
// Hook.Tick: io.Push(readRate, writeRate)
// Render:  return io.Render(c.Theme(), size.Cols, size.Rows)
```

`DualStream` draws the two streams mirrored around a centre line that carries the
labels and the scale (`▲ Read ── [≤1.2M/s] ── ▼ Write`, shortened to the width);
`Stream` draws one, plus `Sparkline` (`[⡀⣀⠤⠒⠂]`) and `ScaleTag`. The scale
follows the samples on screen, so a crest still travelling across keeps the
ceiling up. In TTY mode the glyphs are ASCII. For full control use the
primitives they are built from (`Braille`, `DualWave`, `WaveScaler`, `Resample`).

## Compact pages

A page can offer a compact version for terminals too small for its layout:

```go
{ID: "monitor", Title: "Monitor", Layout: dashygotui.LayoutGrid2x2,
	Fallback:      func(c *Ctx, size dashygotui.Size) []string { ... },
	FallbackBelow: dashygotui.Size{Cols: 60, Rows: 15}},
```

When the page body is narrower than `Cols` or shorter than `Rows` (a zero field is
ignored), the dashboard draws the `Fallback` rows in a panel titled with the page
instead of its panels. While it shows, the page is read-only: its options cannot
be selected and number keys skip it.

For a rule that is not "narrower or shorter", use `FallbackWhen` instead, which
receives the page body size:

```go
FallbackWhen: func(s dashygotui.Size) bool { return (s.Cols < 80 && s.Rows < 22) || s.Rows < 8 },
```

`Fallback` needs exactly one of `FallbackBelow` and `FallbackWhen`; anything else
is a startup error.

## Work that runs only while a page is shown

A page's `Session` works like a view's: it is called when the page becomes the
one on screen and returns the function to call when it stops being.

```go
{ID: "diag", Title: "Diagnostics", Session: func(c *Ctx) func() {
	backend.WantDetail(true)
	return func() { backend.WantDetail(false) }
}},
```

The session ends when the user moves to another page or opens a view over the
page, and a new one starts when they come back. A pop-up over the page does not
end it. The page shown at start begins its session when the dashboard starts,
unless `Config.StartView` covers it. Like a view's, the session is not ended
when the dashboard quits.

## Modals

Ready-made in [`modal`](../modal/): `Edit` (with `Validate`, `Submit` or a
background `Save`), `Confirm`, `Typed`, `Notice`, `Choice`, `Slider`.

```go
c.OpenModal(&modal.Confirm[dashboard.State]{
	Title: "Delete?", Lines: []string{"This cannot be undone."}, Accept: "Delete",
	OnAccept: func(c *dashboard.Ctx) tea.Cmd { ... },
})
```

`Edit.OnSaved(c, message)` runs, with the pop-up closed, after a background `Save`
succeeded, and returns the next command: restart a server with the new setting,
say. `Choice` takes a `Badge` for the top border, an `Intro` for live lines above
the list, `Rows` (instead of `Items`) to recompute the entries on every frame, and
`FooterSelect` / `FooterCancel` to reword the footer.

A custom modal implements `Modal[S]` (`HandleKey`, `Render`). Its state lives in
the modal value, never in a package variable. Return only the box from `Render`;
the dashboard centres and clips it.

## Keys

Three kinds of key, from narrow to wide:

| | Use it for | Where it works |
| --- | --- | --- |
| a number | jumping to a row | every page (typed digits, then Enter) |
| a `Hotkey` in the map | running one row directly | every page; in a window, only that window's rows |
| a `Shortcut` | anything that is not a menu row | its `Scope` |

A **shortcut** has a `Scope`. `ScopePages` (the default) works on the dashboard
pages only. `ScopeEverywhere` works on pages, in views and over pop-ups, and
comes before everything else, including the force-quit keys. That makes it the
right scope for a "stop everything" key:

```go
var stopShortcut = dashygotui.Shortcut[dashboard.State]{
	Key: "ctrl+q", Label: "Stop", Scope: dashygotui.ScopeEverywhere,
	Run: func(c *dashboard.Ctx) tea.Cmd { return c.Quit("stop", "Stopped.") },
}
```

A shortcut's footer hint can be worded and coloured: `Display: "Ctrl+Q"` writes
the key as people say it, and `Tone` (`ToneDanger`, `ToneWarn`, `ToneInfo`,
`ToneGood`, `ToneMuted`) draws the whole hint in a colour the theme chooses, so it
follows the palette. A `Hotkey`'s `Display` and `Tone` do the same for the `(key)`
shown after a row's label.

A shortcut may use any key Bubble Tea reports (`ctrl+r`, `f5`, `d`), except a
navigation key or a digit. Only an `ScopeEverywhere` shortcut may take over the
force-quit keys (`ctrl+q`, `ctrl+c`); if you do, keep a way out.

To keep the default quit but intercept it, set `Config.OnForceQuit` (for
instance to open a "Stop and exit?" confirmation). Rebind the built-in keys with
`Config.Keys`. To flash a footer hint when a key nothing handles is pressed, set
`Config.OnUnboundKey`. On a view or subpage, `q` backs out like Esc; it only
quits from a dashboard tab page.

## Logs

A full-screen UI cannot share the terminal with the standard logger. Capture it
in a [`logs.Buffer`](../logs/) before the dashboard starts, and declare the log
view:

In `dashboard.go`:

```go
var Logs = logs.NewBuffer(2000)
```

In `main.go`:

```go
restore := dashboard.Logs.Divert(log.Default())
defer restore()
```

In the map, as a row:

```go
dashboard.Hotkey{Key: "g", Item: stdaddons.LogView[dashboard.State]{Buffer: dashboard.Logs}}
```

Entries can carry a **category**, and the view then shows tabs along the top.
File a logger's output under one with `buf.In("net")`, which gives an
`io.Writer`, a `Divert` and an slog `Handler`, and give the view its tabs:

```go
var NetLog = slog.New(Logs.In("net").Handler(nil)) // in dashboard.go

stdaddons.LogView[dashboard.State]{ // a row in the map
	Buffer:      dashboard.Logs,
	Categories:  []string{"net", "disk"}, // tabs always shown, in this order
	CategoryKey: "c",                     // c: next tab, C: previous (wraps)
}
```

The strip reads `All  [net]  disk (2)  auth`: exactly one tab, the selected one,
is in brackets and highlighted; a tab you are not on counts the entries that
arrived since you left it. Categories nobody declared get their tab when their
first entry arrives. Each tab keeps its own scroll position, and `f` filters by
level inside it. The developer chooses the key: without `CategoryKey` the tabs
are shown but cannot be switched. Entries added without a category are in
`General`.

The view shows the **newest entry on top**. Scroll with the arrows (or `j`/`k`),
PgUp/PgDn; **Home** returns to the live top and **End** jumps to the oldest.
Scrolled away from the top, the view holds the line you are reading while new
entries arrive, and the badge counts how many are newer. **f** cycles the
minimum level. Both the standard `log` package and `log/slog`
(`buf.Handler(nil)`) are captured; a leading `[ERROR]`, `WARN:` or slog's
`level=` sets the level. Control characters in log text are replaced, so a
message can never move the cursor or restyle the screen.

## Display mode (TTY)

Some terminals, notably the **Linux console**, have a small font and no
rounded corners, `●`, `✓` or Braille. The dashboard has a second symbol set for
them, [`theme.TTYGlyphs`](../theme/glyphs.go): ASCII plus straight line-drawing,
shade-block graphs and the 16-colour palette.

- `Config.Glyphs` is `GlyphsAuto` by default: the TTY set when `TERM=linux` (or
  `DASHYGOTUI_TTY=1`), Unicode otherwise. `GlyphsUnicode` and `GlyphsTTY` force
  one.
- `Config.DetectTTY` replaces the detection when the application can tell better
  (for instance by asking the operating system whether the terminal is a virtual
  console). A forced `GlyphsUnicode` or `GlyphsTTY` still wins over it.
- The user's choice, made in the `stdaddons.DisplayPicker`, is saved and wins
  over detection. `c.TTY()` tells an addon which set is in use.
- **Never hard-code a symbol in an addon.** Use the theme: `th.G.Bullet`,
  `th.G.Up`, `th.G.Down`, `th.G.Check`, `th.G.Cross`, `th.G.Pointer`,
  `th.G.Ellipsis`, `th.G.MeterFill`... and prefer plain ASCII in your own text
  (`...`, not `…`). The charts, panels and modals already do.

## Themes and backgrounds

Every built-in palette except the 16-colour console one sets a `Background` and
`Foreground` (`#RRGGBB`), the default included, so the app has a background out
of the box. The dashboard asks the terminal to use them, so the whole screen
matches the theme, and Bubble Tea restores the terminal's own colours on exit.
Add your own palettes with `Config.Palettes`; leave `Background` empty to keep
the terminal's colour.

**The developer** picks the starting mode with `Config.Background`; **the user**
can change it any time (saved, and it wins over the config):

| Mode | What it does | Transparent terminal |
| --- | --- | --- |
| `BackgroundTerminal` (default) | asks the terminal to use the palette's colours | usually stays transparent |
| `BackgroundPaint` | does that, and also paints every cell, for terminals that ignore the request | opaque |
| `BackgroundOff` (`BackgroundTransparent`) | never touches the terminal's colours | always transparent |

Place the pickers in the map as rows:

```go
stdaddons.BackgroundPicker[dashboard.State]{},   // choose a mode, with live preview
stdaddons.TransparencyToggle[dashboard.State]{}, // a one-key transparent on/off
```

`TransparencyToggle` flips between Off and the mode that was in use before, and
always agrees with `BackgroundPicker` because they change the same setting. If you see no
background at all, try Paint: some terminals, and multiplexers such as tmux, do
not pass the colour request through.

**Transparent terminals.** Alacritty applies its `window.opacity` to the default
background, which `BackgroundTerminal` changes, so a transparent window should
stay transparent. Cells with an explicit colour, which Paint writes, stay opaque
unless Alacritty's `colors.transparent_background_colors` is on. For a terminal
that is always see-through, use `BackgroundTransparent`. Nothing is applied in
TTY mode.

## Waveforms

[`chart`](../chart/) draws a series of values several ways. All take the values
oldest to newest (right-aligned, newest at the right), a `scale` that fills the
full height, and fall back to ASCII in TTY mode.

| | Draws | Use it for |
| --- | --- | --- |
| `Sparkline` | one row of eighth-blocks | a trend beside a label |
| `HeatStrip` | one row, colour carries the value | many series stacked |
| `BarGraph` | vertical bars, eighth-block tops | a spectrum or histogram |
| `LineTrace` | a thin Braille line | an oscilloscope-style trace |
| `DualWave` | two streams mirrored on an axis | download over upload |
| `Braille` | a filled Braille area | a compact traffic graph |

Feed them from a history kept with `dashygotui.Local` and filled by the widget's
`Tick` (see `waves_widget.go` and `network_widget.go` in the demo). `chart.WaveScaler` keeps the scale steady with
hysteresis; `chart.Resample` maps timestamped samples onto columns.

## Ready-made addons

[`stdaddons`](../stdaddons/) provides a theme picker, the four spacing sliders, a
refresh-rate picker, a display-mode picker, the background and transparency
switches, the log view, and the settings above. Each is placed in the map as a row,
directly; a hotkey is the map's, as for any row:

```go
dashboard.Hotkey{Key: "m", Item: stdaddons.ThemePicker[dashboard.State]{}},
```

## Rules an addon must follow

They are in [APP-RULES.md](APP-RULES.md), numbered, with the check that enforces each
(`dashygotui check`). In short: one file and its line in the map, no edits
elsewhere; the `<name>_<kind>.go` name; nothing of a sibling feature or of the
application but the `dashboard` package's types; every feature placed once; no
package-level or `init`-captured mutable state (use `Local`); no goroutines (use
`RunTask`); theme glyphs and plain ASCII.

## Testing an addon

Drive the whole dashboard with [`dashtest`](../dashtest/): a fake clock and fake
timers, so a test never sleeps.

```go
func TestStorage(t *testing.T) {
	d := dashtest.New(t, dashboard.Config(store.NewMemory()), 140, 40)
	d.Type("10")        // jump to [10]
	d.Press("enter")    // open it
	d.Type("75")
	d.Press("enter")
	d.RequireContains("Storage limit set to 75 GB.")
	d.Tick()            // fire timers: refresh ticks, scheduled polls
}
```

A command run with `c.Exec` is never started under `dashtest`: the driver records it
and waits. `d.Execs()` lists the waiting commands as command lines, and
`d.FinishExec(err)` ends the oldest one as if it had exited with `err` (nil for
success), so the addon receives its `TaskResult` as it would in a terminal:

```go
d.Press("e")                          // edit the notes in the user's editor
if got := d.Execs(); len(got) != 1 || got[0] != "vi notes.txt" {
	t.Fatalf("Execs = %q", got)
}
d.FinishExec(nil)                     // the editor exited cleanly
d.RequireContains("Notes saved.")
```

Match menu rows with a whitespace-tolerant pattern such as `\[10\]\s+Storage`:
the number column widens when another addon adds a longer number, and your test
must not break because someone added a file.

An addon's own state (`dashygotui.Local`) belongs to the dashboard a test builds, so
tests do not inherit each other's counters or histories and can run in any order
(`make race` shuffles them). The application's shared pieces, such as a log buffer
the application owns, are still process-wide: clear them when a test needs a known
start.

Add one `dashtest.Audit(t, cfg)` test per application: it draws every page and view at
ten terminal sizes (exact frames, no panic) and in TTY mode (only console-safe
characters), and with `dashtest.WalkOptions()` opens every option too.
