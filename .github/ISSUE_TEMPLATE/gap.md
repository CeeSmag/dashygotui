---
name: Library gap
about: An application needed something dashygotui does not provide (or found a workaround for it)
title: "gap: "
labels: gap
---

<!-- Read docs/DESIGN.md (the generic gate and the gap protocol) first. -->

## What the application needed

What it was trying to show or do, in the user's terms. Leave out the application's
domain: say "a setting that restarts a service", not what the service is.

## What it did instead

The workaround, and where it lives (file and line). Is it marked
`// dashygotui:gap <this issue>`?

## The shortest API that would have avoided it

A struct field or a function, as the addon would have used it:

```go

```

## Would a second, unrelated application use it?

Name one, and how.

## Checked

- [ ] `docs/COOKBOOK.md` and `docs/REFERENCE.md` do not already cover this
- [ ] It passes the generic gate in `docs/DESIGN.md`
