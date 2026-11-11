package dashygotui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// onKey routes one key press. The force-quit key works everywhere, including
// over a pop-up; a pop-up then takes every other key; otherwise the view or
// the page decides.
func (m *model[S]) onKey(msg tea.KeyPressMsg) tea.Cmd {
	name := strings.ToLower(msg.String())
	// A shortcut that works everywhere comes first: over pop-ups, in views, and
	// ahead of the force-quit keys, so an application can rebind them.
	for _, s := range m.d.reg.shortcuts {
		if s.Scope == ScopeEverywhere && s.Key == name {
			m.digits = ""
			return s.Run(m.ctx)
		}
	}
	if key.Matches(msg, m.d.keys.Force) {
		m.digits = ""
		if f := m.d.cfg.OnForceQuit; f != nil {
			return f(m.ctx)
		}
		return m.quit(ExitInfo{})
	}
	if m.modal != nil {
		return m.modalKey(msg)
	}
	if m.view != "" {
		return m.viewKey(msg)
	}
	return m.dashKey(msg)
}

// modalKey gives a pop-up the key, after scrolling it when it can scroll.
func (m *model[S]) modalKey(msg tea.KeyPressMsg) tea.Cmd {
	if sm, ok := m.modal.(ScrollableModal[S]); ok && m.scrollKey(msg, sm) {
		return nil
	}
	return m.modal.HandleKey(m.ctx, msg)
}

// scrollKey scrolls a scrollable modal and reports whether it used the key.
func (m *model[S]) scrollKey(msg tea.KeyPressMsg, sm ScrollableModal[S]) bool {
	k := m.d.keys
	limit := sm.ScrollMax(m.ctx, Size{Cols: m.cols, Rows: m.rows})
	switch {
	case key.Matches(msg, k.Up):
		m.modalScroll--
	case key.Matches(msg, k.Down):
		m.modalScroll++
	case key.Matches(msg, k.PageUp):
		m.modalScroll -= 5
	case key.Matches(msg, k.PageDown):
		m.modalScroll += 5
	case key.Matches(msg, k.Home):
		m.modalScroll = 0
	case key.Matches(msg, k.End):
		m.modalScroll = limit
	default:
		return false
	}
	m.modalScroll = min(max(m.modalScroll, 0), max(limit, 0))
	return true
}

// dashKey handles a key press on a dashboard page.
func (m *model[S]) dashKey(msg tea.KeyPressMsg) tea.Cmd {
	name := strings.ToLower(msg.String())
	for _, s := range m.d.reg.shortcuts {
		if s.Scope == ScopePages && s.Key == name {
			m.digits = ""
			return s.Run(m.ctx)
		}
	}
	if o, ok := m.d.hotkeys[name]; ok {
		m.digits = ""
		return o.Run(m.ctx)
	}
	if cmd, ok := m.navKey(msg); ok {
		return cmd
	}
	k := m.d.keys
	switch {
	case key.Matches(msg, k.NextPage):
		m.cyclePage(1)
	case key.Matches(msg, k.PrevPage):
		m.cyclePage(-1)
	case key.Matches(msg, k.Quit):
		return m.quit(ExitInfo{})
	case key.Matches(msg, k.Faster):
		m.stepRate(1)
	case key.Matches(msg, k.Slower):
		m.stepRate(-1)
	case key.Matches(msg, k.Back):
		m.digits = ""
	default:
		m.digitKey(name)
	}
	return nil
}

// digitKey feeds a typed digit to the number jump.
func (m *model[S]) digitKey(name string) {
	if len(name) == 1 && name[0] >= '0' && name[0] <= '9' {
		m.typeDigit(rune(name[0]))
	}
}

