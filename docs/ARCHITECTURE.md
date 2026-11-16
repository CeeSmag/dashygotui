# Architecture

## The idea

A dashboard is a **foundation** plus **addons**. The foundation declares pages
and sections; each addon is one file of values. Getting "add a file and it
appears" without `init()` or global state drives most of the design.

```
main.go ──imports──▶ dashboard         pages, sections, Run
        └─imports──▶ dashboard/addons  opt_*.go widget_*.go view_*.go modal_*.go
                                       doc.go (//go:generate), addons_gen.go (All)
addons ──imports──▶ dashboard ──imports──▶ dashygotui
dashboard never imports addons
```

Go cannot list a package's variables at run time, so `dashygotui gen` (run by
`go generate`) writes `addons_gen.go`: a function `All` that returns every
top-level `var` of the addon files, each wrapped in `At("file.go:line", ...)`.
`main` passes `addons.All()` to the dashboard, which takes the list in
`Config.Addons`. Adding a file and regenerating is the whole change; `dashygotui
check` (rule A11) fails while the list is stale.

## Packages

| Package | Role |
| --- | --- |
| `dashygotui` (root) | `Config`, `Dashboard`, `Addon`, `At`, `Hook`, `Context`, `Local` (per-dashboard addon state), the Bubble Tea model, keys, `Run` |
| `text` | ANSI- and wide-character-aware width, truncation, padding, wrapping |
| `theme` | `Palette`, the symbol sets (`Glyphs`) and the immutable `Theme` built from them |
| `panel` | bordered panels, stacking, joining, overlay, header and footer fitting |
| `chart` | meter, slider, waveforms (sparkline, bars, line trace, mirrored wave, heat strip, Braille), time series, adaptive wave scaler |
| `logs` | a bounded log `Buffer` (std `log` and `slog`) and a scrollable, newest-first `Viewer` |
| `modal` | edit, confirm, typed, notice, choice, slider pop-ups |
| `stdaddons` | opt-in addons: theme picker, spacing sliders, refresh rate, display mode, background, log view, and the one-line settings `Toggle`, `Number`, `Pick`, `Text` |
| `store` | `Store` implementations: JSON file, in-memory |
| `dashtest` | terminal-free test driver with a fake clock and timers, and `Audit` |
| `cmd/dashygotui` | the companion command: `rules`, `check`, `new`, `gen`, `init` (standard library only) |
| `docs`, `templates` | the design documents and addon templates, embedded for the command |
| `internal/guard` | the source-level rules, shared by the library's guard tests and the command |
| `layout` | page arrangement, spacing and edge insets; pure and non-generic (re-exported by the root as `Layout` and `Spacing`) |

Leaf packages (`text`, `theme`, `panel`, `chart`, `layout`) know nothing
about Bubble Tea or application state and are tested without them.

## Why generics, and why there is no global

`Dashboard[S]` is parameterised by the application's state type `S`, so addons
receive a typed `*Context[S]` and never type-assert. An addon is a value of a
generic type (`Option[S]`, `Hook[S]`, ...), and a dashboard is built from a list of
them, so neither the library nor the application needs a package-level registry:
the list is passed in, the way Go code passes its dependencies. A test passes the
addons it needs; two dashboards in one program pass two lists. An addon's own state
lives in `Local`, per dashboard, so nothing leaks between them.

## Lifecycle

1. **`go generate`**: `dashygotui gen` lists the addon values in `addons_gen.go`.
2. **`New(cfg)`**: expands `Config.Addons` (an addon may stand for several) and
   validates everything together: mistakes in addons (each named by the place `At`
   recorded), unknown pages and sections, duplicate IDs, reserved shortcut keys.
   All problems are reported at once, wrapped in `ErrInvalidConfig`; nothing
   panics. Panels are collected and sorted by `Order`. `New` reads the list once.
3. **`Run(ctx)`**: starts Bubble Tea. Settings load from the `Store`.

The order of the list is irrelevant: it follows file names, so panels and options
are ordered by explicit `Order`, then `Number`/`Key`.

## Data flow

```
Poll ──(tea.Cmd)──▶ stateMsg[S] ──▶ Update ──▶ model.state ──▶ View ──▶ Render funcs
                                       ▲
              key / tick / task result ┘
```

- Polling is a `tea.Cmd`, not a goroutine with a channel. The result arrives as a
  message and is applied in `Update`.
- **All model state changes happen in `Update`.** Addon callbacks run there (or
  in `View`, read-only). That is why a `Context` must not leave its callback.
- One poll is in flight at a time, and one timer pending; `Refresh()` never
  piles up chains. A refresh asked for while a poll is in flight repeats that
  poll once when it ends.
- Every poll runs under `Config.PollTimeout`. A poll past its deadline is a failed
  poll; one that ignores its ctx is abandoned (its goroutine ends when the call
  returns, into a buffered channel nobody reads). While the latest poll failed the
  footer shows `STALE`.
- **Pushing in from outside.** `Dashboard.Notify()` hands the running program a
  `notifyMsg` (through the `Sender` that `Run` attaches), which polls at once; a
  request made while a poll is in flight sets a flag and one more poll follows,
  so a change is never lost and requests never pile up. The goroutine that calls
  `Notify` is the application's; the library starts none of its own.
- Code running inside `Update` that cannot return a command (the `OnEnter` and
  `OnLeave` hooks of a view, run by `OpenView` and `Back`) queues it, and
  `Update` batches the queue with its own result.
- Time is injectable (`Config.Clock`, `Config.Timer`), which is what lets
  `dashtest` run without sleeping.

## Rendering

`View` builds exactly `rows` lines of exactly `cols` cells, for every size from
0x0 up:

