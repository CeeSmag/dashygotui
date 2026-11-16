# dashygotui

A Go library for building terminal dashboards on
[Bubble Tea v2](https://charm.land/), with as little application code as
possible.

An application's dashboard is **a directory of features and one map**. Each
feature is a single file that says only what it does: an option, a widget, a view,
a subpage, a shortcut, a pop-up. The map, one hand-written file, says where
everything goes: the pages, the panels on each page, the rows in each panel. Add a
feature's file and its line in the map; no other file is edited.

![status: pre-release](https://img.shields.io/badge/status-pre--release-orange)

## A dashboard in four files

**`dashboard/dashboard.go`**: the frame: the state, the poll, the short type names.

```go
package dashboard

type State struct{ CPU float64 }

type (
	Map    = dashygotui.Map[State]
	Page   = dashygotui.Page[State]
	Panel  = dashygotui.Panel[State]
	Addon  = dashygotui.Addon[State]
	Ctx    = dashygotui.Context[State]
	Option = dashygotui.Option[State]
	Widget = dashygotui.Widget[State]
)

func Run(ctx context.Context, st dashygotui.Store, mp Map) (dashygotui.ExitInfo, error) {
	d, err := dashygotui.New(dashygotui.Config[State]{
		Title: "My App",
		Map:   mp,
		Store: st,
		Poll:  func(context.Context) (State, error) { return State{CPU: readCPU()}, nil },
	})
	if err != nil {
		return dashygotui.ExitInfo{}, err
	}
	return d.Run(ctx)
}
```

**`dashboard/addons/backup_opt.go`**: a feature. This is the whole file: a value.

```go
package addons

var backupOpt = dashboard.Option{
	Label: "Run Backup",
	Run: func(c *dashboard.Ctx) tea.Cmd { // Run, because it returns a command
		return c.RunTask("backup", "Backing up...", func(context.Context) (string, error) {
			return "Backup finished.", runBackup()
		})
	},
}
```

**`dashboard/addons/addons.go`**: the map. The only file that places anything.

```go
package addons

func Map() dashboard.Map {
	return dashboard.Map{Pages: []dashboard.Page{
		{Title: "Monitor", Panels: []dashboard.Panel{{Widget: cpuWidget}}},
		{Title: "Control", Panels: []dashboard.Panel{
			{Title: "Quick Actions", Items: []dashboard.Addon{backupOpt}}, // [1]
		}},
	}}
}
```

**`main.go`**: passes the map in.

```go
info, err := dashboard.Run(ctx, st, addons.Map())
```

Most features are shorter still: an option that opens a view is `Opens: "logs"`,
one that shows a pop-up is `Modal: ...`, an action with nothing to return is
`Do: ...`, a view placed as a row is opened by that row, and a saved setting is one
`stdaddons.Toggle`, `Number`, `Pick` or `Text`. An option that opens a **window**
is a **subpage**, a plain option in `diag_subpage.go`; everything inside the window
is named after it (`diag_logs_subopt.go`), and the map opens it, fills it and gives
every row its number and hotkey. A big dashboard splits the map by page into
`<name>_map.go` files. Ready-made addons such as the theme picker go in the map directly, with no
file. `dashygotui new <kind> <name>` writes a starter from [`templates/`](templates/)
and prints the line to add to the map.

## What you get

- **Pages, panels and widgets** laid out automatically (1 to N panels, graceful
  stacking on narrow terminals, never overflowing at any size).
- **Numbered options**, numbered by their place in the map: type a number
  anywhere to jump to that option, even on another page.
- **Windows**: a plain option opens a page of panels of its own, which the map
  fills.
- **Modals**: edit, confirm, type-the-word, notice, choice list, slider
  ([`modal`](modal/)).
- **Full-screen views** (with engine-managed scrolling and enter/leave hooks),
  a compact **fallback** for pages on small terminals, background **tasks**, tick
  hooks, ready-made live graphs (`chart.Stream`, `chart.DualStream`), and
  `Dashboard.Notify()` for pushing updates in from outside the UI.
- **Keys**: option hotkeys, global shortcuts (including `ctrl+` combinations that
  work over pop-ups), and an overridable force-quit.
- **Logs**: capture the standard logger and `slog`, and browse them newest first,
  scrollable, with category tabs you switch with a key you choose
  ([`logs`](logs/)).
- **TTY mode**: a conservative symbol set and 16-colour palette, auto-detected
  for the Linux console and switchable at run time.
- **Themes that reach the terminal**: palettes set the terminal's own background
  and foreground, with a switch for the user to turn it off or make it
  transparent.
- **Primitives**: panel, meter, slider, six waveform styles (sparkline, bars,
  line trace, mirrored wave, heat strip, Braille) with adaptive scaling, time
  series, ANSI-aware text helpers ([`panel`](panel/), [`chart`](chart/),
  [`text`](text/)).
- **Themes and spacing** at run time, saved between runs
  ([`stdaddons`](stdaddons/), [`theme`](theme/), [`store`](store/)).
- **Short features.** An option that opens a view, a pop-up or an action is one
  declaration (`Opens`, `Modal`, `Do`); a widget that samples is one value
  (`Tick`); a saved setting is one line (`stdaddons.Toggle`, `Number`, `Pick`,
  `Text`).
- **Deterministic tests** with a fake clock and no sleeping, and a one-call
  `dashtest.Audit` that checks every page and view at every size
  ([`dashtest`](dashtest/)).
- **Rules that stay with the application.** The `dashygotui` command prints the
  design rules for the version you use, lints your addons, scaffolds new ones and
  sets up `AGENTS.md` for people and AI agents (see below).

## Mistakes are loud and early

A feature that is not where the map says is a compile error at that line. And
`dashygotui.New` checks the whole map and reports every problem, with its place in
the map, before the UI starts:

```
invalid dashboard configuration: Control > Quick Actions > row 3: option "Trash"
needs exactly one of Run, Do, Opens and Modal, has 0
```

## Keeping an application on the rails

The point of the library is that an application's dashboard code stays tiny. The
rules that make that so travel with the module, so they reach whoever (or
whatever) works on an application:

```sh
# the version you use; the rules match its API
V=$(go list -m -f '{{.Version}}' github.com/CeeSmag/dashygotui)
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V init      # AGENTS.md, CLAUDE.md, GAPS.md
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V rules     # the charter, rules and cookbook
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V new opt storage   # storage_opt.go from a template, and its line for the map
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V check     # lint the features (exit 1 on a break)
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V check -changed   # ...and: adding a feature edits nothing but the map
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V init -claude    # a Claude Code skill + a check-after-edit hook
```

Add `check` and a `dashtest.Audit(t, cfg)` test to your own `make check`. If the
library cannot do something, record a gap instead of working around it; see
[docs/DESIGN.md](docs/DESIGN.md).

## Try it

```sh
make run        # the showcase: every feature of the library, on six pages
make run-demo   # the small quickstart
go run ./examples/buildwatch   # a second, unrelated app in the short forms
make example    # build the showcase into bin/showcase
make help       # every target, including cross-compiling with `make dist`
```

The [showcase](examples/showcase/) uses every feature; [`examples/README.md`](examples/README.md)
maps each feature to the file that shows it. [`buildwatch`](examples/buildwatch/) is a
second, unrelated application written entirely in the short forms, and
[`demo`](examples/demo/) is the small quickstart. `DASHYGOTUI_TTY=1 go run ./examples/demo`
shows the console symbol set.

## Docs

- [docs/DESIGN.md](docs/DESIGN.md): the design charter, the generic gate, the gap protocol
- [docs/COOKBOOK.md](docs/COOKBOOK.md): "I want X, so use Y"
- [docs/ADDONS.md](docs/ADDONS.md): writing options, widgets, views and modals
- [docs/REFERENCE.md](docs/REFERENCE.md): every field an addon fills in (generated)
- [docs/APP-RULES.md](docs/APP-RULES.md): the rules for applications, checked by `dashygotui check`
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): how it fits together
- [RULES.md](RULES.md): the mandatory architecture and coding rules for the library itself

## Requirements

Go 1.27 or newer. The library is OS-agnostic; platform-specific behaviour
belongs in the application.

## Development

```sh
make check   # format, vet, staticcheck, golangci-lint, govulncheck, race tests,
             # deletion test, template test, consumer simulation, application
             # rules on the examples, API/changelog check, tidy check
```

All contributors, human and AI, follow [RULES.md](RULES.md).

## License

MIT, see [LICENSE](LICENSE).
