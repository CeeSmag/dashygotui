# dashygotui design charter

This page is short on purpose. Read it before changing the library or writing an
application on top of it. The detailed rules are in `RULES.md` (the library),
`APP-RULES.md` (applications) and the recipes in `COOKBOOK.md`.

## The one idea

**The complexity lives in the library. The simplicity lives in the application.**

An application declares what its dashboard contains: pages, sections, and, one
file per feature, options, widgets, views, shortcuts and pop-ups. The library does
everything else: layout, keys, scrolling, theming, terminal quirks, polling,
persistence, pop-ups, tests. If an addon is long, something is missing from the
library, not from the addon.

A rough yardstick: a feature that opens a view, saves a setting or shows a pop-up
is **one declaration of a few lines**, not a function.

## The generic gate (what may enter the library)

The library is used by many applications, none of them its centre. A change enters
the library only if **all** of these hold:

1. Two unrelated applications (a build monitor and a media server, say) could use
   it as it is.
2. Its names, docs and examples carry no application's vocabulary. The library
   says "service", "item" and "setting", never what an application's service does.
3. It is expressed in what the user sees or does (a panel, a key, a setting), not
   in what one application computes.
4. It is additive, or the breaking change is called out in `CHANGELOG.md`.

A feature that fails the gate belongs in the application's `dashboard` package or
in one of its addons.

## Where code lives

| It is... | It goes in... |
| --- | --- |
| Pure UI mechanics: layout, scrolling, keys, themes, pop-ups, settings widgets | the library |
| The application's data and the actions on it | an addon file (one feature, one file) |
| Helpers two or more addons share | the application's `dashboard` package |
| Pages and sections | the application's dashboard file |
| Anything specific to an operating system or a product | the application, never the library |

## The gap protocol (what to do when the library cannot do it)

When something the application needs is missing, **stop**. Do not wrap a library
type, copy its code, override its defaults with a workaround, or fork it. Instead:

1. Look in `COOKBOOK.md` and the godoc. Most "gaps" are an existing field.
2. If it is really missing, record it: an issue from the gap template, or an entry
   in the application's `GAPS.md`, saying what was needed, what the shortest API
   would be, and whether a second application would use it.
3. If the application must move on, write the smallest workaround and mark it with
   `// dashygotui:gap <where the gap is recorded>`. `dashygotui check` lists every
   marker and fails on one with no reference, so a workaround can never hide.
4. When a release closes the gap, delete the workaround and the marker.

## How the rules are kept

- **They travel with the module.** `go run github.com/CeeSmag/dashygotui/cmd/dashygotui@<version> rules`
  prints the rules that match the version an application pins.
- **They are checked.** `dashygotui check` lints an application's addons (rules A1
  to A12, including a stale `addons_gen.go`), `check -changed` applies the one-file rule to a change, and `dashtest.Audit`
  checks the dashboard at every size. All belong in the application's own `make check`.
- **They reach the agent at the moment it matters.** `dashygotui init -claude` installs a
  skill and a hook, so findings arrive after the edit that caused them, not at the end.
- **The library governs itself.** The exported API is snapshotted, the docs must
  mention every exported field, and the library is scanned for application
  vocabulary.
- **The easy way is the right way.** The short forms (`Opens`, `Do`, `Entry`,
  `stdaddons.Toggle`...) exist so that following the design is also the least
  typing.

## Decision: addons are values (v0.4.0)

Up to v0.3 an addon registered itself from `init()` into a package-level
`Registry` that the application declared, and `main` blank-imported the addons
package. That gives "add a file and it appears", but it is not standard Go: the
Google and Uber style guides discourage `init()` and mutable package state, a test
could not load a subset of an application's addons, and one set of addon files
could not be assembled into two different dashboards.

From v0.4.0 an addon is a **value**, and Go's own `go generate` writes the list.

```go
// addons/opt_backup.go: the whole file. No init(), no registry.
package addons

var backup = dashboard.Option{
	Key: "backup", Section: "actions", Number: 1, Label: "Run Backup",
	Do: func(c *dashboard.Ctx) { ... },
}
```

```go
// addons/doc.go: written once by `dashygotui init`, never edited.

// Package addons holds the dashboard's features, one per file.
package addons

//go:generate go tool dashygotui gen -state example.com/app/dashboard.State
```

```go
// addons/addons_gen.go: generated, checked in, never edited.
func All() []dashygotui.Addon[dashboard.State] {
	return []dashygotui.Addon[dashboard.State]{
		dashygotui.At("opt_backup.go:4", backup),
	}
}
```

```go
// main.go: the addons are passed in, not blank-imported.
info, err := dashboard.Run(ctx, st, addons.All())
```

