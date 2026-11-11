package dashygotui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/CeeSmag/dashygotui/layout"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// menuColumnGap is the space between the two columns of a wide menu.
const menuColumnGap = 3

// dashScreen draws the active page with header, tab bar, body and footer,
// inset on every side by the terminal-to-page gap.
func (m *model[S]) dashScreen(cols, rows int) []string {
	return layout.Inset(cols, rows, m.spacing.TerminalPage, layout.MinCols, layout.MinRows,
		func(w, h int) []string {
			body := max(h-3, 1) // header, tab bar and footer take one row each
			out := []string{m.header(w), m.tabBar(w)}
			out = append(out, m.renderPage(m.d.pages[m.page], w, body)...)
			return append(out, m.footer(w, "LIVE "+formatInterval(m.interval), m.autoHints()))
		})
}

// header is the strip along the top of every screen.
func (m *model[S]) header(cols int) string {
	var segs []panel.Segment
	if m.d.cfg.Header != nil {
		segs = m.d.cfg.Header(m.ctx)
	} else {
		segs = []panel.Segment{
			{Text: m.th.White.Render(m.title()), Prio: 100},
			{Text: m.th.Green.Render(m.clock()), Prio: 80},
		}
	}
	return panel.Header(m.th, segs, cols)
}

func (m *model[S]) clock() string { return m.now().Format("15:04:05") }

// tabBar renders the page navigation across the top of the dashboard.
func (m *model[S]) tabBar(cols int) string {
	th := m.th
	parts := make([]string, 0, len(m.d.pages))
	for i, p := range m.d.pages {
		num := strconv.Itoa(i + 1)
		label := m.tabLabel(p.page, cols, i == m.page)
		if i == m.page {
			parts = append(parts, th.Border.Render("[")+th.White.Render(num)+" "+th.Green.Render("●")+" "+th.Title.Render(label)+th.Border.Render("]"))
		} else {
			parts = append(parts, th.Muted.Render(num+" "+label))
		}
	}
	tabs := " " + strings.Join(parts, th.Border.Render(" │ "))
	hint := th.Key.Render("[Tab]") + " " + th.Muted.Render("Next Page") + " "
	if text.Width(tabs)+text.Width(hint)+2 <= cols {
		return text.Pad(tabs, cols-text.Width(hint)) + hint
	}
	hint = th.Key.Render("[Tab]") + " "
	if text.Width(tabs)+text.Width(hint)+1 <= cols {
		return text.Pad(tabs, cols-text.Width(hint)) + hint
	}
	return text.Fit(tabs, cols)
}

// tabLabel is the tab's text at this width: shorter titles on narrower
// terminals, and the badge on the active tab.
func (m *model[S]) tabLabel(p Page[S], cols int, active bool) string {
	if active && p.Badge != nil {
		if b := p.Badge(m.ctx); b != "" {
			base := p.ShortTitle
			if base == "" {
				base = p.Title
			}
			return base + " " + b
		}
	}
	switch {
	case cols < 75 && p.TinyTitle != "":
		return p.TinyTitle
	case cols < 100 && p.ShortTitle != "":
		return p.ShortTitle
	}
	return p.Title
}

// footer is the bottom row: a live spinner, the key hints (or the latest
// message, which takes their place), and the number being typed.
func (m *model[S]) footer(cols int, mode string, hints []panel.Hint) string {
	th := m.th
	left := " " + th.Green.Render(th.Spinner(m.ticks)) + " " + th.White.Render(mode)
	right := ""
	if m.digits != "" {
		right = th.Key.Render("#"+m.digits) + " "
	}
	budget := cols - text.Width(left) - text.Width(right) - 3
	body := panel.FitHints(th, hints, budget)
	if m.flash != "" {
		style := th.Mint
		if m.flashBad {
			style = th.Amber
		}
		body = style.Render(text.Trunc(m.flash, max(budget, 0)))
	}
	row := left + th.Border.Render(" │ ") + body
	return text.Fit(text.Pad(row, cols-text.Width(right))+right, cols)
}