// navKey handles the navigation every menu shares and reports whether the key
// was used.
func (m *model[S]) navKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	k := m.d.keys
	switch {
	case key.Matches(msg, k.Up):
		m.moveCursor(-1)
	case key.Matches(msg, k.Down):
		m.moveCursor(1)
	case key.Matches(msg, k.Right):
		m.movePanel(1)
	case key.Matches(msg, k.Left):
		m.movePanel(-1)
	case key.Matches(msg, k.PageDown):
		m.moveCursor(5)
	case key.Matches(msg, k.PageUp):
		m.moveCursor(-5)
	case key.Matches(msg, k.Home):
		m.setCursor(m.sect, 0)
	case key.Matches(msg, k.End):
		m.setCursor(m.sect, len(m.currentItems())-1)
	case key.Matches(msg, k.Select):
		return m.activate(), true
	default:
		return nil, false
	}
	return nil, true
}

// stepRate moves the refresh rate one notch faster (delta > 0) or slower.
func (m *model[S]) stepRate(delta int) {
	steps := RefreshIntervals()
	idx, best := 3, steps[len(steps)-1]*2
	for i, s := range steps {
		if d := absDuration(m.interval - s); d < best {
			idx, best = i, d
		}
	}
	next := idx - delta
	switch {
	case next < 0:
		m.setFlash("Refresh rate: "+formatInterval(steps[0])+" (fastest)", false, FlashHint)
	case next >= len(steps):
		m.setFlash("Refresh rate: "+formatInterval(steps[len(steps)-1])+" (slowest)", false, FlashHint)
	default:
		m.ctx.SetRefreshInterval(steps[next])
		m.setFlash("Refresh rate: "+formatInterval(steps[next]), false, FlashResult)
	}
}

// menus returns the interactive menus of the current view or page, panel by
// panel. Cursor positions index into this list.
func (m *model[S]) menus() [][]Option[S] {
	if m.view != "" {
		v := m.d.views[m.view]
		if v.Body != nil || v.Options == nil {
			return nil
		}
		return [][]Option[S]{v.Options(m.ctx)}
	}
	return m.pageMenus(m.page)
}

// pageMenus returns the menus of the sections on page idx that have options.
func (m *model[S]) pageMenus(idx int) [][]Option[S] {
	var out [][]Option[S]
	for _, p := range m.d.pages[idx].panels {
		if len(p.opts) > 0 {
			out = append(out, p.opts)
		}
	}
	return out
}

// currentItems is the menu the cursor is in.
func (m *model[S]) currentItems() []Option[S] {
	menus := m.menus()
	if m.sect < 0 || m.sect >= len(menus) {
		return nil
	}
	return menus[m.sect]
}

// cursorAt is the cursor row inside the focused menu.
func (m *model[S]) cursorAt() int {
	if m.sect < 0 || m.sect >= len(m.cur) {
		return 0
	}
	return m.cur[m.sect]
}

// resetCursor puts the focus on the first row of the first menu.
func (m *model[S]) resetCursor() {
	m.sect = 0
	m.cur = make([]int, max(len(m.menus()), 1))
	m.digits = ""
}

// setCursor puts the cursor on one row of one menu.
func (m *model[S]) setCursor(sect, idx int) {
	menus := m.menus()
	if sect < 0 || sect >= len(menus) {
		return
	}
	for len(m.cur) < len(menus) {
		m.cur = append(m.cur, 0)
	}
	n := len(menus[sect])
	if n == 0 {
		return
	}
	m.sect = sect
	m.cur[sect] = (idx%n + n) % n
}

// moveCursor steps the cursor within its menu, wrapping at both ends.
func (m *model[S]) moveCursor(delta int) {
	m.digits = ""
	m.setCursor(m.sect, m.cursorAt()+delta)
}

// movePanel moves the focus to the next menu on the page, keeping each menu's
// own row so switching back returns where it was.
func (m *model[S]) movePanel(delta int) {
	menus := m.menus()
	n := len(menus)
	if n < 2 {
		return
	}
	m.digits = ""
	for step := 1; step <= n; step++ {
		cand := ((m.sect+delta*step)%n + n) % n
		if len(menus[cand]) > 0 {
			m.sect = cand
			m.setCursor(cand, m.cursorAt())
			return
		}
	}
}

// activate runs the highlighted option.
func (m *model[S]) activate() tea.Cmd {
	m.digits = ""
	items := m.currentItems()
	idx := m.cursorAt()
	if idx < 0 || idx >= len(items) || items[idx].Run == nil {
		return nil
	}
	return items[idx].Run(m.ctx)
}