**One feature is still one file.** Adding an addon means adding exactly one
hand-written file. `addons_gen.go` changes with it, but nobody edits it, the way
`go.sum` changes when a dependency is added. Go cannot list a package's variables
at run time, so without `init()` something must write the list; a checked-in
generated file is the standard Go answer (`stringer`, `wire`).

### The contract

- **Every top-level `var` in an addon file is an addon.** Its value is an addon
  literal (an `Option`, `Widget`, `View`, `Shortcut`, `Hook`, or a `stdaddons`
  value) and it is never assigned to. Other package-level state is not allowed in
  an addon file anyway (rule A3), so the generator needs no type checking: a
  `var` that is not an addon is a compile error in `addons_gen.go`, which is loud.
- **Names are unexported.** The addons package exports only `All()`; two files
  that pick the same name are a compile error.
- **`dashygotui gen -state <import path>.<Type>`** writes `addons_gen.go` next to
  the addon files, sorted by file name and line. It is a Go tool directive
  (`tool github.com/CeeSmag/dashygotui/cmd/dashygotui` in the application's
  `go.mod`), so the generator is always the version of the library the
  application builds against.
- **`At(site, addon)`** carries the source position, so a startup error still
  reads `opt_trash.go:14: option "trash" reuses number 12`. A hand-written list
  without `At` works too; its errors name the addon's position in the list.
- **Hooks are values.** `Hook[S]{Tick, Msg, Result}` replaces
  `Registry.OnTick/OnMsg/OnResult`, so an addon that samples on every tick is
  still one file.
- **Ready-made settings are values.** `stdaddons.ThemePicker{Section:
  "settings", Number: 20}` and its siblings (`RefreshRatePicker`, `SpacingMenu`,
  `DisplayPicker`, `BackgroundPicker`, `TransparencyToggle`, `LogView`) replace
  the functions that took a `*Registry`, the way `stdaddons.Toggle` already works.
- **`Config.Addons`** replaces `Config.Registry`. `New` validates the list exactly
  as `Registry.Validate` did. The `Registry` type leaves the public API.
- **Helpers share the package.** What used to be a local inside `init()` (a helper
  closure, a constant) is now package-level, in one namespace with every other
  addon file. Name helpers after their file (`networkHistory`, `exportTag`). A
  clash is a compile error, so it is never silent; it is the one real cost of
  this form, found while converting the examples.

### How the rules stay enforced

| Risk | Guard |
| --- | --- |
| `gen` is forgotten, so an addon silently does not appear | `dashygotui check` regenerates in memory and fails when `addons_gen.go` differs (rule A11). The agent hook that `init` writes runs `gen` before the check. |
| Deleting an addon file breaks the build | `make deletion-test` deletes, regenerates, builds and runs. A person sees the compile error point at `addons_gen.go`; `go generate ./...` fixes it. |
| The one-file check (C1) counts the generated file | C1 counts hand-written files only; `addons_gen.go` and `doc.go` are exempt. |
| A test wants only some addons | It passes its own list: `Config{Addons: []dashygotui.Addon[S]{backup, restart}}`. |

### Phases

Each phase ends with `make check` passing and is one commit.

| Phase | Work | Breaking | Status |
| --- | --- | --- | --- |
| 0 | Engine fixes: a timeout for `Poll`, poll errors shown in the footer, `Context.Refresh` repeats a poll that was already in flight, a second task's "working..." message survives the first one ending, every `OnMsg` hook sees every message | no | done |
| 1 | `Config.Addons`, `At`, `Hook`, value forms of the `stdaddons` helpers, beside the registry | no | done |
| 2 | `dashygotui gen`, rules A11 and A12 in `check`, generated-file exemption in C1, the scripts | no | done |
| 3 | The examples and templates become values | no | done |
| 4 | `Registry` and `init()` registration removed, A10 retired, rules and docs final, v0.4.0 | yes | done |
| 5 | The first production application moves onto v0.4.0 | n/a | |

## Decision: the map (v0.5.0)

In v0.4 every addon said where it goes: `Section: "actions", Number: 3`,
`Page: "monitor", Order: 20`. Nothing showed the whole dashboard, placement was
string IDs checked at startup, numbers were picked by hand and had to be unique,
and a generator had to list the files. One feature that needed a sub-page with two
sections and shared helpers grew into one 464-line file, because the rules gave it
nowhere else to go.

From v0.5.0 **an addon says only what it does, and one hand-written file, the map,
says where everything goes.**

```
dashboard/
  dashboard.go          State (fields and methods), short type names, Poll, Run
  addons/
    addons.go           the map: pages, panels, rows, sub-pages, shortcuts, hooks
    backup_opt.go       one feature per file: <name>_<kind>.go
    network_widget.go
    logs_view.go
    diag_page.go
```

