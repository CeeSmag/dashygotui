# Cookbook: I want to..., so use...

The shortest correct form for each common need. If an addon is longer than these,
look here first. `S` is your state type; in the examples `dashboard` is the
application's package with aliases `Ctx`, `Option`, `View`, `Entry`, `Shortcut`.

## Open a view from a menu row

```go
dashboard.Registry.Add(dashboard.Option{Key: "logs", Section: "tools", Number: 5, Label: "Logs", Opens: "logs"})
```

Not `Run: func(c) tea.Cmd { return OpenView(...) }`. `Opens` is checked at startup.

## Declare a view and the way into it together

```go
dashboard.Registry.Add(dashboard.View{
	ID: "diag", Title: "Diagnostics",
	Entry: &dashboard.Entry{Section: "tools", Number: 16, Hotkey: "d", HotkeyHint: "Diagnostics"},
	Status: func(c *dashboard.Ctx, _ dashygotui.Size) []string { return lines(c) },
	Items: []dashboard.Option{
		{Label: "Run Check", Note: "tests everything", Do: runCheck},
		{Label: "Live Log", Opens: "log"},
	},
})
```

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
dashboard.Registry.Add(
	stdaddons.Toggle[State]{Key: "notify", Section: "settings", Number: 11, Label: "Notifications", Default: true},
	stdaddons.Number[State]{Key: "limit", Section: "settings", Number: 10, Label: "Storage Limit", Unit: "GB", Min: 1, Max: 1000, Default: 50},
	stdaddons.Pick[State]{Key: "region", Section: "settings", Number: 13, Label: "Region", Items: regions},
	stdaddons.Text[State]{Key: "name", Section: "settings", Number: 18, Label: "Display Name"},
)
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

`chart.Stream` / `chart.DualStream`: `Push` on each tick, `Render` on each frame.

## Start in a view

`Config.StartView: []string{"diag", "log"}` (bottom first).

## React to something outside the UI

`dash.Notify()` from any goroutine (not from a callback). The dashboard polls once.

## Keep an addon's own state (a history, a selection, a counter)

`history := dashygotui.Local[[]float64](c, "cpu")` returns a pointer to a value that
belongs to this dashboard (zero on first use); `LocalWith(c, "key", func() *T {...})`
builds it first. Not a variable captured in `init`: that is shared by every dashboard
in the process.

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
