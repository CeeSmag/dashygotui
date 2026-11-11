# Writing addons

An addon is **one file** in the application's `addons/` directory. It registers
itself from `init()`. No other file is edited: not the dashboard file, not a
list, not a README.

Start from a template in [`templates/`](../templates/), change the import path
of your `dashboard` package, and pick a number or an order. The
[`examples/showcase`](../examples/showcase/) app shows every kind and every
feature ([`examples/README.md`](../examples/README.md) maps each to its file);
[`examples/demo`](../examples/demo/) is the small quickstart.

| You want | File | Register with | Template |
| --- | --- | --- | --- |
| a numbered menu row | `opt_<name>.go` | `Registry.Option` | `opt.go.example` |
| a read-only panel on a page | `widget_<name>.go` | `Registry.Widget` | `widget.go.example` |
| a full-screen page (and the option that opens it) | `view_<name>.go` | `Registry.View` + `Registry.Option` | `view.go.example` |
| a custom pop-up | `modal_<name>.go` | `Registry.Option` that calls `OpenModal` | `modal.go.example` |
| a key that is not a menu row | `shortcut_<name>.go` | `Registry.Shortcut` | `shortcut.go.example` |

Call the registry's methods **directly** (`dashboard.Registry.Option(...)`), not
through a wrapper: the registry records the caller's `file:line` so an error can
point at your file.

## What the foundation file declares

Addons fill slots; they do not create them. The dashboard file declares:

- **Pages**: `Config.Pages` (tab order is list order).
- **Sections**: `Config.Sections`, each on a page. A section is a bordered panel
  of numbered options.

Options target a section by ID (`Section: "settings"`). Widgets target a page by
ID (`Page: "monitor"`). An unknown ID is a startup error.

## Options

```go
dashboard.Registry.Option(dashboard.Option{
	Key:     "storage",   // unique, stable; prefix for Store keys
	Section: "settings",
	Number:  10,          // typed to jump here; unique across the dashboard
	Order:   0,           // optional: sort within the section (default: Number)
	Label:   "Storage Limit",
	Badge:   func(c *dashboard.Ctx) string { return c.Badge("50 GB") },
	Run:     func(c *dashboard.Ctx) tea.Cmd { ... },
})
```

- **Numbers are yours to assign** and are checked: a clash is an error naming
  both files. Typing `1` then `0` jumps to `[10]`.
- **Order** comes only from `Order`, then `Number`, then `Key`. Never from file
  names: Go runs `init()` in file-name order and you must not rely on it.
- `Badge` runs on every frame; keep it cheap. Helpers: `c.OnOff(bool)`,
  `c.Badge(text)`, `c.Count(n, "files")`.
- **`Hotkey`** is an optional key that runs the option from any page without
  selecting it: `Hotkey: "l"` to open the logs. The row shows `Logs (l)`. A
  hotkey may not be a navigation key, a digit, or the same as a shortcut or
  another option's hotkey; a clash is a startup error naming both files.

## Widgets

```go
dashboard.Registry.Widget(dashboard.Widget{
	Key: "cpu", Page: "monitor", Order: 10, Title: "CPU", MinHeight: 6,
	Render: func(c *dashboard.Ctx, size dashygotui.Size) []string { ... },
})
```

`Render` returns the **body rows** for the inner size and runs on every frame,
so read from `c.State()`. Panels on a page (sections and widgets together) are
arranged by the page's `Layout` and sorted by `Order`.

A widget that **samples over time** keeps its history in variables declared
inside `init()` (closures), fed by an `OnTick` hook. See
`examples/demo/dashboard/addons/widget_network.go`. Package-level variables are
not allowed in addons.

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

To handle results yourself, register `Registry.OnResult` and return
`handled == true`.

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

Use `c.Store()` with a key starting with your option's `Key`:

```go
var on bool
_, _ = c.Store().Get("notifications.enabled", &on)
err := c.Store().Set("notifications.enabled", !on)
```

The `dashygotui.` key prefix belongs to the library (theme, spacing, refresh
rate).

## Views

A view is a full-screen page opened with `c.OpenView(id)` and left with Back. Its
body is a `Status` panel above `Options`, or your own `Body`.

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
- **`Panel`** draws a `Body` in a bordered panel, like the status and options
  panels, instead of edge to edge: `Panel: &dashygotui.ViewPanel[S]{Title:
  "Live Logs", Subtitle: "newest first"}`. `Body` is then given the panel's inner
  size, and a scrolling body shows `rows 12-40 of 310` in the bottom border.
- **`OptionsTitle`, `OptionsSubtitle`** name the options panel of a view that has
  both `Status` and `Options` ("Options" by default).
- **`Header`** returns extra header segments (`panel.Segment`) shown after the
  view's title and before the clock, such as `Mode: ON`. On a narrow terminal the
  lowest `Prio` goes first; the app title and view title stay.
- **`Hints`** returns extra footer hints (`dashygotui.Hint{Key, Label, Tone,
  Prio}`) for the view's own keys.

To open at a view, or a stack of them, set `Config.StartView` (bottom first); the
last is on screen and Back walks down the rest. Each is opened as if the user had
navigated, so `OnEnter` and `OnLeave` run.

### State that outlives a poll

`State` is replaced by every poll, so it is the wrong place for UI state such as
a pending restart or an open flag. Keep that behind a pointer inside `S`
(`type State struct { ...; UI *UIState }`, set once in `Poll`) or in the addon's
closure. The library does not hold app state for you: `S` is yours. The one thing
it does keep is the scroll position of a `Scroll` view.

