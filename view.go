package dashygotui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/layout"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
)

// viewKey handles a key press in a full-screen view: the view's own keys first,
// then Back, the shared menu navigation, and Quit.
func (m *model[S]) viewKey(msg tea.KeyPressMsg) tea.Cmd {
	v := m.d.views[m.view]
	if v.Keys != nil {
		if cmd, handled := v.Keys(m.ctx, msg); handled {
			return cmd
		}
	}
	if v.Options != nil && v.Body == nil {
		name := strings.ToLower(msg.String())
		for _, o := range v.Options(m.ctx) {
			if o.Hotkey != "" && o.Hotkey == name && o.Run != nil {
				m.digits = ""
				return o.Run(m.ctx)
			}
		}
	}
	k := m.d.keys
	if v.Scroll && m.scrollViewKey(v, msg) {
		return nil
	}
	if key.Matches(msg, k.Back) {
		m.back()
		return nil
	}
	if cmd, ok := m.navKey(msg); ok {
		return cmd
	}
	if key.Matches(msg, k.Quit) {
		return m.quit(ExitInfo{})
	}
	m.digitKey(strings.ToLower(msg.String()))
	return nil
}

// viewScreen draws a full-screen view: header, body, footer. The body is the
// view's own Body, or a status panel above a menu.
func (m *model[S]) viewScreen(cols, rows int) []string {
	v := m.d.views[m.view]
	th := m.th
	body := max(rows-2, 1) // header and footer
	size := Size{Cols: cols, Rows: body}

	var lines []string
	scrollable := false
	switch {
	case v.Body != nil && v.Scroll:
		all, bar := m.scrollContent(v, cols, body)
		scrollable = bar
		lines = m.scrollWindow(v, all, cols, body, bar)
	case v.Body != nil:
		lines = v.Body(m.ctx, size)
	default:
		badge := ""
		if v.Badge != nil {
			badge = v.Badge(m.ctx)
		}
		var opts []Option[S]
		if v.Options != nil {
			opts = v.Options(m.ctx)
		}
		switch {
		case v.Status != nil && len(opts) > 0:
			status := v.Status(m.ctx, size)
			statusH := min(len(status)+2, max(body-5, 3))
			menuH := max(body-statusH, 3)
			lines = panel.VStack(
				panel.Panel(th, v.Title, badge, v.Subtitle, cols, statusH, status),
				panel.Panel(th, "Options", m.optionsBadge(len(opts)), "", cols, menuH,
					m.menuBody(opts, 0, cols-4, menuH-2, len(opts)+2 <= menuH)),
			)
		case v.Status != nil:
			lines = panel.Panel(th, v.Title, badge, v.Subtitle, cols, body, v.Status(m.ctx, size))
		case len(opts) > 0:
			lines = panel.Panel(th, v.Title, badge, v.Subtitle, cols, body,
				m.menuBody(opts, 0, cols-4, body-2, len(opts)+2 <= body))
		}
	}

	segs := []panel.Segment{
		{Text: th.White.Render(m.title()), Prio: 100},
		{Text: th.Title.Render(v.Title), Prio: 90},
		{Text: th.Muted.Render(m.clock()), Prio: 40},
	}
	var hints []panel.Hint
	if v.Options != nil && v.Body == nil {
		hints = append(hints, m.hint(th.G.UpDown, "Move", th.Key, 60), m.hint(th.G.Enter, "Open", th.Key, 55))
	}
	if scrollable {
		hints = append(hints, m.hint(th.G.UpDown, "Scroll", th.Key, 60))
		if v.Tail && m.viewScroll > 0 {
			hints = append(hints, m.hint("End", "Live", th.Key, 58))
		}
	}
	for _, s := range m.d.reg.shortcuts {
		if s.Scope == ScopeEverywhere && s.Label != "" {
			prio := s.Prio
			if prio == 0 {
				prio = 30
			}
			hints = append(hints, m.hint(s.Key, s.Label, th.Key, prio))
		}
	}
	hints = append(hints, m.hint("Esc/b", "Back", th.Key, 1000), m.hint("q", "Quit", th.Red, 900))

	mode := v.Mode
	if mode == "" {
		mode = "VIEW"
	}
	out := append([]string{panel.Header(th, segs, cols)}, panel.Frame(lines, cols, body)...)
	return append(out, m.footer(cols, mode, hints))
}

