# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows
[Semantic Versioning](https://semver.org/). Until 1.0, a minor version may change
the API; each such change is listed under **Changed** or **Removed**.

## [0.6.0] - 2026-11-16

**A window is a subpage, and the map gives the keys.** A feature says only what it
does: no number and no hotkey, both are the map's. An option that opens a window is
a subpage (`<name>_subpage.go`), everything inside the window is named after it
(`<name>_<element>_sub<kind>.go`), and the map opens the window and fills it. The map
can be split by page. The decision is in `docs/DESIGN.md`. This release breaks the
API; see **Removed** and **Migrating from 0.5**.

### Added

- **`Hotkey`**, the map's way to give a row a key: `Hotkey{Key, Display, Tone, Hint,
  Item}` wraps an option, a view, a window or a ready-made setting.
- **`Page.Row`**: the option that opens a page placed as a row (a window). It is an
  ordinary `Option` with a label and badge and no action; the map gives it its
  action. A window's `Title` defaults to its row's label.
- **Subpages.** New feature kinds `subpage`, `subopt`, `subwidget`, `subview` and
  `submodal`. Rule **A14**: a window opens from a `<name>_subpage.go` option, and its
  elements, named `<name>_<element>_sub<kind>.go`, are placed inside it and nowhere
  else. `dashygotui new subpage <name>` writes one, and `new subopt <subpage>_<name>`
  (and the other sub kinds) an element, refusing one whose subpage does not exist.
- **Map files.** The map may be split by page: `addons.go` keeps `Map()`, and a page
  or window gets `<name>_map.go`, a function returning it. `dashygotui new map
  <name>` writes one. `make deletion-test` allows a deleted feature to break any map
  file.
- Templates `subpage` and `map`; the `page` template is now a tab's behaviour.
- A row that opens a view or a window and sets an action of its own is a startup
  error.

### Changed

- **`View.Row`** replaces `View.Entry`: the row that opens a placed view is an
  `Option`, like any row.
- The examples: Diagnostics is a subpage with its elements named after it, every
  hotkey is given in the map, and the showcase's map is split by page.

### Removed

- **`Option.Hotkey`, `HotkeyDisplay`, `HotkeyTone`, `HotkeyHint`**: use `Hotkey` in the
  map. The `Hotkey` fields of the `stdaddons` values go too.
- **`Entry`**, `Page.Entry` and `View.Entry`: use `Row`.

### Migrating from 0.5

1. Move every option's `Hotkey`, `HotkeyDisplay`, `HotkeyTone` and `HotkeyHint` into
   the map: `dashboard.Hotkey{Key: "s", Display: ..., Tone: ..., Hint: ..., Item:
   storageOpt}`. The same for a `stdaddons` value's `Hotkey`. Add the alias
   `Hotkey = dashygotui.Hotkey[State]`.
2. `Entry: &dashboard.Entry{...}` becomes `Row: dashboard.Option{...}`, without its
   hotkey fields (step 1). Remove the `Entry` alias.
3. Turn each window into a subpage: its row into `<name>_subpage.go`
   (`var diagSubpage = dashboard.Option{Label: "Diagnostics"}`), with its `Session`
   and `Badge` functions; rename what is inside it to `<name>_<element>_sub<kind>.go`;
   give the window `Row: diagSubpage`.
4. Optionally, split a long map: a function per page in `<name>_map.go`, listed in
   `addons.go`.
5. Run `dashygotui check`: A14 names every element placed in the wrong window.

## [0.5.0] - 2026-11-16

**The map.** A feature says only what it does, and one hand-written file, the map
(`addons/addons.go`), says where everything goes: the pages, the panels on each
page and the rows in each panel. Rows are numbered by their place in the map, there
is nothing to generate, and a feature that is not placed does not appear. The
decision is in `docs/DESIGN.md`. This release breaks the API; see **Removed** and
**Migrating from 0.4**.

### Added

- **`Map`** (`Pages`, `Shortcuts`, `Hooks`, `Views`) and **`Config.Map`**.
- **`Panel`**: one widget, or a list of rows (`Items`). Rows are numbered by
  position across the tab pages; `Panel.Number` sets where a panel's count starts.
- **`Page.Panels`** and **`Page.Entry`**. A `Page` placed as a row of a panel is a
  sub-page: the row opens it full-screen with its own panels, its own numbering
  from 1, its own hotkeys, its `Session` and its badge in the header, and Back
  returns. `Context.OpenView` and `Config.StartView` open a sub-page by its `ID`.
- A `View` placed as a row is opened by that row (`Entry` says how the row looks).
  A ready-made addon such as `stdaddons.LogView` is placed the same way; one may
  bring a view, a shortcut or a hook along, registered without a row.
- **`Widget.Tick`**: a widget that samples on every tick is one value, without a
  separate `Hook`.
- Mistakes are reported with their place in the map, such as
  `Control > Quick Actions > row 3: option "Trash" needs exactly one of Run, Do,
  Opens and Modal, has 0`.
- **`dashygotui check`**: rule **A13**, every feature is placed in the map exactly
  once (and an addons package has a map).
- **`dashygotui new`** writes `<name>_<kind>.go` for the kinds `opt`, `widget`,
  `view`, `page`, `modal`, `shortcut` and `hook`, and prints the line that places it
  in `addons.go`. Two new templates: `page` and `hook`.
- A `stdaddons` setting (`Toggle`, `Number`, `Pick`, `Text`) without a `Key`, or
  with one in the library's reserved prefix, is a startup error.

### Changed

- **A1**: feature files are named `<name>_<kind>.go`; the map is `addons.go`.
- **A2**: a feature uses nothing of a sibling feature file, no other application
  package, and of the `dashboard` package only its types. What the application
  knows and does is a method of `State`.
- **C1**: adding a feature may edit the map.
- **`View`** is a full-screen `Body`. A view that was a status panel above a menu
  becomes a sub-page. `View.ID` is optional for a view placed as a row.
- **`Entry`** says only how the row that opens a view or a sub-page looks.
- **`AddonError.Site`** is the place in the map.
- **`Outline`**: a page with no ID is named by its title, sub-pages are listed with
  the views, and `OutlineOption` has no `Key`.
- **`dashygotui init`** writes instructions for the map; its agent hook runs only
  the check.
- **`make deletion-test`** deletes each feature file and requires that only the
  map's lines that placed it break.
- **The examples are on the map.** A ready-made addon such as the theme picker is
  placed in the map directly and needs no file; Diagnostics is a sub-page; widgets
  sample with `Tick`. The showcase keeps its option numbers, except that the
  navigation rows are numbered in the order they are listed.

### Removed

- `Config.Pages`, `Config.Sections`, `Config.Addons` and `SectionSpec`: the map
  replaces them.
- `Option.Key`, `Option.Section`, `Option.Number`, `Option.Order`;
  `Widget.Key`, `Widget.Page`, `Widget.Order`; `Entry.Section`, `Entry.Number`,
  `Entry.Order`: the map places and numbers.
- `View.Status`, `View.Options`, `View.Items`, `View.OptionsTitle`,
  `View.OptionsSubtitle`, `View.Subtitle`: use a sub-page.
- `At`: errors name the place in the map.
- `Section`, `Number` and `Order` on the `stdaddons` values.
- The `dashygotui gen` command, `addons_gen.go`, the addons package's `doc.go`, and
  rule A11: the map is the list.

### Migrating from 0.4

1. Write `addons/addons.go` with `func Map() dashboard.Map`: move the pages from
   `Config.Pages` into `Map.Pages` and turn each `SectionSpec` into a `Panel` of
   its page, with the options that named it as its `Items`, in the order you want
   them on screen. A widget becomes `{Widget: x}` among its page's panels. Use
   `Panel.Number` where a section's numbers started somewhere else than where the
   previous one stopped.
2. Delete `Key`, `Section`, `Number` and `Order` from options and widgets, `Page`
   from widgets, and `Section`, `Number` and `Order` from entries and `stdaddons`
   values. A `stdaddons` setting keeps its `Key`: it is where the setting is saved.
3. Shortcuts go in `Map.Shortcuts` and hooks in `Map.Hooks`; a widget's sampling
   hook can become its `Tick`. A view that no row opens goes in `Map.Views`.
4. A view with `Status` and `Items` becomes a `Page` placed as a row: a widget
   panel for the status and a panel of rows for the items.
5. Rename the files `opt_x.go` to `x_opt.go` (and likewise for every kind), and
   delete `addons_gen.go`, `doc.go` and the `go:generate` line.
6. Give `Config` the map (`Map: addons.Map()`) instead of `Addons: addons.All()`.
7. Run `dashygotui check`. A feature that uses a function or variable of the
   `dashboard` package (A2) needs it as a method of `State`; one the map does not
   place is A13.

## [0.4.2] - 2026-11-16

### Added

- **`Page.Session`** does for a page what `View.Session` does for a view: it is
  called when the page becomes the one on screen and returns the function to call
  when it stops being (another page is chosen, or a view opens over it). Use it to
  ask a backend for detail only while the page is visible. A pop-up does not end
  it; quitting does not call the end function. Additive: existing pages are
  unchanged.
- **`Config.Exec`** is the command that hands the terminal to a program for
  `Context.Exec`, a test seam like `Config.Timer`. The default is `tea.Exec`, so a
  running dashboard behaves as before.
- **`dashtest.Driver.Execs` and `Driver.FinishExec`**: under `dashtest`, a command
  given to `Context.Exec` is recorded instead of run. `Execs` lists the waiting
  command lines (`"vi notes.txt"`) and `FinishExec(err)` ends the oldest with that
  outcome, delivering the `TaskResult` the addon expects. Until now a test could
  not get past the point where the dashboard handed the terminal over.

## [0.4.1] - 2026-11-16

### Changed

- The examples say which graphs follow the clock and which follow refreshes. The
  Network graph spans the last 2 minutes of clock time, so it scrolls at the same
  speed whatever the refresh rate; the CPU Waveforms keep the last 120 refreshes and
  speed up with `+`. Their subtitles ("last 2 minutes, by the clock", "last 120
  refreshes, four styles") and comments now say so. The library is unchanged.

## [0.4.0] - 2026-11-16

**Addons are values.** An addon is an unexported top-level value in its own file;
`dashygotui gen` (run by `go generate`) writes `addons_gen.go`, whose `All()` lists
them, and the application passes that list to `New` in `Config.Addons`. There is no
`init()` and no registry. Adding a feature is still one hand-written file. The
decision and its costs are in `docs/DESIGN.md`. This release breaks the API; see
**Removed** and **Migrating from 0.3**.

### Added

- **`Config.Addons`** takes the dashboard's addons as a list. A dashboard with no
  addons at all is valid, so deleting the last addon file keeps it building.
- **`At(site, addon)`** records where an addon is declared, so startup errors name
  that place; without it an error names the addon's index, such as `Config.Addons[2]`.
- **`Hook`** is an addon with `Tick`, `Msg` and `Result` functions.
- **`stdaddons.ThemePicker`, `RefreshRatePicker`, `SpacingMenu`, `DisplayPicker`,
  `BackgroundPicker`, `TransparencyToggle` and `LogView`**: the ready-made addons as
  values, with optional `Hotkey` and `Order` fields.
- **`dashygotui gen`** writes `addons_gen.go`. `-state` names the state type
  (`<import path>.<Type>`) and is recorded in the file's header, so a later `gen`
  needs no flags; `-all` regenerates every list under a directory. A package without
  a `doc.go` gets one with the `//go:generate` line. `dashygotui new` regenerates the
  list and renames a template's Go names after the addon; the agent hook that `init` writes runs
  `gen -all` before the check.