func (m *model[S]) hint(key, label string, st lipgloss.Style, prio int) panel.Hint {
	return panel.Hint{Key: key, Label: label, Style: st, Prio: prio}
}

// autoHints builds the footer hint strip from what the current page offers.
func (m *model[S]) autoHints() []panel.Hint {
	th := m.th
	menus := m.menus()
	var hints []panel.Hint
	if len(menus) >= 1 {
		hints = append(hints, m.hint("↑↓", "Move", th.Key, 60), m.hint("↵", "Open", th.Key, 55))
	}
	if len(menus) >= 2 {
		hints = append(hints, m.hint("←→", "Panel", th.Key, 50))
	}
	if n := len(m.d.pages); n > 1 {
		hints = append(hints, m.hint("Tab", fmt.Sprintf("Page %d/%d", (m.page+1)%n+1, n), th.Key, 70))
	}
	for _, s := range m.d.reg.shortcuts {
		if s.Label != "" {
			prio := s.Prio
			if prio == 0 {
				prio = 30
			}
			hints = append(hints, m.hint(s.Key, s.Label, th.Key, prio))
		}
	}
	return append(hints,
		m.hint("+/-", "Rate", th.Key, 35),
		m.hint("q", "Quit", th.Red, 1000),
	)
}

// renderPage draws a page's panels into body rows, inset by the page-to-section
// gap and separated by the section gap.
func (m *model[S]) renderPage(p pageDef[S], cols, body int) []string {
	if len(p.panels) == 0 {
		return panel.Blank(cols, body)
	}
	sp := m.spacing
	return layout.Inset(cols, body, sp.PageSection, layout.MinCols, 5, func(w, h int) []string {
		return layout.Arrange(m.layoutPanels(p), p.page.Layout, sp.Section, w, h)
	})
}

// layoutPanels adapts a page's panels for the layout engine.
func (m *model[S]) layoutPanels(p pageDef[S]) []layout.Panel {
	sp := m.spacing
	out := make([]layout.Panel, len(p.panels))
	inter := 0
	for i := range p.panels {
		def := p.panels[i]
		if def.widget != nil {
			out[i] = layout.Panel{
				Want: func(int) int { return widgetRows(*def.widget) },
				Draw: func(w, h int) []string { return m.widgetPanel(*def.widget, w, h) },
			}
			continue
		}
		idx := -1
		if len(def.opts) > 0 {
			idx = inter
			inter++
		}
		out[i] = layout.Panel{
			Want: func(int) int { return len(def.opts) + 2 + 2*sp.SectionOption },
			Draw: func(w, h int) []string { return m.sectionPanel(def, idx, w, h) },
		}
	}
	return out
}

// widgetRows is the height a widget asks for, borders included.
func widgetRows[S any](w Widget[S]) int {
	if w.MinHeight > 0 {
		return w.MinHeight
	}
	return layoutMinHeight
}

// widgetPanel draws a widget into a w x h panel.
func (m *model[S]) widgetPanel(wd Widget[S], w, h int) []string {
	badge := ""
	if wd.Badge != nil {
		badge = wd.Badge(m.ctx)
	}
	inner := Size{Cols: max(w-4, 0), Rows: max(h-2, 0)}
	return panel.Panel(m.th, wd.Title, badge, wd.Subtitle, w, h, wd.Render(m.ctx, inner))
}

