# Architecture

## The idea

A dashboard is a **foundation** plus **addons**. The foundation declares pages
and sections; each addon is one self-registering file. Getting that to work in Go
without global magic or code generation drives most of the design.

```
main.go ──imports──▶ dashboard         pages, sections, the one Registry, Run
        └─blank────▶ dashboard/addons  opt_*.go widget_*.go view_*.go modal_*.go
addons ──imports──▶ dashboard ──imports──▶ dashygotui
dashboard never imports addons
```

An addon's `init()` runs only if its package is imported, so `main` imports the
addons package once, blank. Every file in it then registers with the
`Registry` that `dashboard` declares.

## Packages

| Package | Role |
| --- | --- |
| `dashygotui` (root) | `Config`, `Dashboard`, `Registry`, `Context`, the Bubble Tea model, keys, `Run` |
| `text` | ANSI- and wide-character-aware width, truncation, padding, wrapping |
| `theme` | `Palette`, the symbol sets (`Glyphs`) and the immutable `Theme` built from them |
| `panel` | bordered panels, stacking, joining, overlay, header and footer fitting |
| `chart` | meter, slider, waveforms (sparkline, bars, line trace, mirrored wave, heat strip, Braille), time series, adaptive wave scaler |
| `logs` | a bounded log `Buffer` (std `log` and `slog`) and a scrollable, newest-first `Viewer` |
| `modal` | edit, confirm, typed, notice, choice, slider pop-ups |
| `stdaddons` | opt-in addons: theme picker, spacing sliders, refresh rate, display mode, log view |
| `store` | `Store` implementations: JSON file, in-memory |
| `dashtest` | terminal-free test driver with a fake clock and timers |
| `layout` | page arrangement, spacing and edge insets; pure and non-generic (re-exported by the root as `Layout` and `Spacing`) |

Leaf packages (`text`, `theme`, `panel`, `chart`, `layout`) know nothing
about Bubble Tea or application state and are tested without them.

## Why generics, and where the global lives

`Dashboard[S]` is parameterised by the application's state type `S`, so addons
receive a typed `*Context[S]` and never type-assert. Go cannot declare a generic
package-level variable, so the library cannot own a global registry for an
unknown `S`. The library supplies `Registry[S]`; the **application** declares
the single instance in its dashboard file. That one variable is the only global
in an application; the library has none.

Tests inject their own registry, so nothing leaks between them.

## Lifecycle

1. **`init()`**: each addon calls `Registry.Option/Widget/View/Shortcut/On...`.
   Mistakes are recorded with the caller's `file:line`; nothing panics.
2. **`New(cfg)`**: freezes the registry, then validates everything together:
   registration mistakes, unknown pages and sections, duplicate IDs, reserved
   shortcut keys. All problems are reported at once, wrapped in
   `ErrInvalidConfig`. Panels are collected and sorted by `Order`.
3. **`Run(ctx)`**: starts Bubble Tea. Settings load from the `Store`.

Registration order is irrelevant: Go runs `init()` in file-name order, so the
registry orders panels and options by explicit `Order`, then `Number`/`Key`.

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
  piles up chains.
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

## What changed from the original (CloudVault) engine

| Before | Now |
| --- | --- |
| engine exposed `Config`, `Snapshot`, `SetStorageLimit`, ... | generic `Dashboard[S]`; no domain types |
| package-level mutable globals (gaps, palette, TTY mode) | immutable `Theme` and spacing on the dashboard |
| hand-assigned option numbers could collide silently | collisions are a startup error naming both files |
| the registry lived in the addons package | the library supplies `Registry[S]`; the app owns one |
| modal-kind / view / page enums to extend | interface values and string IDs; nothing central to edit |
| `Updated()` channel from a backend | polling as a command, plus `Dashboard.Notify()` to push; no data races |
| exit-handler registry | `Run` returns `ExitInfo`; `main` acts on it |
| `/proc` TTY detection, `sudo`, `systemd` in the engine | OS-agnostic core; platform code lives in addons |
| guard tests grep source for substrings | guard tests parse the AST |

## Not in v1

Addon-defined pages (pages are declared in the dashboard file), mouse support,
and a CloudVault migration. A third-party chart library
([ntcharts](https://github.com/NimbleMarkets/ntcharts)) is deliberately not a
dependency: it is large, ties the module to a Bubble Tea version, and does not
follow the theme's glyph sets. Its charts can still be used inside any widget,
because a widget's `Render` returns plain strings.
