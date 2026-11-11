# RULES.md: dashygotui

These rules are **mandatory** for every contributor and every AI coding agent.
Read this file completely before changing anything. When a rule here conflicts
with a habit, the rule wins. When the four baseline style guides below conflict
with each other, the stricter one wins.

Machine-enforced rules are marked **[enforced]** and are checked by
`make check`. A task is not finished until `make check` passes.

---

## 1. Purpose and the one-file rule

`dashygotui` is a generic library for building Bubble Tea v2 dashboards with as
little application code as possible.

> **Zero-edit extension.** Adding an option, widget, view, shortcut or modal to
> an application means adding **exactly one new file** in that application's
> `addons/` directory. No other file is edited: not the dashboard file, not a
> registry table, not a README index. **[enforced: addon guard tests and
> `make deletion-test`]**

Corollaries:

- Deleting any single addon file must leave the module compiling and the
  dashboard running. **[enforced: `make deletion-test`]**
- A feature that needs persistence, a background task, a shortcut, a modal and
  a sub-view registers all of them from the same file.
- Editing an existing file to introduce a new addon is prohibited unless it is
  genuinely unavoidable and the reason is stated in the change description.

## 2. Style baseline (all four are required)

1. [Effective Go](https://go.dev/doc/effective_go)
2. [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
3. [Google Go Style Guide](https://google.github.io/styleguide/go/) (decisions and best practices)
4. [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

Where they differ, apply the stricter rule. The sections below make the
project-specific consequences explicit.

## 3. Layout and dependency boundaries

```
main.go  --imports-->  dashboard   (pages, section slots, the one global Registry, Run)
         --blank---->  dashboard/addons   (opt_*.go, widget_*.go, view_*.go, modal_*.go, shortcut_*.go)
addons   --imports-->  dashboard  --imports-->  dashygotui (this library)
```

- The library (`dashygotui`, `layout`, `theme`, `text`, `panel`, `chart`,
  `modal`, `stdaddons`, `store`, `dashtest`, and anything under `internal/`)
  **never** imports `examples/` or any `addons` package. **[enforced]**
- `dashboard` **never** imports `addons`. Only `main` does, with a blank import.
- Addon files **never** depend on each other: no calling a function, using a
  type or reading a variable declared in a sibling addon file. Shared code
  moves into the `dashboard` package or the library. **[enforced]**
- The core is OS-agnostic: no `syscall`, `golang.org/x/sys`, `net/http`, no
  `runtime.GOOS`/`GOARCH` branching, no `/proc` or other platform paths.
  `os/exec` is allowed in exactly one library file (`exec.go`), which wraps
  `tea.ExecProcess`. **[enforced]**
- Platform-specific behaviour (privilege escalation, service managers, `/proc`,
  log readers, ...) belongs to the application's own addons, never to the library.
- Non-public code lives in `internal/`. Everything not listed in the layout is
  unexported by default.

## 4. Package and API design

- No `util`, `common`, `helpers`, `base` or `misc` packages or files. Name a
  package for what it provides (`text`, `panel`, `chart`).
- No stuttering: `panel.Panel` is acceptable for a package's central type,
  `panel.PanelRenderer` is not; `dashygotui.DashygotuiConfig` is never.
- Interfaces are small, defined where they are **consumed**, and introduced only
  when there is a second implementation or a test seam. Accept interfaces,
  return concrete types.
- Use generics only where the application state type `S` flows. Never add a type
  parameter "for flexibility".
- Prefer plain struct literals for configuration (`Config[S]`, `Option[S]`,
  `Widget[S]`). Do not add fluent builders. Use functional options only for APIs
  with several optional, independent arguments.
- Zero values must be useful or explicitly rejected by validation.
- Every exported identifier has a doc comment that starts with its name and is a
  complete sentence. Each package has a package comment. **[enforced: lint]**
- The library follows semantic versioning. Do not change an exported signature
  without calling it out as a breaking change.

## 5. Globals, `init()` and registration

- The library has **no mutable package-level state** and **no `init()`**.
  Package-level `var` is limited to sentinel errors (`errors.New` /
  `fmt.Errorf`), compiled `regexp`s, and `var _ Iface = (*T)(nil)` assertions.
  **[enforced]**
- Theme, spacing, refresh interval and every other piece of runtime state live
  on the `Dashboard` value and are passed explicitly to renderers.
- The **only** global in an application is its single `Registry[S]`, declared in
  the application's dashboard file. The library merely provides the generic
  type.
- `init()` is allowed only in `addons/` and `examples/`, and only to register
  with a registry. No I/O, no goroutines, no reading environment or flags in
  `init()`.
- Registration never panics and never returns an error to `init()` (it cannot).
  The registry records the problem together with the caller's `file:line`, and
  `Registry.Validate` / `dashygotui.New` fail loudly before the UI starts.
- `init()` order follows filenames and must not be relied on. Ordering of
  options, widgets and panels comes only from the explicit `Order` field, then
  the stable `Key`.
- Option numbers are assigned by hand and are unique across the dashboard. A
  duplicate number, duplicate key, or reference to an unknown page or section
  is a startup error that names both offending call sites.

## 6. Errors

- Errors are values. Wrap with `fmt.Errorf("doing x: %w", err)`; inspect with
  `errors.Is` / `errors.As`; combine with `errors.Join`.
- Error strings are lowercase, have no trailing punctuation and no
  "failed to" / "error:" prefixes.
- The library does not panic. The only exception is a `Must*` constructor,
  whose name says so, used for programmer errors in program setup.
- Never ignore an error silently. A deliberate discard is written `_ = f()` with
  a comment saying why.
- The library never writes application output to stdout or stderr and never
  calls `log`, `os.Exit` or `fmt.Print*`. Results flow out through return values,
  `Context.Flash`, and `ExitInfo`. The one exception is terminal control
  sequences (the opt-in `Config.AltScroll` mode, DEC 1007), which configure the
  terminal rather than print anything, are restored on exit, and are written
  only from `run.go`.

## 7. Concurrency

- `context.Context` is the first parameter of anything that blocks or does I/O,
  and is not stored in a struct. The single exception is the program context
  held by the Bubble Tea model (`model.run`), because commands created inside
  `Update` must be cancelled when the program ends; it is documented at the field
  and is never exposed to addons.
- Every goroutine has a clear owner and a defined stop condition; never start
  one in `init()`.
- **All model state is mutated only inside `Update`.** Background work (polling,
  tasks) returns results as `tea.Msg`; it never touches the model directly.
- Protect any shared state with a mutex that has a short, obvious scope.
  `go test -race ./...` must pass. **[enforced]**

## 8. Rendering safety

- Every page, section, widget, modal and view must render **without panic at
  any terminal size from 0x0 upward**, and must return exactly `cols x rows`
  cells (clipped or padded). **[enforced: resize tests]**
- Below the minimum usable size the dashboard shows a "terminal too small"
  notice instead of a broken layout.
- Layout code clamps spacing instead of overflowing. Width is always measured
  ANSI-aware (`text.Width`), never with `len`.
- The mouse is not captured; the terminal keeps selection and copy.
- **Symbols come from the theme.** A renderer draws with `theme.Glyphs` (`th.G`)
  and never hard-codes a symbol that the Linux console font may lack: no `●`,
  `✓`, `✗`, `▲`, `▼`, `…`, `↑↓`, rounded corners or Braille outside the Unicode
  glyph set. TTY mode must draw only ASCII, the straight line-drawing characters
  and the shade blocks. Text written by addons is plain ASCII (`...`, not `…`).
  The demo's TTY sweep test enforces this for every page, view and pop-up.

## 9. Testing

- Table-driven tests with named subtests. Mark helpers with `t.Helper()`. Use
  `t.Parallel()` wherever the test touches no shared state.
- Tests never touch the real user config directory, the real network, or the
  real system. Use `store.Memory`, `t.TempDir()`, and fake poll functions.
- Visual output is verified with golden files under `testdata/`, refreshed with
  `go test -run TestName -update`. Review every golden diff.
- A bug fix lands with a test that fails without the fix.
- Application-agnostic behaviour (layout, text, charts) is tested without
  Bubble Tea. Behaviour that needs the model is driven through `dashtest`, whose
  fake clock and timers mean a test never sleeps.
- Assertions about menu rows must not depend on exact spacing (the number column
  widens when another addon adds a longer number): match `\[10\]\s+Label`, not
  `"[10] Label"`. A test that breaks when a file is added breaks rule 1.

## 9a. Architecture guard tests

`arch_test.go` parses the source with `go/parser` and enforces sections 3, 5
and 10. Do not weaken, skip, or work around a guard test. If a guard is wrong,
change the guard in the same change and explain why.

## 10. Addon rules (`addons/` directories)

- File names are `opt_<name>.go`, `widget_<name>.go`, `view_<name>.go`,
  `modal_<name>.go` or `shortcut_<name>.go` (plus `_test.go`). **[enforced]**
- An addon reaches the library only through the `*Context[S]` it is handed and
  the registry in the dashboard package. No package-level mutable state, and no
  variable captured in `init` and changed by a callback (it is shared by every
  dashboard in the process): an addon's own state lives in `dashygotui.Local`, a
  modal's or view's in its struct. **[enforced on applications: dashygotui check]**
- Pages and section containers are declared in the dashboard file, not in
  addons. An addon targets them by string ID.
- Persisted addon settings use `Context.Store()` with a key namespaced by the
  addon's own `Key`.
- Copy a file from `templates/` to start a new addon.

## 11. Tooling gate

`make check` runs: `gofmt`/`goimports` check, `go vet`, `staticcheck`,
`golangci-lint`, `govulncheck`, `go test -race -shuffle=on ./...` (which includes the
guard, API-snapshot, reference and documentation tests), `make deletion-test` (every
addon removable), `make template-test` (every file in `templates/` compiles and
registers cleanly), `make consumer-test` (a throw-away application scaffolds, builds,
passes `dashygotui check` and fails it for each bad addon), `make app-check` (the
examples obey the application rules), `make api-check` (an API change comes with a
changelog entry) and `go mod tidy -diff`. All must pass. Never skip, silence or loosen a
check to make a change pass; fix the cause. A `//nolint` needs a specific linter
name and a reason on the same line.

New dependencies need explicit approval. The library depends on the Charm v2
stack only (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`,
`charm.land/bubbles/v2`, `github.com/charmbracelet/x/ansi`) plus the standard
library.

## 12. Rules for AI coding agents

1. Read this file first, then `docs/DESIGN.md` and `docs/ARCHITECTURE.md`;
   `docs/ADDONS.md` and `docs/COOKBOOK.md` before writing an addon.
2. Do only what was asked. Do not create, rename or delete files that were not
   requested. Keep diffs minimal and focused.
3. To add an addon, add one file. If you believe another file must change, stop
   and say why instead of editing it.
4. Do not add dependencies, build tags, code generation, global state or
   `init()` outside `addons/` and `examples/`.
5. Do not invent APIs that are not in the library; read the code and the godoc.
6. Run `make check` before declaring a task done. Report failures with their
   output; never claim success for a check you did not run.
7. Commit or push only when asked. Commits are authored as the repository owner,
   with a plain message and **no** AI attribution trailer (no `Co-Authored-By`,
   no "Generated with").
8. Ask before any destructive, irreversible or outward-facing action.
9. If the library cannot do what is needed, **record a gap and stop** (see
   `docs/DESIGN.md`); never work around it in an application or in a copy.
10. A new library feature passes the generic gate in `docs/DESIGN.md`: two
    unrelated applications could use it, and it carries no application's
    vocabulary. A change to the exported API updates `api/dashygotui.txt` and
    `CHANGELOG.md` in the same change. **[enforced]**

## 13. The design charter and the application rules

- `docs/DESIGN.md` is the charter: complexity in the library, simplicity in the
  application; the generic gate; where code lives; the gap protocol.
- `docs/APP-RULES.md` holds the rules for applications built on the library
  (rules A1 to A10). They are enforced on an application by `dashygotui check`
  (`cmd/dashygotui`) and `dashtest.Audit`, and the library's examples are held to
  them by the same code (`internal/guard`).
- `docs/COOKBOOK.md` is the "I want X, so use Y" table. When an addon is longer
  than the cookbook's form, the cookbook, the library or the addon is wrong.
- The rules in `internal/guard` are shared: a rule changed there changes the
  guard tests and the command together.