```
dashScreen  = Inset(terminal gap)  [ header, tab bar, page, footer ]
page        = Inset(page gap)      Arrange(panels, layout mode, section gap)
panel       = section → menu rows  |  widget → its Render rows
modal       = Overlay(box) centred, clipped (keeps the keys and bottom border)
```

- Below 40x10 a "terminal too small" notice replaces the frame.
- `layout.Inset` shrinks a gap whenever it would leave too little room, so no
  setting can overflow a small terminal.
- Layout degrades gracefully: side-by-side panels stack on narrow terminals.
- Width is always measured in cells (`text.Width`), never bytes or runes.
- The mouse is not captured, so the terminal keeps selection and copy.

## Navigation

- Up/down move within a section; left/right move between sections on a page;
  Tab/Shift+Tab switch pages; each page remembers its cursor.
- **Typing a number jumps to that option on any page.** "1" then "2" selects
  `[12]` (a digit that cannot continue the number starts afresh); Enter opens it.
  Nothing opens by accident and there is no timer.
- Numbers are unique across the dashboard, enforced at startup.

## Keys

Dispatch order for a key press: shortcuts with `ScopeEverywhere` (even over a
pop-up and ahead of the force-quit keys, so an application can rebind them);
the force-quit keys (or `Config.OnForceQuit`); the open pop-up; the open view; or
the page. On a page: pages-scope shortcuts, then option hotkeys, then
navigation, then typed digits. Reserved keys (navigation, digits, and for most
shortcuts the force-quit keys) and duplicate keys are startup errors.

## Symbols and colour modes

Every symbol a renderer draws comes from the theme's `Glyphs`, one of two sets:
Unicode (rounded corners, Braille, `●`, `✓`) and TTY (ASCII plus the straight
line-drawing characters the console font has, shade-block graphs). The set is
picked by `Config.Glyphs`: auto-detected from `TERM=linux`, or forced, or chosen
by the user and saved. In TTY mode the 16-colour `console` palette is the
default unless a palette was chosen, and the choice of palette is kept
separate from the palette in effect, so switching back restores the original.

A palette may carry a `Background` and `Foreground`. The model sets them on the
Bubble Tea `View`, which applies them to the terminal and restores them on exit;
`BackgroundPaint` additionally writes the colour into every cell after every
style reset, for terminals that ignore the request.

## Logs

`logs.Buffer` is a mutex-protected ring that implements `io.Writer`, so
`Divert(log.Default())` points the standard logger at it, and `Handler` gives an
`slog` handler. The `Viewer` shows it newest first and anchors the view to an
entry's sequence number, not a row, which is why new entries do not push the line
being read, and why eviction of the anchor degrades gracefully to the oldest
kept entry. Log text is stripped of control characters before drawing.

Categories are a field on `Entry`, set by `Buffer.In(name)`. The viewer keeps one
anchor per category, so tabs have independent scroll positions, and one
"newest seen" sequence number per category, set when you leave it while live, so
inactive tabs can count what is new. The tab strip is drawn by the viewer and
is the first row of `Rows`; the switch key is the developer's.

## Scrolling views and compact pages

A `View` with `Scroll` has its `Body` run, then run again one column narrower if
the result does not fit, so the engine can draw a scroll bar without the body
knowing. The offset is one integer on the model, reset when the view changes,
and clamped on every key and every frame. A `Page.Fallback` is decided from the
page body size, by the same arithmetic the renderer uses, so the key handler and
the renderer always agree on whether options are on screen.

## Settings

Theme, spacing, refresh rate and display mode are changeable at run time and
saved in the `Store` under `dashygotui.*` keys. Pickers **preview without
saving** (`PreviewPalette`, `PreviewSpacing`, `PreviewGlyphMode`) and save only
on confirm, so a cursor move
never writes the settings file. The JSON file store writes atomically
(temp file and rename) with mode 0600, and moves an unreadable file aside
rather than overwriting it.

## Not included

Addon-defined pages (pages are declared in the dashboard file) and mouse support. A
third-party chart library ([ntcharts](https://github.com/NimbleMarkets/ntcharts)) is
deliberately not a dependency: it is large, ties the module to a Bubble Tea version, and
does not follow the theme's glyph sets. Its charts can still be used inside any widget,
because a widget's `Render` returns plain strings.

## Governance: keeping the design on the rails

The library's value is that applications stay small. Four mechanisms keep it so,
for the library and for the applications built on it (the charter is in
`DESIGN.md`):

- **One rule engine.** `internal/guard` holds every source-level rule as a small
  function over parsed Go. The library's own guard tests (`arch_test.go`) and the
  `dashygotui check` command use the same code, so what the library demands of
  itself and of an application cannot drift apart.
- **The command travels with the module.** `cmd/dashygotui` embeds the design
  documents and the addon templates (`docs.FS`, `templates.FS`), so `go run
  ...@<version> rules` always prints rules that match the API an application
  compiles against. `init` writes the instruction files, `new` scaffolds an addon,
  `gen` writes the addon list,
  `check` lints.
- **The library governs itself.** `api/dashygotui.txt` is a snapshot of the exported
  API (a change needs `CHANGELOG.md`: `make api-check`); `docs/REFERENCE.md` is
  generated from the field doc comments and must match; a scan rejects application
  vocabulary in the library; `examples/buildwatch` is an unrelated second
  application that must pass the same checks.
- **The consumer simulation** (`scripts/consumer-test.sh`) builds a throw-away
  application on the working tree, scaffolds every addon kind, requires a clean
  `check`, and requires `check` to fail on a bad addon for each rule.

`dashtest.Audit` covers rendering for an application: every page and view at ten
sizes, exact frames, and only console-safe characters in TTY mode.
