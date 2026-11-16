package dashygotui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/layout"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/store"
	"github.com/CeeSmag/dashygotui/theme"
)

// Config describes a dashboard. Map is required; the rest have sensible
// defaults.
type Config[S any] struct {
	// Title names the application in the header and the terminal window title.
	Title string

	// Map is what the dashboard contains and where: its pages, their panels and
	// rows, and the dashboard's shortcuts and hooks. An application writes it in
	// addons/addons.go. Required: it needs at least one page.
	Map Map[S]

	// Poll fetches the application state. It is called at Interval, in the
	// background, and once at start. When nil the state stays the zero value
	// of S and nothing is polled.
	Poll func(ctx context.Context) (S, error)
	// Interval is how often the dashboard refreshes. Default one second. The
	// user can change it at run time; the choice is kept in Store.
	Interval time.Duration
	// PollTimeout is the deadline of one Poll call. Default ten seconds; a
	// negative value means no deadline. A poll that runs past it counts as
	// failed, with an error wrapping [context.DeadlineExceeded], and the next
	// poll starts on schedule. Poll should return when its ctx is done; a call
	// that ignores it is abandoned and its result is discarded, so Poll may be
	// called again while an abandoned call is still running.
	PollTimeout time.Duration

	// Header returns the segments of the header strip. Lower priorities are
	// dropped first on narrow terminals and the first is never dropped. The
	// default shows Title and a clock.
	Header func(*Context[S]) []panel.Segment

	// StartView is a stack of view IDs to open at start, bottom first: the last
	// is the one on screen and Back walks down the rest and then to the pages.
	// Each is opened as if the user had, so OnEnter and OnLeave run. Optional.
	StartView []string

	// Store keeps settings between runs. The default is in-memory.
	Store Store

	// Palettes are extra colour palettes offered next to the built-in ones.
	Palettes []theme.Palette
	// Palette is the ID of the palette used when the user has not chosen one.
	// Default [theme.DefaultID].
	Palette string
	// Glyphs selects the symbol set. The default, [GlyphsAuto], uses the TTY set
	// when the environment looks like the Linux console and the Unicode set
	// otherwise; the user's choice, saved in Store, wins over it.
	Glyphs GlyphMode
	// Background says how a palette's background colour reaches the terminal.
	// The default sets the terminal's own colours. It is the mode the dashboard
	// starts in; the user's saved choice (see stdaddons.Background) wins over it.
	// Use [BackgroundOff] to never touch the terminal's colours.
	Background BackgroundMode
	// DetectTTY decides what [GlyphsAuto] means, replacing the built-in check of
	// the environment. Use it for a test the library cannot make for you, such
	// as asking the operating system whether the terminal is a Linux virtual
	// console. Optional.
	DetectTTY func() bool
	// Getenv reads environment variables for glyph auto-detection. Default
	// [os.Getenv]; tests inject their own.
	Getenv func(string) string
	// OnForceQuit is called instead of quitting when a force-quit key is
	// pressed (ctrl+q and ctrl+c by default), for example to open a "Stop and
	// exit?" confirmation. Return the command to run. Optional.
	OnForceQuit func(*Context[S]) tea.Cmd

	// Keys overrides the default key bindings. Unset bindings keep their
	// defaults.
	Keys KeyMap
	// Spacing overrides the default spacing; the user's saved choice in Store
	// wins over both.
	Spacing *Spacing
	// AltScroll asks the terminal to turn the mouse wheel into arrow keys while
	// the dashboard is up, which keeps text selection with the terminal. It is
	// off by default.
	AltScroll bool
	// Clock supplies the current time. Default [time.Now]; tests inject their
	// own.
	Clock func() time.Time
	// Timer creates the command that delivers a message after d. Default
	// [tea.Tick]. It is a test seam: the dashtest package substitutes timers it
	// can fire on demand, so tests never sleep.
	Timer func(d time.Duration, fn func(time.Time) tea.Msg) tea.Cmd
	// Exec creates the command that hands the terminal to cmd and, when it
	// ends, delivers done's message: what [Context.Exec] runs. Default
	// [tea.Exec]. It is a test seam: the dashtest package records the command
	// instead, and the test decides how it ends.
	Exec func(cmd tea.ExecCommand, done tea.ExecCallback) tea.Cmd
}

