// Package panel draws the framed boxes a dashboard is made of, and the small
// layout helpers that stack, join and overlay blocks of text rows.
//
// A panel is a rounded box whose top border carries an inverted tab with the
// title and an optional badge, and whose bottom border can carry a subtitle:
//
//	╭─┐ System ┌───┐ ● ONLINE ┌───────────────────────────╮
//	│ CPU     [■■■■■■■■■■■■■■■■]   12.5%                  │
//	├── NETWORK ────────────────────────────────────────────┤
//	│ Net     ▲ 340.0 KB/s   ▼ 1.2 MB/s                    │
//	╰─┘ app.service └─────────────────────────────────────╯
//
// Corners and other symbols come from the theme's glyph set, so the same panel
// is drawn with straight corners in TTY mode.
//
// Every panel is exactly w cells wide and h rows tall, whatever it is given: a
// long title is truncated, then dropped in favour of the badge, and a box too
// narrow for a tab falls back to a plain rule. Callers pass body rows already
// styled; the panel only frames, pads and clips them.
package panel

import (
	"strings"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// dividerMark prefixes a body row that renders as a labelled rule rather than
// as text. [Divider] builds one.
const dividerMark = "\x00div:"

// Divider marks a body row as a labelled rule across the panel:
// ├── LABEL ──┤.
func Divider(label string) string { return dividerMark + label }

// Panel frames body into a w x h panel. Rows beyond h-2 are dropped and
// missing rows are left blank, so the result is always exactly h lines of
// exactly w cells. It returns nil when w or h is not positive.
func Panel(th *theme.Theme, title, badge, subtitle string, w, h int, body []string) []string {
	if w <= 0 || h <= 0 {
		return nil
	}
	out := make([]string, 0, h)
	out = append(out, top(th, title, badge, w))
	if h == 1 {
		return out
	}
	for i := 0; i < h-2; i++ {
		line := ""
		if i < len(body) {
			line = body[i]
		}
		if label, ok := strings.CutPrefix(line, dividerMark); ok {
			out = append(out, rule(th, label, w))
			continue
		}
		out = append(out, row(th, line, w))
	}
	return append(out, bottom(th, subtitle, w))
}

// row frames one body row: │ content │, with the content padded or truncated
// to the w-4 cells between the borders.
func row(th *theme.Theme, content string, w int) string {
	switch {
	case w <= 0:
		return ""
	case w == 1:
		return th.Border.Render("│")
	case w == 2:
		return th.Border.Render("││")
	case w == 3:
		return th.Border.Render("│") + " " + th.Border.Render("│")
	}
	return th.Border.Render("│") + " " + th.Fit(content, w-4) + " " + th.Border.Render("│")
}

// top draws the top border, degrading in four steps: title and badge,
// truncated title and badge, badge or title alone, then a plain rule.
func top(th *theme.Theme, title, badge string, w int) string {
	if w < 4 {
		return th.Border.Render(corners(th.G.CornerTL, th.G.CornerTR, w))
	}
	tw, bw := text.Width(title), text.Width(badge)
	if title != "" && badge != "" {
		if fill := w - 14 - tw - bw; fill >= 1 {
			return tabTop(th, title, badge, fill)
		}
		// Keep the badge and shorten the title, but only while the title is
		// still worth reading.
		if avail := w - 15 - bw; avail >= 4 {
			t := th.Trunc(title, avail)
			return tabTop(th, t, badge, w-14-text.Width(t)-bw)
		}
	}
	if badge != "" {
		if fill := w - 7 - bw; fill >= 1 {
			return soloTop(th, badge, fill, false)
		}
	}
	if title != "" {
		if fill := w - 7 - tw; fill >= 1 {
			return soloTop(th, title, fill, true)
		}
		if avail := w - 8; avail >= 1 {
			t := th.Trunc(title, avail)
			return soloTop(th, t, w-7-text.Width(t), true)
		}
	}
	return th.Border.Render(th.G.CornerTL + strings.Repeat("─", w-2) + th.G.CornerTR)
}

// tabTop is the full tab: ╭─┐ Title ┌───┐ Badge ┌──╮.
func tabTop(th *theme.Theme, title, badge string, fill int) string {
	return th.Border.Render(th.G.CornerTL+"─┐ ") + th.Title.Render(title) +
		th.Border.Render(" ┌───┐ ") + badge +
		th.Border.Render(" ┌"+strings.Repeat("─", fill)+th.G.CornerTR)
}

// soloTop is the tab with one label: ╭─┐ Label ┌──╮.
func soloTop(th *theme.Theme, label string, fill int, isTitle bool) string {
	body := label
	if isTitle {
		body = th.Title.Render(label)
	}
	return th.Border.Render(th.G.CornerTL+"─┐ ") + body +
		th.Border.Render(" ┌"+strings.Repeat("─", fill)+th.G.CornerTR)
}

// bottom draws the bottom border, with the subtitle in its own tab when it
// fits: ╰─┘ subtitle └──╯.
func bottom(th *theme.Theme, subtitle string, w int) string {
	if w < 4 {
		return th.Border.Render(corners(th.G.CornerBL, th.G.CornerBR, w))
	}
	if subtitle != "" {
		sw := text.Width(subtitle)
		if fill := w - 9 - sw; fill >= 1 {
			return th.Border.Render(th.G.CornerBL+"─┘ ") + th.Muted.Render(subtitle) +
				th.Border.Render(" └──"+strings.Repeat("─", fill)+th.G.CornerBR)
		}
		if avail := w - 10; avail >= 6 {
			s := th.Trunc(subtitle, avail)
			return th.Border.Render(th.G.CornerBL+"─┘ ") + th.Muted.Render(s) +
				th.Border.Render(" └──"+strings.Repeat("─", w-9-text.Width(s))+th.G.CornerBR)
		}
	}
	return th.Border.Render(th.G.CornerBL + strings.Repeat("─", w-2) + th.G.CornerBR)
}

// rule draws a labelled rule across the panel: ├── LABEL ──┤.
func rule(th *theme.Theme, label string, w int) string {
	if w < 8 || label == "" {
		return th.Border.Render(corners("├", "┤", w))
	}
	label = th.Trunc(label, w-8)
	fill := w - 6 - text.Width(label)
	return th.Border.Render("├── ") + th.Key.Render(label) +
		th.Border.Render(" "+strings.Repeat("─", fill)+"┤")
}

// corners is the degenerate border for a box too narrow for anything else.
func corners(left, right string, w int) string {
	switch {
	case w <= 0:
		return ""
	case w == 1:
		return left
	default:
		return left + strings.Repeat("─", w-2) + right
	}
}
