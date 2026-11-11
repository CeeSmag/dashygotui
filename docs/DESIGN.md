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
| Pages, sections, the one `Registry` | the application's dashboard file |
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
  to A10), `check -changed` applies the one-file rule to a change, and `dashtest.Audit`
  checks the dashboard at every size. All belong in the application's own `make check`.
- **They reach the agent at the moment it matters.** `dashygotui init -claude` installs a
  skill and a hook, so findings arrive after the edit that caused them, not at the end.
- **The library governs itself.** The exported API is snapshotted, the docs must
  mention every exported field, and the library is scanned for application
  vocabulary.
- **The easy way is the right way.** The short forms (`Opens`, `Do`, `Entry`,
  `stdaddons.Toggle`...) exist so that following the design is also the least
  typing.
