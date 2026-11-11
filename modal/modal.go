// Package modal provides ready-made pop-ups for dashygotui dashboards: a text
// editor, a yes/no confirmation, a type-the-word confirmation, a scrollable
// notice, a choice list and a slider.
//
// Each type is a plain struct literal that implements dashygotui.Modal; open
// one from an option or shortcut with Context.OpenModal:
//
//	return ctx.OpenModal(&modal.Confirm[State]{
//		Title:    "Delete everything?",
//		Lines:    []string{"This cannot be undone."},
//		Accept:   "Delete",
//		OnAccept: func(ctx *dashygotui.Context[State]) tea.Cmd { ... },
//	})
//
// All state lives in the modal value, so every modal is safe to create anew
// each time it is opened. They draw with the dashboard's active theme and are
// clipped to any terminal size.
package modal

import (
	"strings"

	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// width picks a pop-up width of up to want cells that fits inside cols with a
// margin. A terminal too narrow for a sensible box gets the full width.
func width(cols, want int) int {
	w := min(want, cols-4)
	if w < 30 {
		return cols
	}
	return w
}

// popup frames body into a panel just tall enough to hold it, clipped to rows.
func popup(th *theme.Theme, title, badge string, w, rows int, body []string) []string {
	box := panel.Panel(th, title, badge, "", w, len(body)+2, body)
	return panel.ClipHeight(box, rows)
}

// wrapAll wraps every line to inner cells, keeping blank lines.
func wrapAll(lines []string, inner int) []string {
	var out []string
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		out = append(out, text.Wrap(line, inner)...)
	}
	return out
}

// keys renders a footer line such as "Enter Save    Esc Cancel".
func keys(th *theme.Theme, pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, th.Muted.Render(pairs[i])+" "+pairs[i+1])
	}
	return strings.Join(parts, "    ")
}
