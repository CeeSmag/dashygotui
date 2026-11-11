package dashygotui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/panel"
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
	switch {
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
