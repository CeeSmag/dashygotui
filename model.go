package dashygotui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/layout"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// Messages the model sends itself. Their types are unexported so nothing
// outside the package can forge them.
type (
	tickMsg         time.Time
	pollDueMsg      struct{}
	stateMsg[S any] struct {
		state S
		err   error
	}
	taskMsg struct{ res TaskResult }
	// notifyMsg is what [Dashboard.Notify] sends: the world changed, refresh.
	notifyMsg struct{}
)

// RefreshIntervals returns the refresh intervals the +/- keys step through,
// fastest first. It returns a fresh slice on every call.
func RefreshIntervals() []time.Duration {
	return []time.Duration{
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		time.Second,
		2 * time.Second,
		5 * time.Second,
	}
}

// model is the Bubble Tea model. It owns every piece of mutable dashboard
// state, and only Update changes it.
type model[S any] struct {
	d   *Dashboard[S]
	ctx *Context[S] // handed to every addon callback

	// run is the program's context, used to cancel background work. Bubble Tea
	// commands cannot be created from Update without it.
	run context.Context

	cols, rows int

	state    S
	stateErr error
	ready    bool // a poll has succeeded at least once
	polling  bool // a poll is in flight
	dueSet   bool // a pollDueMsg timer is pending
	repoll   bool // a refresh was asked for while a poll was in flight
	interval time.Duration
	ticks    int

	th            *theme.Theme
	palette       string
	paletteChosen bool // the user or the config picked a palette; no TTY default
	glyphMode     GlyphMode
	bgMode        BackgroundMode
	tty           bool // the TTY glyph set is in use
	spacing       Spacing
	store         Store
	startErr      error // a setting could not be loaded at start

	// Where the user is.
	view    string   // "" is the dashboard pages
	stack   []string // views to come back to
	page    int
	pageCur [][]int // remembered cursor rows per page
	pageSec []int   // remembered focused panel per page
	sect    int     // focused menu
	cur     []int   // cursor row per menu
	digits  string  // the number typed so far

	modal       Modal[S]
	modalScroll int

	flash      string
	flashBad   bool
	flashUntil time.Time
	pending    string // the footer message of the task in flight, cleared when it ends

	// queue holds commands that code running inside Update produced without
	// being able to return them (view hooks, for one). Update sends them on.
	queue []tea.Cmd

	viewScroll int // rows scrolled off the top of the open view's body

	token    int
	quitting bool
	exit     ExitInfo
}

// newModel builds a fresh model on the dashboard. run cancels background work.
func newModel[S any](run context.Context, d *Dashboard[S]) *model[S] {
	m := &model[S]{
		d:        d,
		run:      run,
		interval: d.cfg.Interval,
		store:    d.cfg.Store,
		spacing:  DefaultSpacing(),
		pageCur:  make([][]int, len(d.pages)),
		pageSec:  make([]int, len(d.pages)),
		palette:  theme.DefaultID,
	}
	m.ctx = &Context[S]{m: m}
	if d.cfg.Spacing != nil {
		m.spacing = d.cfg.Spacing.Clamp()
	}
	if id := d.cfg.Palette; id != "" {
		m.palette, m.paletteChosen = id, true
	}
	m.glyphMode = d.cfg.Glyphs
	m.bgMode = d.cfg.Background
	m.loadSettings()
	m.rebuildTheme()
	m.resetCursor()
	return m
}

// loadSettings overlays the user's saved choices on the configured defaults. A
// store that fails to load leaves the defaults and is reported in the footer.
func (m *model[S]) loadSettings() {
	var sp Spacing
	if ok, err := m.store.Get(storeKeySpacing, &sp); err != nil {
		m.startErr = err
	} else if ok {
		m.spacing = sp.Clamp()
	}
	var id string
	if ok, err := m.store.Get(storeKeyPalette, &id); err != nil {
		m.startErr = err
	} else if ok && id != "" {
		m.palette, m.paletteChosen = id, true
	}
	var gm string
	if ok, err := m.store.Get(storeKeyGlyphs, &gm); err != nil {
		m.startErr = err
	} else if ok {
		m.glyphMode = parseGlyphMode(gm)
	}
	var bm string
	if ok, err := m.store.Get(storeKeyBG, &bm); err != nil {
		m.startErr = err
	} else if ok {
		m.bgMode = parseBackgroundMode(bm)
	}
	var ms int
	if ok, err := m.store.Get(storeKeyInterval, &ms); err != nil {
		m.startErr = err
	} else if ok && ms > 0 {
		m.interval = time.Duration(ms) * time.Millisecond
	}
}