// DefaultPollTimeout is the deadline of one Poll call when
// [Config.PollTimeout] is zero.
const DefaultPollTimeout = 10 * time.Second

// Dashboard is a validated, ready-to-run dashboard. Create one with [New].
type Dashboard[S any] struct {
	cfg      Config[S]
	reg      registered[S]
	keys     KeyMap
	pages    []pageDef[S]
	views    map[string]View[S]
	palettes []theme.Palette
	hotkeys  map[string]Option[S] // option hotkeys, by key name

	link sync.Mutex // guards out
	out  Sender     // the running program, for Notify
}

// panelDef is one panel of a page: a list of rows, or a widget.
type panelDef[S any] struct {
	sec    *section[S]
	opts   []Option[S]
	widget *Widget[S]
}

// section is how a panel of rows is drawn: its borders and its badge.
type section[S any] struct {
	title, subtitle, noun string
	badge                 func(*Context[S]) string
}

// pageDef is a page with its panels, as the map placed them.
type pageDef[S any] struct {
	page   Page[S]
	panels []panelDef[S]
}

// New validates cfg and its map and returns a dashboard. Every problem is
// reported at once, wrapped in [ErrInvalidConfig], and nothing starts until
// [Dashboard.Run].
func New[S any](cfg Config[S]) (*Dashboard[S], error) {
	keys := cfg.Keys.withDefaults()
	mp := compileMap(cfg.Map)
	reg, pages := mp.reg.contents(), mp.pages
	var errs []error
	if err := mp.reg.err(); err != nil {
		errs = append(errs, err)
	}
	for _, s := range reg.shortcuts {
		if keys.reserved(s.Key, s.Scope == ScopePages) {
			errs = append(errs, fmt.Errorf("shortcut %q is reserved for navigation or a digit", s.Key))
		}
	}
	hotkeys := make(map[string]Option[S])
	for _, o := range reg.options {
		if o.hot.Key == "" {
			continue
		}
		if keys.reserved(o.hot.Key, true) {
			errs = append(errs, fmt.Errorf("option %q hotkey %q is reserved for navigation or a digit", o.Label, o.hot.Key))
			continue
		}
		hotkeys[o.hot.Key] = o
	}
	viewKnown := func(id string) bool {
		return slices.ContainsFunc(reg.views, func(v View[S]) bool { return v.ID == id })
	}
	for _, o := range reg.options {
		if o.Opens != "" && !viewKnown(o.Opens) {
			errs = append(errs, fmt.Errorf("option %q opens unknown view %q", o.Label, o.Opens))
		}
	}
	for _, s := range reg.shortcuts {
		if s.Opens != "" && !viewKnown(s.Opens) {
			errs = append(errs, fmt.Errorf("shortcut %q opens unknown view %q", s.Key, s.Opens))
		}
	}
	for _, id := range cfg.StartView {
		if !viewKnown(id) {
			errs = append(errs, fmt.Errorf("StartView names unknown view %q", id))
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, errors.Join(errs...))
	}

	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.PollTimeout == 0 {
		cfg.PollTimeout = DefaultPollTimeout
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.Timer == nil {
		cfg.Timer = tea.Tick
	}
	if cfg.Exec == nil {
		cfg.Exec = tea.Exec
	}
	if cfg.Getenv == nil {
		cfg.Getenv = os.Getenv
	}
	if cfg.Store == nil {
		cfg.Store = store.NewMemory()
	}
	views := make(map[string]View[S], len(reg.views))
	for _, v := range reg.views {
		views[v.ID] = v
	}
	return &Dashboard[S]{
		cfg:      cfg,
		reg:      reg,
		keys:     keys,
		pages:    pages,
		views:    views,
		palettes: mergePalettes(append(theme.Builtins(), theme.Console()), cfg.Palettes),
		hotkeys:  hotkeys,
	}, nil
}

// mergePalettes appends extras to base; an extra with an existing ID replaces
// that entry in place.
func mergePalettes(base, extra []theme.Palette) []theme.Palette {
	out := append([]theme.Palette(nil), base...)
	for _, p := range extra {
		if i := slices.IndexFunc(out, func(q theme.Palette) bool { return strings.EqualFold(q.ID, p.ID) }); i >= 0 {
			out[i] = p
			continue
		}
		out = append(out, p)
	}
	return out
}

// layoutMinHeight is the default MinHeight of a widget.
const layoutMinHeight = layout.StatusPanelH