```go
// addons/addons.go: where everything goes. Nothing else places anything.
package addons

// Map is the dashboard's contents.
func Map() dashboard.Map {
	return dashboard.Map{
		Shortcuts: []dashboard.Shortcut{refreshShortcut, stopShortcut},
		Pages: []dashboard.Page{
			{Title: "Monitor", Panels: []dashboard.Panel{
				{Widget: systemWidget},
				{Widget: networkWidget},
			}},
			{Title: "Control", Panels: []dashboard.Panel{
				{Title: "Quick Actions", Items: []dashboard.Addon{
					backupOpt,  // [1]
					restartOpt, // [2]
					logsView,   // [3] a row that opens the view
					dashboard.Page{Title: "Diagnostics", Session: diagSession, Panels: []dashboard.Panel{
						{Widget: diagHealthWidget},
						{Title: "Tools", Items: []dashboard.Addon{diagCheckOpt, diagExportOpt}},
					}}, // [4] a row that opens a sub-page
				}},
				{Title: "Settings", Number: 10, Items: []dashboard.Addon{storageOpt, notifyOpt}}, // [10] [11]
			}},
		},
	}
}
```

```go
// addons/backup_opt.go: what the feature does, and nothing about where it goes.
var backupOpt = dashboard.Option{Label: "Run Backup", Badge: backupBadge, Do: backupRun}
```

### The contract

- **The map places everything.** `Map.Pages` are the tabs. A page's `Panels` are
  drawn by its `Layout`; a panel is one widget or a list of rows (`Items`). A row is
  an `Option`, a `View` (the row opens it), a `Page` (the row opens it as a
  sub-page, with panels of its own and the Back key to return), or a ready-made
  setting from `stdaddons`. `Map.Shortcuts` and `Map.Hooks` are the app-wide keys
  and reactions; `Map.Views` are views with no row of their own, opened by a
  shortcut, `Opens`, `Context.OpenView` or `StartView`.
- **Rows are numbered by position**, across the dashboard's pages in map order; a
  panel's `Number` sets where its count starts. A sub-page numbers its own rows from
  1. Two rows with one number is a startup error.
- **An addon file is self-contained.** It uses the library, and from the
  `dashboard` package only types: `State` (its fields and methods), the short names
  (`Ctx`, `Option`, ...). It uses no other application package, no helper file, and
  nothing of another addon file. What the application knows and does lives on
  `State` (a method such as `s.Setup(n)`, a backend it holds); how anything looks,
  where two applications would want it, belongs in the library.
- **Errors name the place in the map**, such as `Control > Quick Actions > row 3
  ("Run Backup"): option has no Do, Run, Opens or Modal`.
- **No generator.** The map is the list: a feature that is not in the map does not
  appear, and `dashygotui check` reports it. Deleting a feature is deleting its file
  and its line; the compiler points at the line.
- **Each feature is placed once.** A second way into a view is an option with
  `Opens`.

### What changes

| v0.4 | v0.5 |
| --- | --- |
| `Config.Pages`, `Config.Sections`, `SectionSpec`, `Config.Addons` | `Config.Map`: `Map{Pages, Shortcuts, Hooks, Views}`, `Page.Panels`, `Panel` |
| `Option.Key`, `Section`, `Number`, `Order`; `Widget.Key`, `Page`, `Order` | gone: the map's order is the order and position is the number |
| `Entry{Section, Number, Order, ...}` | `Entry{Label, Hotkey, Badge, Note, ...}`: only how the row that opens a view or sub-page looks |
| `View.Status`, `Options`, `Items`, `OptionsTitle`, `OptionsSubtitle`, `Subtitle` | a sub-page: a `Page` placed as a row. A `View` is a full-screen `Body` |
| a separate `Hook` value beside a widget that samples | `Widget.Tick` |
| `At`, `dashygotui gen`, `addons_gen.go`, `doc.go`, rule A11, `make deletion-test` | gone |
| `opt_backup.go` | `backup_opt.go`; kinds `opt`, `widget`, `view`, `page`, `modal`, `shortcut`, `hook` |
| `Section`, `Number`, `Order` on the `stdaddons` values | gone (a setting keeps the `Key` it is saved under) |
| rule 1: one file, nothing else edited | one feature file, and its line in `addons.go` |
| A2: shared helpers go to the dashboard package | an addon uses only `dashboard` types and `State`; nothing else of the application |

### Phases

Each phase ends with `make check` passing and is one commit.

| Phase | Work | Breaking | Status |
| --- | --- | --- | --- |
| 0 | This decision | no | done |
| 1 | The map in the library beside the v0.4 API: `Config.Map`, `Panel`, sub-pages, `Widget.Tick`, numbering by position, map-path errors | no | |
| 2 | `dashygotui check`, `new` and `init` for the new layout; new templates | no | |
| 3 | The three examples move onto the map | no | |
| 4 | The v0.4 placement API, `gen` and A11 removed; rules and docs final; v0.5.0 | yes | |
| 5 | The first production application moves onto v0.5.0 | n/a | |