// resolveTTY decides whether the TTY glyph set is in use.
func (m *model[S]) resolveTTY() bool {
	switch m.glyphMode {
	case GlyphsUnicode:
		return false
	case GlyphsTTY:
		return true
	}
	return theme.DetectTTY(m.d.cfg.Getenv)
}

// rebuildTheme resolves the glyph mode and the palette and builds the theme. On
// a TTY with no palette chosen it uses the 16-colour console palette, which the
// Linux console can actually show.
func (m *model[S]) rebuildTheme() {
	m.tty = m.resolveTTY()
	id := m.palette
	if m.tty && !m.paletteChosen {
		id = "console" // the default for a TTY; m.palette keeps the base choice
	}
	m.buildTheme(id)
}

// buildTheme builds the theme for palette id, falling back to the default
// palette when id is unknown. It reports whether id was found.
func (m *model[S]) buildTheme(id string) bool {
	p, ok := theme.Lookup(m.d.palettes, id)
	if !ok {
		p, _ = theme.Lookup(m.d.palettes, theme.DefaultID)
		if p.ID == "" {
			p = theme.Default()
		}
	}
	m.th = theme.New(p, m.tty)
	return ok
}

// applyPalette makes palette id the chosen one and builds the theme from it.
func (m *model[S]) applyPalette(id string) bool {
	ok := m.buildTheme(id)
	m.palette = m.th.Palette.ID
	return ok
}

func (m *model[S]) title() string {
	if m.d.cfg.Title != "" {
		return m.d.cfg.Title
	}
	return "Dashboard"
}

func (m *model[S]) now() time.Time { return m.d.cfg.Clock() }

// Init implements tea.Model. It starts the clock and the first poll.
func (m *model[S]) Init() tea.Cmd {
	if m.startErr != nil {
		m.setFlash("settings not loaded: "+text.OneLine(m.startErr.Error()), true, FlashResult)
	}
	return tea.Batch(m.tickCmd(), m.pollCmd())
}

