# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows
[Semantic Versioning](https://semver.org/). Until 1.0, a minor version may change
the API; each such change is listed under **Changed** or **Removed**.

## [0.3.0] - 2026-11-11

Keeps applications on the library's rails: shorter addons, rules that travel with
the module, and checks for both the library and the applications built on it.
Everything is additive except one visible default (see Changed).

### Added

- **Short forms.** `Option.Opens`, `Do`, `Modal` and `Note`; `Shortcut.Opens`, `Do`
  and `Modal` (exactly one of `Run`, `Do`, `Opens`, `Modal`, checked at startup, and
  `Opens` is checked against the registered views). `Option.HotkeyHint` advertises a
  hotkey in the footer.
- **`Registry.Add(...)`** registers any mix of options, shortcuts, views and widgets
  in one call, and the `Addon` interface (an `Addons()` method) lets a type of your
  own stand for several.
- **View declarations.** `View.Entry` (the option that opens the view, with hotkey and
  footer hint), `Items` (a fixed menu, numbered by position), `Follow` (Scroll + Tail
  + panel), `Framed`, `Tag` (a muted header note) and `Session` (start and end
  together).
- **Ready-made settings.** `stdaddons.Toggle`, `Number`, `Pick` and `Text`: a value
  remembered in the `Store`, with badge, pop-up, validation, saving, confirmation and
  an `OnChange` hook.
- **`dashygotui` command** (`cmd/dashygotui`, standard library only):
  `rules` prints the design rules that match the pinned version, `check` lints an
  application's addons (rules A1 to A10, gap markers, allow comments), `new` writes an
  addon from a template, `init` writes AGENTS.md, CLAUDE.md and GAPS.md.
- **`dashtest.Audit`**: one call in an application's tests draws every page and view
  at ten sizes (exact frames, no panic) and in TTY mode (only console-safe
  characters); `WalkOptions` also opens every option. `Dashboard.Outline` describes
  a dashboard's pages, views and options.
- **Docs**: `docs/DESIGN.md` (the charter and the gap protocol), `docs/APP-RULES.md`,
  `docs/COOKBOOK.md`, and a generated `docs/REFERENCE.md` of every field an addon
  fills in.
- **Governance of the library**: an exported-API snapshot (`api/dashygotui.txt`)
  that must change with `CHANGELOG.md`, a generated field reference that must match
  the code, a scan for application vocabulary, `examples/buildwatch` (an unrelated
  second application), a consumer simulation, and a gap issue template.

- **`dashygotui.Local` and `LocalWith`**: state an addon keeps for one dashboard (a
  history, a selection, a counter), instead of a variable captured in `init`, which
  every dashboard in the process shares. `stdaddons.Logs` keeps its viewer this way.
  `make race` now runs the tests in random order.
- **`View.Appended`**: a counter that lets a `Tail` view scrolled back hold its place
  when its body is a bounded history (its length stops changing once full).
- **Startup validation** for the new forms: duplicate numbers in a view's `Items`,
  an option `Key` starting with `ReservedKeyPrefix` ("dashygotui."), and bad
  `stdaddons` settings (empty number range, empty or duplicate `Pick` items, an
  unknown default), reported through the new `Invalid` addon with the call site.
- **`dashygotui check -changed`** applies the one-file rule (C1) to the git change set: a
  change that adds an addon and edits another Go file is reported (advice; `-strict`
  fails). **`dashygotui init -claude`** writes a Claude Code skill and merges a hook that runs
  the check after every edit, keeping the user's other settings.
- **Rule A10** in `dashygotui check`: a variable declared in an addon's `init` and changed
  by a callback (`n++`, `list = append(...)`, `m[k] = v`) is flagged, with
  `dashygotui.Local` as the fix.
- **`dashygotui check` reaches the whole application**: A5, A7 and A8 now apply to every
  file that imports the library, not only to addons, and the rules catch dot-imports,
  `lipgloss.Style{}` literals, `os.Stdout`/`Stderr`/`Stdin`, and escape sequences in
  strings. A docs check requires every Go sample to parse; the examples README counts
  are tested.

### Changed

- A view's footer `Mode` now defaults to its title in capitals (it was `VIEW`).
- The position shown in a panelled view's border is joined to its subtitle with the
  theme's separator, which is ASCII in TTY mode (found by `dashtest.Audit`).
- The examples and templates use the short forms, register with `Registry.Add`, and
  keep their state with `Local`; the example tests no longer depend on test order.
- Showcase tests that worked around process-wide addon state (`oncePerProcess`) are
  gone, and the demo test clears the shared log buffer.

### Fixed

- The showcase's Event Stream numbered every event #2001 once its capped history
  filled (found by running the tests repeatedly); it now counts events ever recorded.

## [0.2.0] - 2026-11-11

Closes the gaps found while moving a real application onto the library. Every
change is additive; v0.1.0 code builds unchanged.

### Added

- **Footer hints you can word and colour.** `Shortcut.Display` writes the key as
  `Ctrl+Q`, and `Shortcut.Tone` (`ToneDanger`, `ToneWarn`, `ToneInfo`, `ToneGood`,
  `ToneMuted`) draws the whole hint in a theme colour. `Option.HotkeyDisplay` and
  `Option.HotkeyTone` do the same for the `(key)` after an option's label.
- **`Page.FallbackWhen func(Size) bool`** for any compact-page rule, such as
  "narrow and short, or very short". `FallbackBelow` stays for the simple case;
  setting both is a startup error.
- **View chrome.** `View.OptionsTitle`, `OptionsSubtitle`, `Header` (extra header
  segments, dropped before the title on narrow terminals) and `Hints` (extra
  footer hints, as `dashygotui.Hint`).
- **`View.Panel`** draws a `Body` inside a bordered panel with a title, a badge
  and a subtitle; a scrolling body shows its position (`rows 12-40 of 310`) in
  the bottom border. `Body` is given the panel's inner size.
- **`Config.StartView`**, a stack of views to open at start, with `OnEnter` and
  `OnLeave` running as if the user had navigated there.
- **`modal.Edit.OnSaved`** chains a follow-up command after a successful `Save`.
- **`modal.Choice`** gains `Badge` (top border), `Intro` (live lines above the
  list), `Rows` (entries recomputed on every frame and key) and `FooterSelect` /
  `FooterCancel`.

### Notes

- Per-dashboard state that must survive polls (a pending restart, a pop-up flag)
  belongs behind a pointer inside `S`; see "State that outlives a poll" in
  `docs/ADDONS.md`.
- An application that keeps its settings elsewhere implements `Store` as an
  adapter; see "Persisted settings".

## [0.1.0] - 2026-11-11

The first release.

### Added

- **Core.** A generic `Dashboard[S]` built from a `Config` and the application's
  single `Registry[S]`. Pages, sections and widgets; numbered options; polling as
  a command; background tasks; hooks (`OnTick`, `OnResult`, `OnMsg`); a `Store`
  interface with JSON-file and in-memory implementations; `Run` returns
  `ExitInfo`.
- **One-file addons.** Options, widgets, views and shortcuts register from
  `init()`. Registration never panics: mistakes, including number collisions, are
  reported together, with `file:line`, when the dashboard is built.
- **Layouts.** Auto, columns, rows, one-over-two, two-over-one and 2x2 grid; four
  spacing gaps; exact output size at any terminal size down to 1x1.
- **Keys.** Number jump across pages, per-option hotkeys, global shortcuts
  (`ScopePages`, `ScopeEverywhere`, including `ctrl+` combinations that work over
  pop-ups), an overridable force-quit, and a configurable `KeyMap`.
- **Pop-ups.** `Edit`, `Confirm`, `Typed`, `Notice`, `Choice`, `Slider`, and the
  `Modal`, `ResultModal` and `ScrollableModal` interfaces for custom ones.
- **Views.** Full-screen views with status, options or a custom body;
  `View.Scroll` for engine-managed scrolling with a scroll bar; `View.Tail` for
  log-style bodies that hold their place; `View.OnEnter` and `OnLeave`.
- **Compact pages.** `Page.Fallback` and `FallbackBelow` draw a read-only
  summary on small terminals.
- **Pushing updates in.** `Dashboard.Notify` and `Dashboard.Attach`; requests made
  during a poll are merged into one more poll.
- **Logs.** The `logs` package: a bounded, concurrency-safe buffer for the
  standard logger and `slog`, a newest-first viewer that holds the reader's place,
  and category tabs switched with a developer-chosen key (`Buffer.In`,
  `stdaddons.LogsWith`).
- **Charts.** Meters, sliders, sparklines, bar graphs, line traces, mirrored
  waves, heat strips and Braille graphs; `WaveScaler`, `Series` and `Resample`;
  and the self-contained `Stream` and `DualStream`.
- **Themes.** Built-in palettes with terminal backgrounds, runtime palette,
  spacing, refresh-rate, glyph and background pickers, a one-key transparent
  switch, and saved settings.
- **TTY mode.** An ASCII-plus-line-drawing symbol set and a 16-colour palette,
  detected for the Linux console (`TERM=linux`) or by `Config.DetectTTY`, and
  switchable at run time.
- **Testing.** The `dashtest` package: a driver with a fake clock and timers, key
  presses, resizes, `Notify`, and screen assertions, with no sleeping.
- **Tooling.** `RULES.md`, architecture guard tests, `make check` (format, vet,
  staticcheck, golangci-lint, govulncheck, race tests, the addon deletion test
  and the template test), CI, addon templates, and two example applications.