func (m *model[S]) optionsBadge(n int) string {
	return m.ctx.Count(n, map[bool]string{true: "option", false: "options"}[n == 1])
}

// scrollContent runs a scrolling view's Body. It is given the full width; when
// the result does not fit in body rows it is run again one column narrower,
// leaving room for the scroll bar, and bar is true.
func (m *model[S]) scrollContent(v View[S], cols, body int) (lines []string, bar bool) {
	lines = v.Body(m.ctx, Size{Cols: cols, Rows: body})
	if len(lines) <= body || cols < 2 {
		return lines, false
	}
	return v.Body(m.ctx, Size{Cols: cols - 1, Rows: body}), true
}

// scrollMax is the furthest a scrolling view can be scrolled at this size.
func (m *model[S]) scrollMax(v View[S]) int {
	body := max(m.rows-2, 1)
	lines, bar := m.scrollContent(v, m.cols, body)
	if !bar {
		return 0
	}
	return max(len(lines)-body, 0)
}

// scrollWindow cuts the visible rows out of a scrolling body and, when there
// is more than fits, draws the scroll bar down the right-hand edge.
func (m *model[S]) scrollWindow(v View[S], all []string, cols, body int, bar bool) []string {
	if !bar {
		return all
	}
	th := m.th
	total := len(all)
	off := m.topOffset(v, total, body)
	thumb := min(max(body*body/total, 1), body)
	top := 0
	if total > body {
		top = off * (body - thumb) / (total - body)
	}
	out := make([]string, body)
	for i := range body {
		cell := th.Border.Render("│")
		if i >= top && i < top+thumb {
			cell = th.Key.Render(th.G.MeterFill)
		}
		out[i] = text.Pad(text.Fit(all[off+i], cols-1), cols-1) + cell
	}
	return out
}

// scrollViewKey scrolls the open view and reports whether it used the key.
func (m *model[S]) scrollViewKey(v View[S], msg tea.KeyPressMsg) bool {
	k := m.d.keys
	limit := m.scrollMax(v)
	page := max(m.rows-3, 1)
	// toward is the sign of "further along the content"; in a Tail view the
	// offset counts back from the bottom, so down means a smaller offset.
	toward := 1
	if v.Tail {
		toward = -1
	}
	switch {
	case key.Matches(msg, k.Up):
		m.viewScroll -= toward
	case key.Matches(msg, k.Down):
		m.viewScroll += toward
	case key.Matches(msg, k.PageUp):
		m.viewScroll -= toward * page
	case key.Matches(msg, k.PageDown):
		m.viewScroll += toward * page
	case key.Matches(msg, k.Home): // the start of the content
		m.viewScroll = 0
		if v.Tail {
			m.viewScroll = limit
		}
	case key.Matches(msg, k.End): // the end of the content
		m.viewScroll = limit
		if v.Tail {
			m.viewScroll = 0
		}
	default:
		return false
	}
	m.viewScroll = min(max(m.viewScroll, 0), limit)
	return true
}

// topOffset is how many rows of a total-row body are cut off the top.
func (m *model[S]) topOffset(v View[S], total, body int) int {
	furthest := max(total-body, 0)
	off := min(max(m.viewScroll, 0), furthest)
	if v.Tail {
		return furthest - off
	}
	return off
}

// holdTail keeps a Tail view that has been scrolled back where it is while rows
// are appended, by growing its offset from the bottom by the rows added. It
// runs after every message, and only does anything for such a view.
func (m *model[S]) holdTail() {
	if m.view == "" {
		return
	}
	v := m.d.views[m.view]
	if !v.Tail || !v.Scroll || v.Body == nil || m.cols < layout.MinCols || m.rows < layout.MinRows {
		return
	}
	lines, _ := m.scrollContent(v, m.cols, max(m.rows-2, 1))
	if m.viewScroll > 0 && m.viewTotal > 0 && len(lines) > m.viewTotal {
		m.viewScroll += len(lines) - m.viewTotal
	}
	m.viewTotal = len(lines)
}