// typeDigit extends the number being typed and moves the cursor to the first
// option whose number starts with it. Enter then opens it, so "1" and "13" are
// both reachable with no timer and nothing opens by accident.
func (m *model[S]) typeDigit(d rune) {
	want := m.digits + string(d)
	if pg, s, i, ok := m.findByPrefix(want); ok {
		m.jump(pg, s, i) // jumping clears the typed number, so set it afterwards
		m.digits = want
		return
	}
	// The digits so far cannot be continued; treat this one as a fresh start.
	if pg, s, i, ok := m.findByPrefix(string(d)); ok {
		m.jump(pg, s, i)
		m.digits = string(d)
		return
	}
	m.digits = ""
	m.setFlash(fmt.Sprintf("No option [%s]", string(d)), true, FlashHint)
}

func (m *model[S]) jump(page, sect, idx int) {
	if m.view == "" && page != m.page {
		m.gotoPage(page, sect, idx)
		return
	}
	m.setCursor(sect, idx)
}

// findByPrefix finds the first option whose number begins with prefix: in the
// current view, or on the current page first and then on every other page.
func (m *model[S]) findByPrefix(prefix string) (page, sect, idx int, ok bool) {
	search := func(menus [][]Option[S]) (int, int, bool) {
		for si, menu := range menus {
			for ii, o := range menu {
				if strings.HasPrefix(strconv.Itoa(o.Number), prefix) {
					return si, ii, true
				}
			}
		}
		return 0, 0, false
	}
	if m.view != "" {
		s, i, found := search(m.menus())
		return 0, s, i, found
	}
	if s, i, found := search(m.pageMenus(m.page)); found {
		return m.page, s, i, true
	}
	for p := range m.d.pages {
		if s, i, found := search(m.pageMenus(p)); found {
			return p, s, i, true
		}
	}
	return 0, 0, 0, false
}

// saveCursor remembers the cursor of the current page, so leaving it (to
// another page or into a view) and returning puts the user back.
func (m *model[S]) saveCursor() {
	if m.view != "" {
		return
	}
	m.pageSec[m.page] = m.sect
	m.pageCur[m.page] = append([]int(nil), m.cur...)
}

// restoreCursor puts the cursor of the current page back where it was left.
func (m *model[S]) restoreCursor() {
	menus := m.menus()
	m.cur = make([]int, max(len(menus), 1))
	if saved := m.pageCur[m.page]; len(saved) == len(m.cur) {
		copy(m.cur, saved)
	}
	m.sect = 0
	if s := m.pageSec[m.page]; s >= 0 && s < len(menus) {
		m.sect = s
	}
	m.digits = ""
}

// cyclePage moves to the next (delta > 0) or previous page, wrapping.
func (m *model[S]) cyclePage(delta int) {
	n := len(m.d.pages)
	if n == 0 {
		return
	}
	m.saveCursor()
	m.page = ((m.page+delta)%n + n) % n
	m.restoreCursor()
}

// gotoPage switches to page idx and, when sect is not negative, focuses a
// menu and row on it. The other menus keep their remembered rows.
func (m *model[S]) gotoPage(idx, sect, row int) {
	if idx < 0 || idx >= len(m.d.pages) {
		return
	}
	if idx != m.page {
		m.saveCursor()
		m.page = idx
	}
	m.restoreCursor()
	if sect >= 0 {
		m.setCursor(sect, row)
	}
}

// openView shows a registered view, remembering where it came from.
func (m *model[S]) openView(id string) {
	if id == m.view {
		return
	}
	m.saveCursor()
	m.stack = append(m.stack, m.view)
	m.view = id
	m.resetCursor()
}

// back returns to the view or page the current view was opened from.
func (m *model[S]) back() {
	if m.view == "" {
		return
	}
	prev := ""
	if n := len(m.stack); n > 0 {
		prev = m.stack[n-1]
		m.stack = m.stack[:n-1]
	}
	m.view = prev
	if prev == "" {
		m.restoreCursor()
		return
	}
	m.resetCursor()
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