## Live graphs

[`chart.Stream`](../chart/stream.go) and `chart.DualStream` are graphs that keep
their own history and adaptive scale, so a widget needs no series, resampling or
scaler code. `Push` a sample on each tick (from an `OnTick` hook) and `Render` on
every frame:

```go
io := chart.NewDualStream("Read", "Write", chart.StreamOpts{})
// OnTick:  io.Push(readRate, writeRate)
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
| `Option.Hotkey` | running one option directly | every page; in a view, only that view's options |
| `Registry.Shortcut` | anything that is not a menu row | its `Scope` |

A **shortcut** has a `Scope`. `ScopePages` (the default) works on the dashboard
pages only. `ScopeEverywhere` works on pages, in views and over pop-ups, and
comes before everything else, including the force-quit keys. That makes it the
right scope for a "stop everything" key:

```go
dashboard.Registry.Shortcut(dashygotui.Shortcut[dashboard.State]{
	Key: "ctrl+q", Label: "Stop", Scope: dashygotui.ScopeEverywhere,
	Run: func(c *dashboard.Ctx) tea.Cmd { return c.Quit("stop", "Stopped.") },
})
```

A shortcut's footer hint can be worded and coloured: `Display: "Ctrl+Q"` writes
the key as people say it, and `Tone` (`ToneDanger`, `ToneWarn`, `ToneInfo`,
`ToneGood`, `ToneMuted`) draws the whole hint in a colour the theme chooses, so it
follows the palette. `Option.HotkeyDisplay` and `Option.HotkeyTone` do the same for
the `(key)` shown after an option's label.

A shortcut may use any key Bubble Tea reports (`ctrl+r`, `f5`, `d`), except a
navigation key or a digit. Only an `ScopeEverywhere` shortcut may take over the
force-quit keys (`ctrl+q`, `ctrl+c`); if you do, keep a way out.

To keep the default quit but intercept it, set `Config.OnForceQuit` (for
instance to open a "Stop and exit?" confirmation). Rebind the built-in keys with
`Config.Keys`.

## Logs

A full-screen UI cannot share the terminal with the standard logger. Capture it
in a [`logs.Buffer`](../logs/) before the dashboard starts, and register the log
view:

```go
// dashboard.go
var Logs = logs.NewBuffer(2000)

// main.go
restore := dashboard.Logs.Divert(log.Default())
defer restore()

// addons/opt_logs.go
func init() { stdaddons.Logs(dashboard.Registry, dashboard.Logs, "actions", 4, stdaddons.Setup{Hotkey: "g"}) }
```

Entries can carry a **category**, and the view then shows tabs along the top.
File a logger's output under one with `buf.In("net")`, which gives an
`io.Writer`, a `Divert` and an slog `Handler`, and register the view with
`stdaddons.LogsWith`:

```go
var NetLog = slog.New(Logs.In("net").Handler(nil))

stdaddons.LogsWith(dashboard.Registry, dashboard.Logs, "actions", 4, stdaddons.LogOptions{
	Categories:  []string{"net", "disk"}, // tabs always shown, in this order
	CategoryKey: "c",                     // c: next tab, C: previous (wraps)
}, stdaddons.Setup{Hotkey: "g"})
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
- The user's choice, made in the `stdaddons.Display` picker, is saved and wins
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

Add the pickers as one-line addons:

```go
func init() { stdaddons.Background(dashboard.Registry, "appearance", 24) }   // choose a mode, with live preview
func init() { stdaddons.Transparency(dashboard.Registry, "appearance", 25) } // a one-key transparent on/off
```

`Transparency` flips between Off and the mode that was in use before, and always
agrees with `Background` because they change the same setting. If you see no
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

Feed them from a closure filled by an `OnTick` hook (see `widget_waves.go` and
`widget_network.go` in the demo). `chart.WaveScaler` keeps the scale steady with
hysteresis; `chart.Resample` maps timestamped samples onto columns.

## Ready-made addons

[`stdaddons`](../stdaddons/) provides a theme picker, the four spacing sliders, a
refresh-rate picker, a display-mode picker and the log view. Each is a one-line
addon; pass a `stdaddons.Setup` to give it a hotkey or an order:

```go
func init() { stdaddons.Theme(dashboard.Registry, "appearance", 20, stdaddons.Setup{Hotkey: "m"}) }
```

## Rules an addon must follow

1. One file; no edits elsewhere. `make deletion-test` removes each addon in turn
   and requires the module to still build.
2. File name `opt_`, `widget_`, `view_`, `modal_` or `shortcut_` + name.
3. **Never use another addon's functions, types or variables.** Shared code goes
   in the `dashboard` package or the library. A guard test enforces this.
4. No package-level `var` for state. Use closures inside `init()`, or fields of
   a modal or view.
5. Do not run long work inside `Run`, `Render` or `Badge`; use `RunTask`.
6. Draw only with the theme's glyphs (`th.G`) and plain ASCII text, so the
   addon works in TTY mode. The demo's tests fail if a TTY frame contains a
   symbol the Linux console font may lack.

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

Match menu rows with a whitespace-tolerant pattern such as `\[10\]\s+Storage`:
the number column widens when another addon adds a longer number, and your test
must not break because someone added a file.

State kept in an addon's `init()` closure (a history for a graph, an alert
counter) lives as long as the process, and is shared by every dashboard built
from that registry. In a test suite that builds many dashboards, such counters
carry over from test to test, so assert on patterns (`Alert #\d+`), not on exact
counts.
