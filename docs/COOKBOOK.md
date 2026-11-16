# Cookbook: I want to..., so use...

The shortest correct form for each common need. If a feature file is longer than
these, look here first. `S` is your state type; in the examples `dashboard` is the
application's package with aliases `Map`, `Page`, `Panel`, `Ctx`, `Option`, `View`,
`Entry`, `Shortcut`, `Hook`, `Addon`. Each feature is a top-level `var` in its own
file, `<name>_<kind>.go`, and the map (`addons/addons.go`) places it.

## Place a feature

In the map, `addons/addons.go`:

```go
{Title: "Tools", Items: []dashboard.Addon{backupOpt, logsView, diagPage}}, // rows [1] [2] [3]
{Widget: networkWidget},                                                  // a panel
```

Shortcuts go in `Map.Shortcuts`, hooks in `Map.Hooks`, and a view with no row of
its own in `Map.Views`. Rows are numbered by position; `Number: 10` on a panel
starts its count at 10.

## Open a view from a row

Place the view itself as the row: `Items: []dashboard.Addon{logsView}`. Its
`Entry` says how the row looks (`Hotkey`, `HotkeyHint`, `Note`). To open the same
view from a second place, an option with `Opens`:

```go
var logsAgainOpt = dashboard.Option{Label: "Logs", Opens: "logs"}
```

Not `Run: func(c) tea.Cmd { return OpenView(...) }`. `Opens` is checked at startup.

## A page of panels opened from a row: a sub-page

A `Page` placed as a row, in the map:

```go
dashboard.Page{Title: "Diagnostics", Session: diagSession, Entry: &dashboard.Entry{Hotkey: "d"},
	Panels: []dashboard.Panel{
		{Widget: diagHealthWidget},
		{Title: "Tools", Items: []dashboard.Addon{diagCheckOpt, diagLogsOpt}}, // [1] [2]
	}}
```

It numbers its own rows from 1, its rows' hotkeys work only inside it, and Back
returns. What it does apart from its panels (`Session`, `Badge`) lives in
`diag_page.go`.

`Entry` creates the option, hotkey and footer hint. `Items` rows are numbered by
position. `Mode` defaults to the title in capitals.

## Run an action that returns nothing

`Do: func(c *dashboard.Ctx) { c.Flash("done") }`, on an option or a shortcut. Use
`Run` only when you must return a `tea.Cmd`.

## Show a pop-up from a row or a key

`Modal: func(c *dashboard.Ctx) dashboard.Modal { return &modal.Confirm[State]{...} }`

## Save a setting

One line each, remembered in the `Store`, with a badge, a pop-up, validation and a
confirmation:

```go
var notifySetting = stdaddons.Toggle[State]{Key: "notify", Label: "Notifications", Default: true}
var limitSetting = stdaddons.Number[State]{Key: "limit", Label: "Storage Limit", Unit: "GB", Min: 1, Max: 1000, Default: 50}
var regionSetting = stdaddons.Pick[State]{Key: "region", Label: "Region", Items: []modal.Item{{Label: "eu"}, {Label: "us"}}}
var nameSetting = stdaddons.Text[State]{Key: "name", Label: "Display Name"}
```

Each takes an `OnChange` to apply the new value (restart a service, say). To keep
settings elsewhere than the library's file, give `Config.Store` an adapter.

## Run something in the background

`return c.RunTask("tag", "Working...", func(ctx context.Context) (string, error) {...})`.
The message it returns is shown in the footer. Never `go func()`.

## Edit a value and then do something with it

`modal.Edit{Save: ..., OnSaved: func(c, msg) tea.Cmd { return restart(c) }}`.

## Show a log, or a long body

`View{Follow: true, Body: ...}` is a scrolling log that starts at the newest row and
holds your place. `Scroll: true` scrolls from the top; add `Framed: true` for a
panel with the scroll position in its border. `Body` is given the size to fill.

## Show a number that changes: a graph

`chart.Stream` / `chart.DualStream`: `Push` on each tick (the widget's own `Tick`),
`Render` on each frame. The widget stays one value.

## React to every tick, a message of your own, or a task's result

`var sampler = dashboard.Hook{Tick: ..., Msg: ..., Result: ...}`, placed in
`Map.Hooks`: set the ones you need. Every `Msg` sees every message the dashboard
does not handle itself. A widget that only samples needs no hook: `Widget.Tick`.

## Start in a view

`Config.StartView: []string{"diag", "log"}` (bottom first).

## React to something outside the UI

`dash.Notify()` from any goroutine (not from a callback). The dashboard polls once.

## Keep an addon's own state (a history, a selection, a counter)

`history := dashygotui.Local[[]float64](c, "cpu")` returns a pointer to a value that
belongs to this dashboard (zero on first use); `LocalWith(c, "key", func() *T {...})`
builds it first. Not a package-level variable: that is shared by every dashboard in
the process.

## Several alike rows from one file

A small type of the file's own with an `Addons()` method, one value per row:
`var gotoCharts = jump{label: "Jump to Charts", page: "charts"}` (see the
showcase's `goto_opt.go`).

## Test with only some features

`cfg.Map = dashboard.Map{Pages: []dashboard.Page{{Title: "T", Panels: []dashboard.Panel{{Title: "Rows", Items: []dashboard.Addon{backupOpt, restartOpt}}}}}}`
(in the addons package's own tests), or `addons.Map()` for the whole dashboard.

## Use what the application knows

A feature uses, from the `dashboard` package, only types. What it needs of the
application is a field or a method of `State`: `c.State().Logger("net")`,
`c.State().Setup(3)`. Never a function or variable of the `dashboard` package.

`State` is replaced by every poll, so it is the wrong place for UI state.

## Show a key hint

`Shortcut{Key: "ctrl+q", Display: "Ctrl+Q", Tone: dashygotui.ToneDanger, Label: "Stop"}`;
for an option, `Hotkey` plus `HotkeyHint`. Tones follow the theme.

## Colour something

`c.Theme().Green.Render(...)`, `c.Theme().Muted...`, or a `Tone`. Never a colour
literal or `lipgloss.NewStyle`.

## Draw a symbol

`c.Theme().G.Bullet`, `.Cross`, `.Pointer`... Plain ASCII elsewhere.

## Test it

`dashtest.New(t, cfg, cols, rows)` then `Press`, `Type`, `Tick`, `Resize`,
`RequireContains`. Add `dashtest.Audit(t, cfg)` once per application.

## Something is not here

Record a gap (`DESIGN.md`). Do not work around it silently.
