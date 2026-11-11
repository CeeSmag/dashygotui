# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows
[Semantic Versioning](https://semver.org/). Until 1.0, a minor version may change
the API; each such change is listed under **Changed** or **Removed**.

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
