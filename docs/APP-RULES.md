# Rules for applications built on dashygotui

These rules apply to an application's `addons/` directories. They are checked by
`dashygotui check` (the rule IDs below) and, for rendering, by `dashtest.Audit`.
Run both in the application's `make check`. Read `DESIGN.md` for the reasons.

An exception is possible, never silent: write `//dashygotui:allow A5 <reason>` on
the line (or alone on the line above). The check prints every exception so a
reviewer sees it. An allow comment with no reason is itself an error (X1).

## The shape of an application

- **One feature, one file.** Adding an option, widget, view, shortcut or pop-up
  means adding one file in `addons/` and editing no other file. Deleting an addon
  file leaves the application building and running.
- The `dashboard` package declares the pages, the sections and the one `Registry`.
  `main` imports `addons` with a blank import and nothing else.
- Addons reach the library through the `*Context` they are given and the registry.
  They register from `init()`, with `Registry.Add`.

## The rules

| ID | Rule | Instead |
| --- | --- | --- |
| A1 | Addon files are named `opt_*.go`, `widget_*.go`, `view_*.go`, `modal_*.go` or `shortcut_*.go`. | Rename the file. |
| A2 | An addon never uses a declaration from a sibling addon file. | Move shared code to the `dashboard` package. |
| A3 | No package-level mutable variable in an addon. | `dashygotui.Local` for state of the dashboard, the `Store` for saved settings. |
| A4 | No `go` statement, `time.Sleep`, `os.Exit`, `fmt.Print*`, `log.Fatal*`, `panic`, or use of `os.Stdout`, `os.Stderr` or `os.Stdin`. | `Context.RunTask`, `Context.Quit`, `Context.Flash`, `Context.FlashError`. |
| A5 | No `lipgloss.NewStyle`, `lipgloss.Color`, `lipgloss.Style{}` literal (also when dot-imported), and in an addon no colour literal such as `"#38BDF8"`. | `Context.Theme()` styles, or a `Tone`. |
| A6 | Text an addon writes is plain ASCII (`...`, not an ellipsis; no tick or cross characters) with no control or escape characters. | `Context.Theme().G` symbols; styling is the theme's job. |
| A7 | No `tea.Quit`, `tea.Println`, `tea.Printf`, `tea.ExecProcess` or `tea.Exec`. | `Context.Quit`, `Context.Exec`. |
| A8 | No struct embedding a library modal, and no type defined from one. | Use the modal's fields; if it lacks one, record a gap. |
| A9 | No `Run` that only opens a view or only returns nil. | `Opens: "<view>"`, `Do: func(c) {...}`, or an `Entry` on the view. |
| A10 | No variable declared in `init` and changed by a callback (`n++`, `list = append(...)`, `m[k] = v`): it is shared by every dashboard in the process. | `dashygotui.Local` (see "Where an addon keeps its state"). |

**Where they apply.** A1 to A4, A6 and A9 apply to files in a directory named
`addons`. A5 (raw styles), A7 and A8 apply to **every file that imports the
library**, because a workaround is as likely to sit in the `dashboard` package as in
an addon (a wrapper type around a library modal there is still a wrapper).

**What the check is.** It reads the syntax; it is not a proof. It catches the usual
ways around a rule (aliased and dot imports, literals, `os.Stdout`) but a determined
author can still hide one. It exists to stop the easy mistakes and to make the
deliberate ones visible: that is what the allow comment and the gap marker are for.

**The change set (C1).** `dashygotui check -changed` also looks at what a change touches
(`git diff` against `-base`, default `HEAD`, plus untracked files). A change that **adds** an
addon file and also **edits, deletes or renames another Go file** (the dashboard file, a
sibling addon, a registry table) is reported as C1: adding an addon should edit nothing
else. Tests, `go.mod`, `go.sum` and non-Go files are exempt. It is advice, because an edit
can be unavoidable (a new section in the dashboard file); say why in the change
description, or run with `-strict` to make it fail.

**For AI agents.** `dashygotui init -claude` writes a Claude Code skill
(`.claude/skills/dashygotui/SKILL.md`) that tells the agent when to read these rules and
what loop to follow, and merges a `PostToolUse` hook into `.claude/settings.json` that
runs `dashygotui check -q` after every edit and hands the findings back to the agent.
Your other settings and hooks are kept; a settings file that is not a JSON object is
never overwritten.

Two more checks come with the same command: **G1** (a `//dashygotui:gap` marker
must say where the gap is recorded) and **X1** (an allow comment needs a reason).

## Where an addon keeps its state

- **Saved between runs:** the `Store` (settings; `stdaddons.Toggle` and friends do it).
- **Alive while the dashboard runs** (a history it samples, a selection, a counter):
  `dashygotui.Local[T](c, "key")`, which gives the addon a value that belongs to
  *this dashboard*. Do not capture a variable in `init` for this: a captured variable
  is shared by every dashboard in the process, so two dashboards (a test, an
  embedding) corrupt each other and tests depend on their order. The check cannot see
  such a variable when it is changed through a method (`series.Add(...)`) rather than assigned, so that case is on the author; A10 catches assignments, increments and element or field writes.
- **The application's data:** the state type `S` that `Poll` returns.

## Conduct (for people and AI agents)

1. Read `DESIGN.md`, this file and `COOKBOOK.md` before writing an addon. They
   match the library version in `go.mod`: `dashygotui rules`.
2. Use what exists. Search `COOKBOOK.md` and the godoc first; do not invent APIs.
3. If the library cannot do it, **stop and record a gap** (see `DESIGN.md`). Do not
   wrap, copy or fork library code, and do not override a default to get around a
   limit.
4. Make the smallest diff. Do not create, rename or delete files that were not
   requested.
5. Run the application's `make check` (which runs `dashygotui check` and the
   audit test) before declaring a task done. Report failures with their output.
6. Commit or push only when asked.