- **Rules A11 and A12** in `dashygotui check`: every addons package has an
  `addons_gen.go` that matches what `gen` writes now (A11), and no `init()` (A12).
  `doc.go` and `addons_gen.go` are exempt from A1, A2 and the one-file rule C1, and
  `make deletion-test` regenerates the list after deleting an addon.
- **`Config.PollTimeout`** (default `DefaultPollTimeout`, ten seconds; negative for no
  deadline). A poll that runs past it fails with an error wrapping
  `context.DeadlineExceeded` and the next poll starts on schedule, so a poll that
  never returns can no longer freeze the dashboard.
- **Poll failures are visible.** While the latest poll failed, the footer shows
  `STALE` in place of the refresh rate, and says `poll failed: <reason>` once when the
  data goes stale (unless an addon's own message is showing).

### Changed

- **Every `OnMsg` hook (now `Hook.Msg`) sees every message.** Before, the first hook
  that returned a command hid the message from the rest, so the outcome depended on
  the order the hooks were registered in. The commands of all hooks are run together.
- **Rule A3** now says what a package-level variable of an addon file must be: one
  unexported addon literal of a named type, never assigned to.
- **`RegistrationError` is now `AddonError`.** Its `Site` is the place `At` recorded,
  or the addon's index in `Config.Addons`.
- The examples and templates declare their addons as values.

### Fixed

- `Context.Refresh` while a poll is in flight now repeats that poll when it ends,
  as `Dashboard.Notify` already did; before, it returned nil and the screen could show
  data read before the change until the next interval. A task that succeeds and is
  not handled by a hook refreshes the same way.
- A task's "working..." message stays up for as long as the task runs. Before, it
  vanished after 30 seconds, and when two tasks ran, the first one ending removed the
  second one's message.

### Removed

- **`Registry`**, `NewRegistry` and all of its methods (`Add`, `Option`, `Widget`,
  `View`, `Shortcut`, `OnTick`, `OnMsg`, `OnResult`, `Validate`), and
  **`Config.Registry`**. Use `Config.Addons`.
- **`stdaddons.Theme`, `RefreshRate`, `Spacing`, `Display`, `Background`,
  `Transparency`, `Logs`, `LogsWith`**, and their **`Setup`** and **`LogOptions`**
  types. Use the value types above.
- **Rule A10** (state captured in `init`): A12 forbids `init`, and A3 forbids
  assigning to an addon value.

### Migrating from 0.3

1. In each addon file, turn `func init() { dashboard.Registry.Add(x, y) }` into
   `var xOption = x` and `var yView = y` (unexported names, one per value).
   `Registry.OnTick(fn)` becomes `var xTick = dashboard.Hook{Tick: fn}`, and
   `stdaddons.Theme(dashboard.Registry, "s", 20, stdaddons.Setup{Hotkey: "m"})`
   becomes `var themeOption = stdaddons.ThemePicker[dashboard.State]{Section: "s",
   Number: 20, Hotkey: "m"}`.
2. Helpers that lived inside `init` become package-level functions and constants;
   name them after their file so two files do not clash.
3. In the dashboard file, delete `var Registry`, add `type Addon =
   dashygotui.Addon[State]` and `type Hook = dashygotui.Hook[State]`, and take the
   addons as a parameter: `Config{Addons: addons, ...}`.
4. Run `dashygotui gen -state <import path of the dashboard package>.State
   ./dashboard/addons` once; it writes `doc.go` and `addons_gen.go`. Add
   `tool github.com/CeeSmag/dashygotui/cmd/dashygotui` to `go.mod`; from then on
   `go generate ./...` keeps the list current.
5. In `main`, replace the blank import with `addons.All()`, passed to the dashboard.

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
