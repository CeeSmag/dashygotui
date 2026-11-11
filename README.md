# dashygotui

A Go library for building terminal dashboards on
[Bubble Tea v2](https://charm.land/), with as little application code as
possible.

An application's dashboard is **one foundation file plus a directory of
addons**. The foundation declares the pages, their layout and the sections on
them. Every addon is a single self-registering file: add the file and it appears;
no other file is edited.

![status: pre-release](https://img.shields.io/badge/status-pre--release-orange)

## A dashboard in three files

**`dashboard/dashboard.go`**: the foundation. It owns the one `Registry`.

```go
package dashboard

type State struct{ CPU float64 }

var Registry = dashygotui.NewRegistry[State]()

type (
	Ctx    = dashygotui.Context[State]
	Option = dashygotui.Option[State]
	Widget = dashygotui.Widget[State]
)

func Run(ctx context.Context, st dashygotui.Store) (dashygotui.ExitInfo, error) {
	d, err := dashygotui.New(dashygotui.Config[State]{
		Title:    "My App",
		Registry: Registry,
		Store:    st,
		Pages: []dashygotui.Page[State]{
			{ID: "monitor", Title: "Monitor"},
			{ID: "control", Title: "Control"},
		},
		Sections: []dashygotui.SectionSpec[State]{
			{ID: "actions", Page: "control", Title: "Quick Actions"},
		},
		Poll: func(context.Context) (State, error) { return State{CPU: readCPU()}, nil },
	})
	if err != nil {
		return dashygotui.ExitInfo{}, err
	}
	return d.Run(ctx)
}
```

**`dashboard/addons/opt_backup.go`**: an addon. This is the whole file.

```go
package addons

func init() {
	dashboard.Registry.Add(dashboard.Option{
		Key: "backup", Section: "actions", Number: 1, Label: "Run Backup",
		Run: func(c *dashboard.Ctx) tea.Cmd { // Run, because it returns a command
			return c.RunTask("backup", "Backing up...", func(context.Context) (string, error) {
				return "Backup finished.", runBackup()
			})
		},
	})
}
```

**`main.go`**: one blank import pulls in every addon.

```go
import _ "myapp/dashboard/addons"
```

Most addons are shorter still: an option that opens a view is `Opens: "logs"`, one
that shows a pop-up is `Modal: ...`, an action with nothing to return is `Do: ...`,
a view and its menu row are one `View` with an `Entry`, and a saved setting is one
`stdaddons.Toggle`, `Number`, `Pick` or `Text`. A widget is the same idea:
`dashboard.Registry.Add(...)` in a `widget_*.go` file puts a read-only panel on a
page. Copy a file from [`templates/`](templates/)
to start.

## What you get

- **Pages, sections and widgets** laid out automatically (1 to N panels, graceful
  stacking on narrow terminals, never overflowing at any size).
- **Numbered options**: type a number anywhere to jump to that option, even on
  another page.
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
- **Short addons.** An option that opens a view, a pop-up or an action is one
  declaration (`Opens`, `Modal`, `Do`); a view and the way into it are one
  (`Entry`); a saved setting is one line (`stdaddons.Toggle`, `Number`, `Pick`,
  `Text`).
- **Deterministic tests** with a fake clock and no sleeping, and a one-call
  `dashtest.Audit` that checks every page and view at every size
  ([`dashtest`](dashtest/)).
- **Rules that stay with the application.** The `dashygotui` command prints the
  design rules for the version you use, lints your addons, scaffolds new ones and
  sets up `AGENTS.md` for people and AI agents (see below).

## Mistakes are loud and early

Addons register from `init()`, where an error cannot be returned, so the registry
records problems with the `file.go:line` of the offending call and
`dashygotui.New` reports them all before the UI starts:

```
invalid dashboard configuration: opt_trash.go:14: option "trash" reuses number 12
(first used at opt_storage.go:11)
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
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V new opt storage   # a new addon from a template
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V check     # lint the addons (exit 1 on a break)
go run github.com/CeeSmag/dashygotui/cmd/dashygotui@$V check -changed   # ...and: adding an addon edits no other file
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
belongs in an application's own addons.

## Development

```sh
make check   # format, vet, staticcheck, golangci-lint, govulncheck, race tests,
             # deletion test, template test, consumer simulation, application
             # rules on the examples, API/changelog check, tidy check
```

All contributors, human and AI, follow [RULES.md](RULES.md).

## License

MIT, see [LICENSE](LICENSE).
