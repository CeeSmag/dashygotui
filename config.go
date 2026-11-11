package dashygotui

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/layout"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/store"
	"github.com/CeeSmag/dashygotui/theme"
)

// Config describes a dashboard. Registry, Pages and Sections are required; the
// rest have sensible defaults.
type Config[S any] struct {
	// Title names the application in the header and the terminal window title.
	Title string

	// Registry holds the addons. Required. Applications pass their one global
	// registry; tests pass a fresh one.
	Registry *Registry[S]

	// Pages are the dashboard tabs, in tab order. At least one is required.
	Pages []Page[S]
	// Sections are the option containers on the pages. Options and widgets
	// refer to pages and sections declared here by ID.
	Sections []SectionSpec[S]

	// Poll fetches the application state. It is called at Interval, in the
	// background, and once at start. When nil the state stays the zero value
	// of S and nothing is polled.
	Poll func(ctx context.Context) (S, error)
	// Interval is how often the dashboard refreshes. Default one second. The
	// user can change it at run time; the choice is kept in Store.
	Interval time.Duration

	// Header returns the segments of the header strip. Lower priorities are
	// dropped first on narrow terminals and the first is never dropped. The
	// default shows Title and a clock.
	Header func(*Context[S]) []panel.Segment

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
	// The default sets the terminal's own colours.
	Background BackgroundMode
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
}

// Dashboard is a validated, ready-to-run dashboard. Create one with [New].
type Dashboard[S any] struct {
	cfg      Config[S]
	reg      registered[S]
	keys     KeyMap
	pages    []pageDef[S]
	views    map[string]View[S]
	palettes []theme.Palette
	hotkeys  map[string]Option[S] // option hotkeys, by key name
}

// panelDef is one panel of a page: a section with its options, or a widget.
type panelDef[S any] struct {
	id     string
	order  int
	sec    *SectionSpec[S]
	opts   []Option[S]
	widget *Widget[S]
}

// pageDef is a page with its panels already collected and sorted.
type pageDef[S any] struct {
	page   Page[S]
	panels []panelDef[S]
}

// New validates cfg and its registry and returns a dashboard. Every problem is
// reported at once, wrapped in [ErrInvalidConfig], and nothing starts until
// [Dashboard.Run].
//
// New freezes the registry: registering after it is an error.
func New[S any](cfg Config[S]) (*Dashboard[S], error) {
	if cfg.Registry == nil {
		return nil, fmt.Errorf("%w: Registry is nil", ErrInvalidConfig)
	}
	reg := cfg.Registry.freeze()
	keys := cfg.Keys.withDefaults()

	var errs []error
	if err := cfg.Registry.Validate(); err != nil {
		errs = append(errs, err)
	}
	pages, pageErrs := buildPages(cfg, reg)
	errs = append(errs, pageErrs...)
	for _, s := range reg.shortcuts {
		if keys.reserved(s.Key, s.Scope == ScopePages) {
			errs = append(errs, fmt.Errorf("shortcut %q is reserved for navigation or a digit", s.Key))
		}
	}
	hotkeys := make(map[string]Option[S])
	for _, o := range reg.options {
		if o.Hotkey == "" {
			continue
		}
		if keys.reserved(o.Hotkey, true) {
			errs = append(errs, fmt.Errorf("option %q hotkey %q is reserved for navigation or a digit", o.Key, o.Hotkey))
			continue
		}
		hotkeys[o.Hotkey] = o
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, errors.Join(errs...))
	}

	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.Timer == nil {
		cfg.Timer = tea.Tick
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

// buildPages checks the declared pages and sections against each other and the
// registry, and assembles the sorted panels of every page.
func buildPages[S any](cfg Config[S], reg registered[S]) ([]pageDef[S], []error) {
	var errs []error
	if len(cfg.Pages) == 0 {
		errs = append(errs, errors.New("no Pages declared"))
	}

	pageIdx := map[string]int{}
	pages := make([]pageDef[S], len(cfg.Pages))
	for i, p := range cfg.Pages {
		switch {
		case strings.TrimSpace(p.ID) == "":
			errs = append(errs, fmt.Errorf("page #%d has no ID", i+1))
		case p.Title == "":
			errs = append(errs, fmt.Errorf("page %q has no Title", p.ID))
		}
		if _, dup := pageIdx[p.ID]; dup {
			errs = append(errs, fmt.Errorf("duplicate page ID %q", p.ID))
		}
		pageIdx[p.ID] = i
		pages[i] = pageDef[S]{page: p}
	}

	secOpts := map[string][]Option[S]{}
	for _, o := range reg.options {
		secOpts[o.Section] = append(secOpts[o.Section], o)
	}

	secIDs := map[string]bool{}
	for i := range cfg.Sections {
		s := cfg.Sections[i]
		switch {
		case strings.TrimSpace(s.ID) == "":
			errs = append(errs, fmt.Errorf("section #%d has no ID", i+1))
			continue
		case secIDs[s.ID]:
			errs = append(errs, fmt.Errorf("duplicate section ID %q", s.ID))
			continue
		}
		secIDs[s.ID] = true
		if s.Title == "" {
			errs = append(errs, fmt.Errorf("section %q has no Title", s.ID))
		}
		pi, ok := pageIdx[s.Page]
		if !ok {
			errs = append(errs, fmt.Errorf("section %q is on unknown page %q", s.ID, s.Page))
			continue
		}
		opts := append([]Option[S](nil), secOpts[s.ID]...)
		slices.SortStableFunc(opts, func(a, b Option[S]) int {
			return cmp.Or(cmp.Compare(a.order(), b.order()), cmp.Compare(a.Number, b.Number), cmp.Compare(a.Key, b.Key))
		})
		pages[pi].panels = append(pages[pi].panels, panelDef[S]{id: s.ID, order: s.Order, sec: &s, opts: opts})
	}
	for _, o := range reg.options {
		if !secIDs[o.Section] {
			errs = append(errs, fmt.Errorf("option %q targets unknown section %q", o.Key, o.Section))
		}
	}
	for i := range reg.widgets {
		w := reg.widgets[i]
		pi, ok := pageIdx[w.Page]
		if !ok {
			errs = append(errs, fmt.Errorf("widget %q targets unknown page %q", w.Key, w.Page))
			continue
		}
		pages[pi].panels = append(pages[pi].panels, panelDef[S]{id: w.Key, order: w.Order, widget: &w})
	}
	for i := range pages {
		slices.SortStableFunc(pages[i].panels, func(a, b panelDef[S]) int {
			return cmp.Or(cmp.Compare(a.order, b.order), cmp.Compare(a.id, b.id))
		})
	}
	return pages, errs
}

// layoutMinHeight is the default MinHeight of a widget.
const layoutMinHeight = layout.StatusPanelH
