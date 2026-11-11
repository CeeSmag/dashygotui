# Examples

Three applications, all built on the same library and tested end to end.

| | What it is | Run it |
| --- | --- | --- |
| [`showcase`](showcase/) | **Uses every feature.** Six pages, 44 options, 10 widgets, 5 views, 4 shortcuts, standard and custom pop-ups, hooks, themes, TTY mode, all from 50 one-file addons. Start here. | `make run` |
| [`buildwatch`](buildwatch/) | **A second, unrelated app** (a CI build monitor) written entirely in the short forms: about 100 lines of addon code. Compare it with the showcase. | `go run ./examples/buildwatch` |
| [`demo`](demo/) | The small quickstart: three pages and a handful of addons. Also the library's integration test. | `make run-demo` |

## Building

```sh
make example     # build the showcase into bin/showcase
make examples    # build the showcase and the demo into bin/
make run         # run the showcase without building a binary
make run-tty     # run it in TTY mode (the Linux console symbol set)
make dist        # cross-compile for Linux, macOS and Windows into bin/dist/
make clean       # delete the binaries those built
make help        # list every target
```

The binary takes a few flags:

```sh
./bin/showcase -h
./bin/showcase -glyphs tty                       # force the console symbol set
./bin/showcase -palette solarized                # a palette that recolours the terminal
./bin/showcase -palette solarized -background paint   # ...and paint every cell, for terminals that ignore it
./bin/showcase -background transparent           # leave the background to the terminal (transparent, blurred...)
./bin/showcase -ephemeral                        # remember nothing between runs
./bin/showcase -view diagnostics,stdaddons.logs   # start inside Diagnostics, then Live Logs
```

Settings are kept in `~/.config/dashygotui-showcase/dashboard.json`. **Reset All
Settings** deletes that file on exit.

## Where each feature is shown

Everything below is in [`showcase/dashboard/`](showcase/dashboard/). The number in
brackets is the option number: type it anywhere to jump there.

| Feature | Where |
| --- | --- |
| Pages, sections, widgets; `Registry` owned by the foundation file | `dashboard.go` |
| Every layout: grid, rows, auto, columns, two-over-one, one-over-two | the page list in `dashboard.go` |
| Tab badges; short and tiny tab titles (resize the terminal) | `Monitor` tab, every page title |
| Section `Noun`, section `Badge`, widget `Badge`, `MinHeight`, `Subtitle` | `Actions` page, `widget_system.go` |
| Custom header; rebound key (Backspace goes back); `OnForceQuit` (ctrl+c asks) | `dashboard.go` |
| Custom palette that sets the terminal background | `Solarized` in `dashboard.go` |
| **Options**: badge, `Order`, `Hotkey`, `OnOff`/`Badge`/`Count` helpers | `opt_*.go`; `[60]`-`[62]` for `Order` |
| **Tasks**: `RunTask`, pending message, `OnResult` hook | `[1]` backup, `[3]` export (opens a report) |
| **Hooks**: `OnTick` (per-dashboard state with `Local`), `OnMsg` (own message type), `OnResult` | `widget_*.go`, `[70]` alert |
| `Context.Exec` (hand the terminal to a program) | `[72]` |
| `Context.Err` / `Ready` (stale data, last good state kept) | `[71]` simulate an outage |
| `Context.GotoPage` | `[60]`-`[62]` |
| **Pop-ups**: `Notice` (static, live, scrolling), `Confirm`, `Typed`, `Edit` (validate, save, submit, secret, empty, limit), `Choice`, `Slider` | `[30]`-`[38]`, `[10]`-`[15]` |
| **Custom pop-ups**: `Modal`, `ResultModal`, chained flows | `[40]` dice, `[41]` ping, `[42]` three steps |
| **Views**: status and options, custom body and keys, view-local hotkeys, view opening a view | `[50]` diagnostics, `[51]` processes |
| **Scrolling view** (`Scroll`) with a scroll bar; `OnEnter` / `OnLeave` hooks | `[53]` release notes (`n`) |
| **Tail view** (`Tail`): starts at the newest row, holds its place while rows arrive, End goes live | `[54]` event stream (`e`) |
| **Live graphs that keep their own history**: `chart.Stream`, `chart.DualStream` | `widget_chart_streams.go` (Charts page) |
| **Compact page** (`Page.Fallback`): shrink the window on the Monitor page (`FallbackBelow`) and the Charts page (`FallbackWhen`, any rule) | `dashboard.go` (`monitorSummary`, `chartsSummary`) |
| **Coloured, worded hints**: `Shortcut.Display` / `Tone`, `Option.HotkeyDisplay` / `HotkeyTone` | `shortcut_*.go`, `[4]` open web UI (`Ctrl+O`) |
| **View chrome**: `OptionsTitle`, `OptionsSubtitle`, `Header`, `Hints` | `[50]` diagnostics |
| **View in a panel** (`View.Panel`) with the scroll position in its border | `[54]` event stream |
| **`Config.StartView`**: open at a view stack (`-view diagnostics,stdaddons.logs`) | `main.go` |
| **`Edit.OnSaved`**: chain a follow-up after a save | `[10]` storage limit (reloads the service) |
| **`Choice`**: `Badge`, `Intro`, `Rows`, `FooterSelect`, `FooterCancel` | `[13]` region |
| **`Dashboard.Notify`**: events pushed from a goroutine outside the UI (see "Pushed" in Diagnostics) | `dashboard.go` (`feed`) |
| **Shortcuts**: `ScopeEverywhere` (ctrl+q, ctrl+r, ctrl+l) and `ScopePages` (`?`) | `shortcut_*.go` |
| **Logs**: standard `log` and `slog` captured, newest first, scroll, filter, escape-safe | `[52]` logs (`g`), `[73]` writes entries |
| **Log categories**: tabs along the top, switched with `c` (`C` goes back); one logger per category | `[52]`, `[73]`; `NetLog`, `DiskLog`... in `dashboard.go` |
| **Charts**: `Meter`, `Sparkline`, `HeatStrip`, `BarGraph`, `LineTrace`, `DualWave`, `Braille`, `WaveScaler`, `Series`, `Resample` | Monitor and Charts pages |
| Panel dividers, `panel.Row`, `text.Rate` | `widget_system.go` |
| **Theme and display**: palettes, spacing sliders, refresh rate, TTY symbols, live preview | `[20]`-`[23]`; the `Theme` page shows the active glyphs and palette |
| **Background**: terminal / paint / off modes, and a one-key transparent switch (`t`) | `[24]`, `[25]`; or `-background` |
| Glyph modes (`auto`, `unicode`, `tty`) | `[23]`; or `-glyphs` |
| `Store`: JSON file or in memory (`-ephemeral`), restored on the next run | every setting under `Settings` |
| `ExitInfo`: reasons `stop` and `reset` handled after the terminal is restored | `main.go` |

## Adding to the showcase

Add one file to [`showcase/dashboard/addons/`](showcase/dashboard/addons/) and run
`make run`. Nothing else is edited. See [`docs/ADDONS.md`](../docs/ADDONS.md).