// sectionPanel draws a section with its options into a w x h panel. inter is
// the section's index among the page's menus, or -1 when it has no options.
func (m *model[S]) sectionPanel(def panelDef[S], inter, w, h int) []string {
	spec := def.sec
	var badge string
	switch {
	case spec.Badge != nil:
		badge = spec.Badge(m.ctx)
	default:
		noun := spec.Noun
		if noun == "" {
			noun = "option"
		}
		n := len(def.opts)
		if n != 1 {
			noun += "s"
		}
		badge = m.th.Muted.Render(fmt.Sprintf("[%d %s]", n, noun))
	}
	body := m.menuBody(def.opts, inter, w-4, h-2, true)
	return panel.Panel(m.th, spec.Title, badge, spec.Subtitle, w, h, body)
}

// menuBody lays options out as body rows: one per row where they fit, two
// columns where that is the only way to show them all (unless oneCol), and a
// window around the cursor when even that is not enough. It insets the rows by
// the section-to-option gap on every side.
func (m *model[S]) menuBody(opts []Option[S], inter, w, h int, oneCol bool) []string {
	if len(opts) == 0 || w <= 0 || h <= 0 {
		return nil
	}
	gap := m.spacing.SectionOption
	return layout.Inset(w, h, gap, 16, 1, func(iw, ih int) []string {
		numW, labelW := numTagWidth(opts), labelWidth(opts)
		selected := func(i int) bool { return inter == m.sect && i == m.cursorAt() }

		if !oneCol && iw >= layout.MenuTwoColCols-4 && len(opts) > ih && (len(opts)+1)/2 <= ih {
			half := (len(opts) + 1) / 2
			colW := (iw - menuColumnGap) / 2
			rightW := iw - menuColumnGap - colW
			lines := make([]string, 0, half)
			for r := range half {
				left := m.menuRow(opts[r], colW, numW, labelW, selected(r))
				right := strings.Repeat(" ", max(rightW, 0))
				if j := r + half; j < len(opts) {
					right = m.menuRow(opts[j], rightW, numW, labelW, selected(j))
				}
				lines = append(lines, left+m.th.Border.Render(" │ ")+right)
			}
			return lines
		}

		start := 0
		shown := opts
		if len(opts) > ih {
			if inter == m.sect {
				start = min(max(m.cursorAt()-ih/2, 0), len(opts)-ih)
			}
			shown = opts[start : start+ih]
		}
		lines := make([]string, 0, len(shown))
		for i, o := range shown {
			lines = append(lines, m.menuRow(o, iw, numW, labelW, selected(start+i)))
		}
		return lines
	})
}

// numTagWidth is the width of the widest "[n]" among the options, so every
// label starts in the same column.
func numTagWidth[S any](opts []Option[S]) int {
	w := 3
	for _, o := range opts {
		w = max(w, len(strconv.Itoa(o.Number))+2)
	}
	return w
}

// labelWidth is the width of the longest label, which is the column every
// badge starts at.
func labelWidth[S any](opts []Option[S]) int {
	w := 0
	for _, o := range opts {
		w = max(w, text.Width(o.Label))
	}
	return w
}

// menuRow draws one option w cells wide: the cursor, the number, the label, and
// the option's badge one space after the longest label.
func (m *model[S]) menuRow(o Option[S], w, numW, labelW int, selected bool) string {
	if w <= 0 {
		return ""
	}
	th := m.th
	badge := ""
	if o.Badge != nil {
		badge = o.Badge(m.ctx)
	}
	mark := "  "
	if selected {
		mark = th.Key.Render(th.Pointer()) + " "
	}
	head := mark + th.Key.Render(text.Pad(fmt.Sprintf("[%d]", o.Number), numW)) + " "
	avail := w - text.Width(head)
	if avail <= 0 {
		return text.Fit(head, w)
	}
	label := func(width int) string {
		s := text.Pad(text.Trunc(o.Label, width), width)
		if selected {
			return th.White.Render(s)
		}
		return s
	}
	bw := text.Width(badge)
	if badge == "" || bw+2 > avail {
		return text.Fit(head+label(avail), w)
	}
	col := min(max(labelW, 1), avail-bw-1)
	return text.Fit(head+label(col)+" "+badge, w)
}