func (m *model[S]) tickCmd() tea.Cmd {
	return m.d.cfg.Timer(m.interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// pollCmd starts a poll unless one is already running or there is nothing to
// poll.
func (m *model[S]) pollCmd() tea.Cmd {
	poll := m.d.cfg.Poll
	if poll == nil || m.polling {
		return nil
	}
	m.polling = true
	run := m.run
	return func() tea.Msg {
		s, err := poll(run)
		return stateMsg[S]{state: s, err: err}
	}
}

// refresh polls now. A poll already in flight may have read the data before
// whatever prompted this, so it is repeated once when it ends.
func (m *model[S]) refresh() tea.Cmd {
	if m.d.cfg.Poll == nil {
		return nil
	}
	if m.polling {
		m.repoll = true
		return nil
	}
	return m.pollCmd()
}

// scheduleDue arranges the next poll one interval from now, once.
func (m *model[S]) scheduleDue() tea.Cmd {
	if m.d.cfg.Poll == nil || m.dueSet {
		return nil
	}
	m.dueSet = true
	return m.d.cfg.Timer(m.interval, func(time.Time) tea.Msg { return pollDueMsg{} })
}

// Update implements tea.Model. Every state change in the dashboard happens
// here, on the Bubble Tea loop.
func (m *model[S]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	if len(m.queue) > 0 {
		cmd = tea.Batch(append([]tea.Cmd{cmd}, m.queue...)...)
		m.queue = nil
	}
	return next, cmd
}

// later queues cmd to be sent with whatever the current Update returns.
func (m *model[S]) later(cmd tea.Cmd) {
	if cmd != nil {
		m.queue = append(m.queue, cmd)
	}
}

func (m *model[S]) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.cols, m.rows = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		return m, m.onTick(time.Time(msg))
	case pollDueMsg:
		m.dueSet = false
		return m, m.pollCmd()
	case stateMsg[S]:
		m.polling = false
		if msg.err != nil {
			m.stateErr = msg.err
		} else {
			m.state, m.stateErr, m.ready = msg.state, nil, true
		}
		if m.repoll { // a refresh was asked for mid-poll: the answer may already be stale
			m.repoll = false
			return m, tea.Batch(m.pollCmd(), m.scheduleDue())
		}
		return m, m.scheduleDue()
	case notifyMsg:
		return m, m.refresh()
	case taskMsg:
		return m, m.onResult(msg.res)
	case tea.KeyPressMsg:
		return m, m.onKey(msg)
	}
	for _, hook := range m.d.reg.onMsg {
		if cmd := hook(m.ctx, msg); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

// onTick advances everything that moves on its own.
func (m *model[S]) onTick(now time.Time) tea.Cmd {
	m.ticks++
	if m.flash != "" && now.After(m.flashUntil) {
		m.flash, m.flashBad = "", false
	}
	cmds := []tea.Cmd{m.tickCmd()}
	for _, hook := range m.d.reg.onTick {
		if cmd := hook(m.ctx, now); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// onResult routes a finished task: the open modal first, then the registered
// hooks, then the footer.
func (m *model[S]) onResult(res TaskResult) tea.Cmd {
	// A task is over: take its "working..." message off the footer, whoever
	// handles the result. A handler that has something to say replaces it.
	if m.pending != "" && m.flash == m.pending {
		m.flash, m.flashBad = "", false
	}
	m.pending = ""
	if rm, ok := m.modal.(ResultModal[S]); ok {
		if handled, cmd := rm.OnResult(m.ctx, res); handled {
			return cmd
		}
	}
	for _, hook := range m.d.reg.onResult {
		if handled, cmd := hook(m.ctx, res); handled {
			return cmd
		}
	}
	m.flashResult(res.Text, res.Err)
	if res.Err == nil {
		return m.pollCmd()
	}
	return nil
}

// runTask starts fn in the background and reports its result as a taskMsg.
func (m *model[S]) runTask(tag, pending string, fn func(context.Context) (string, error)) tea.Cmd {
	if pending != "" {
		m.setFlash(pending, false, 30*time.Second)
		m.pending = pending
	}
	m.token++
	token, run := m.token, m.run
	return func() tea.Msg {
		out, err := fn(run)
		return taskMsg{res: TaskResult{Token: token, Tag: tag, Text: out, Err: err}}
	}
}

// setFlash shows a message in the footer for d.
func (m *model[S]) setFlash(msg string, bad bool, d time.Duration) {
	m.flash, m.flashBad = msg, bad
	m.flashUntil = m.now().Add(d)
}

// flashResult shows the outcome of an action: a tick for success, a cross and
// the reason for a failure.
func (m *model[S]) flashResult(msg string, err error) {
	switch {
	case err != nil:
		out := text.OneLine(err.Error())
		if msg != "" {
			out = text.OneLine(msg) + " " + out
		}
		m.setFlash(m.th.G.Cross+" "+out, true, FlashResult)
	case msg != "":
		m.setFlash(m.th.G.Check+" "+text.OneLine(msg), false, FlashResult)
	}
}

// ExitInfo reports why the dashboard asked to quit. It is meant for programs
// that embed the model from [Dashboard.Model] and for the dashtest package:
// assert the model to interface{ ExitInfo() ExitInfo }.
func (m *model[S]) ExitInfo() ExitInfo { return m.exit }

func (m *model[S]) quit(info ExitInfo) tea.Cmd {
	m.exit = info
	m.quitting = true
	return tea.Quit
}

// View implements tea.Model. The mouse is never captured, so the terminal
// keeps selection and copy.
func (m *model[S]) View() tea.View {
	v := tea.NewView(m.render(m.cols, m.rows))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeNone
	v.WindowTitle = m.title()
	if m.bgMode != BackgroundOff {
		if bg, fg, ok := m.th.Backdrop(); ok {
			v.BackgroundColor, v.ForegroundColor = bg, fg
		}
	}
	return v
}

// render is the frame as one string: rows lines of cols cells joined by
// newlines.
func (m *model[S]) render(cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}
	frame := panel.Frame(m.screen(cols, rows), cols, rows)
	if m.bgMode == BackgroundPaint {
		frame = m.th.Paint(frame)
	}
	return strings.Join(frame, "\n")
}

// screen builds the active page or view and, when one is open, the pop-up over
// it.
func (m *model[S]) screen(cols, rows int) []string {
	if cols < layout.MinCols || rows < layout.MinRows {
		return panel.TooSmall(m.th, cols, rows, layout.MinCols, layout.MinRows)
	}
	var s []string
	if m.view != "" {
		s = m.viewScreen(cols, rows)
	} else {
		s = m.dashScreen(cols, rows)
	}
	if m.modal != nil {
		box := m.modal.Render(m.ctx, Size{Cols: cols, Rows: rows})
		return panel.Overlay(panel.Frame(s, cols, rows), box, cols, rows)
	}
	return s
}

// formatInterval renders a refresh interval for the footer, e.g. "1s", "250ms".
func formatInterval(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%gs", d.Seconds())
}
