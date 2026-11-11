# Writing addons

An addon is **one file** in the application's `addons/` directory. It registers
itself from `init()`. No other file is edited: not the dashboard file, not a
list, not a README.

Start from a template in [`templates/`](../templates/), change the import path
of your `dashboard` package, and pick a number or an order. The
[`examples/demo`](../examples/demo/) app shows every kind.

| You want | File | Register with | Template |
| --- | --- | --- | --- |
| a numbered menu row | `opt_<name>.go` | `Registry.Option` | `opt.go.example` |
| a read-only panel on a page | `widget_<name>.go` | `Registry.Widget` | `widget.go.example` |
| a full-screen page (and the option that opens it) | `view_<name>.go` | `Registry.View` + `Registry.Option` | `view.go.example` |
| a custom pop-up | `modal_<name>.go` | `Registry.Option` that calls `OpenModal` | `modal.go.example` |

Call the registry's methods **directly** (`dashboard.Registry.Option(...)`), not
through a wrapper: the registry records the caller's `file:line` so an error can
point at your file.

## What the foundation file declares

Addons fill slots; they do not create them. The dashboard file declares:

- **Pages**: `Config.Pages` (tab order is list order).
- **Sections**: `Config.Sections`, each on a page. A section is a bordered panel
  of numbered options.

Options target a section by ID (`Section: "settings"`). Widgets target a page by
ID (`Page: "monitor"`). An unknown ID is a startup error.

## Options

```go
dashboard.Registry.Option(dashboard.Option{
	Key:     "storage",   // unique, stable; prefix for Store keys
	Section: "settings",
	Number:  10,          // typed to jump here; unique across the dashboard
	Order:   0,           // optional: sort within the section (default: Number)
	Label:   "Storage Limit",
	Badge:   func(c *dashboard.Ctx) string { return c.Badge("50 GB") },
	Run:     func(c *dashboard.Ctx) tea.Cmd { ... },
})
```

- **Numbers are yours to assign** and are checked: a clash is an error naming
  both files. Typing `1` then `0` jumps to `[10]`.
- **Order** comes only from `Order`, then `Number`, then `Key`. Never from file
  names: Go runs `init()` in file-name order and you must not rely on it.
- `Badge` runs on every frame; keep it cheap. Helpers: `c.OnOff(bool)`,
  `c.Badge(text)`, `c.Count(n, "files")`.

## Widgets

```go
dashboard.Registry.Widget(dashboard.Widget{
	Key: "cpu", Page: "monitor", Order: 10, Title: "CPU", MinHeight: 6,
	Render: func(c *dashboard.Ctx, size dashygotui.Size) []string { ... },
})
```

`Render` returns the **body rows** for the inner size and runs on every frame,
so read from `c.State()`. Panels on a page (sections and widgets together) are
arranged by the page's `Layout` and sorted by `Order`.

A widget that **samples over time** keeps its history in variables declared
inside `init()` (closures), fed by an `OnTick` hook. See
`examples/demo/dashboard/addons/widget_network.go`. Package-level variables are
not allowed in addons.

## The Context

Every callback receives a `*Context[S]`. It is valid **only during that
callback**: do not keep it or use it from another goroutine.

| Need | Use |
| --- | --- |
| the data | `c.State()`, `c.Ready()`, `c.Err()` |
| colours | `c.Theme()` (pass to `panel`, `chart`) |
| a footer message | `c.Flash`, `c.FlashError`, `c.FlashResult(msg, err)` |
| a pop-up | `c.OpenModal(m)`, `c.CloseModal()` |
| another screen | `c.OpenView(id)`, `c.Back()`, `c.GotoPage(id)` |
| work that takes time | `c.RunTask(tag, pending, fn)` |
| fresh data now | `c.Refresh()` |
| a terminal program | `c.Exec(cmd, tag)` |
| saved settings | `c.Store().Get/Set` (namespace with your `Key`) |
| end the program | `c.Quit(reason, message)` |

### Tasks

`RunTask` runs `fn` in the background and reports in the footer (`✓ text` or
`✗ error`), then refreshes the data. Take what the task needs **before**
starting it; the Context is not for other goroutines:

```go
st := c.Store()
return c.RunTask("backup", "Backing up…", func(ctx context.Context) (string, error) {
	return "done", st.Set("backup.last", time.Now())
})
```

To handle results yourself, register `Registry.OnResult` and return
`handled == true`.

### Ending the program

`c.Quit("reset", "message")` ends the dashboard; `Run` returns `ExitInfo`. The
application's `main` then does what the reason asks (uninstall, restart, print),
with the terminal back to normal. The library has no exit-handler registry.

## Persisted settings

Use `c.Store()` with a key starting with your option's `Key`:

```go
var on bool
_, _ = c.Store().Get("notifications.enabled", &on)
err := c.Store().Set("notifications.enabled", !on)
```

The `dashygotui.` key prefix belongs to the library (theme, spacing, refresh
rate).

## Modals

Ready-made in [`modal`](../modal/): `Edit` (with `Validate`, `Submit` or a
background `Save`), `Confirm`, `Typed`, `Notice`, `Choice`, `Slider`.

```go
c.OpenModal(&modal.Confirm[dashboard.State]{
	Title: "Delete?", Lines: []string{"This cannot be undone."}, Accept: "Delete",
	OnAccept: func(c *dashboard.Ctx) tea.Cmd { ... },
})
```

A custom modal implements `Modal[S]` (`HandleKey`, `Render`). Its state lives in
the modal value, never in a package variable. Return only the box from `Render`;
the dashboard centres and clips it.

## Ready-made addons

[`stdaddons`](../stdaddons/) provides a theme picker, the four spacing sliders and a
refresh-rate picker. Each is a one-line addon:

```go
func init() { stdaddons.Theme(dashboard.Registry, "appearance", 20) }
```

## Rules an addon must follow

1. One file; no edits elsewhere. `make deletion-test` removes each addon in turn
   and requires the module to still build.
2. File name `opt_`, `widget_`, `view_` or `modal_` + name.
3. **Never use another addon's functions, types or variables.** Shared code goes
   in the `dashboard` package or the library. A guard test enforces this.
4. No package-level `var` for state. Use closures inside `init()`, or fields of
   a modal or view.
5. Do not run long work inside `Run`, `Render` or `Badge`; use `RunTask`.

## Testing an addon

Drive the whole dashboard with [`dashtest`](../dashtest/): a fake clock and fake
timers, so a test never sleeps.

```go
func TestStorage(t *testing.T) {
	d := dashtest.New(t, dashboard.Config(store.NewMemory()), 140, 40)
	d.Type("10")        // jump to [10]
	d.Press("enter")    // open it
	d.Type("75")
	d.Press("enter")
	d.RequireContains("Storage limit set to 75 GB.")
	d.Tick()            // fire timers: refresh ticks, scheduled polls
}
```

Match menu rows with a whitespace-tolerant pattern such as `\[10\]\s+Storage`:
the number column widens when another addon adds a longer number, and your test
must not break because someone added a file.
