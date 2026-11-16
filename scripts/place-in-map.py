#!/usr/bin/env python3
"""Place a new feature in an application's map, for the template and consumer tests.

    place-in-map.py <addons.go> <kind> <id>

kind is a feature kind (opt, widget, view, page, modal, shortcut, hook) or map, and id
the feature's Go name prefix, as `dashygotui new` writes it (storageLimit for
storage_limit). A shortcut joins Map.Shortcuts, a hook Map.Hooks; anything else
goes on a new last page, so the rows already there keep their numbers.
"""
import sys


def closing(src, open_at):
    """Index of the brace that closes the one at open_at."""
    depth = 0
    for i in range(open_at, len(src)):
        if src[i] == "{":
            depth += 1
        elif src[i] == "}":
            depth -= 1
            if depth == 0:
                return i
    raise SystemExit("place-in-map: unbalanced braces")


def main():
    path, kind, ident = sys.argv[1], sys.argv[2], sys.argv[3]
    src = open(path).read()
    if kind == "shortcut":
        key = "Shortcuts: []dashboard.Shortcut{"
        if key in src:
            src = src.replace(key, key + ident + "Shortcut, ", 1)
        else:
            src = src.replace("return dashboard.Map{", "return dashboard.Map{\n\t\t" + key + ident + "Shortcut},", 1)
    elif kind == "map":
        start = src.index("Pages: []dashboard.Page{") + len("Pages: []dashboard.Page")
        end = closing(src, start)
        src = src[:end] + ident + "Page(),\n" + src[end:]
    elif kind == "hook":
        key = "Hooks: []dashboard.Hook{"
        if key in src:
            src = src.replace(key, key + ident + "Hook, ", 1)
        else:
            src = src.replace("return dashboard.Map{", "return dashboard.Map{\n\t\t" + key + ident + "Hook},", 1)
    else:
        panel = {
            "widget": "{Widget: %sWidget}" % ident,
            "opt": '{Title: "New", Items: []dashboard.Addon{%sOpt}}' % ident,
            "view": '{Title: "New", Items: []dashboard.Addon{%sView}}' % ident,
            "modal": '{Title: "New", Items: []dashboard.Addon{%sModalOpt}}' % ident,
            "page": '{Title: "New", Items: []dashboard.Addon{dashboard.Page{Title: "Sub", Badge: %sBadge, '
                    'Session: %sSession, Panels: []dashboard.Panel{{Title: "In", Items: []dashboard.Addon{'
                    'dashboard.Option{Label: "Hello", Do: func(*dashboard.Ctx) {}}}}}}}}' % (ident, ident),
        }.get(kind)
        if panel is None:
            raise SystemExit("place-in-map: unknown kind " + kind)
        start = src.index("Pages: []dashboard.Page{") + len("Pages: []dashboard.Page")
        end = closing(src, start)
        page = '{Title: "New %s", Panels: []dashboard.Panel{%s}},\n' % (kind, panel)
        src = src[:end] + page + src[end:]
    open(path, "w").write(src)


main()
