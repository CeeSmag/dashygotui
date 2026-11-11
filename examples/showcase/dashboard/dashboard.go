// Package dashboard is the foundation of the showcase application: it declares
// the pages and the sections on them, owns the one Registry, and builds the
// dashboard. It never imports the addons package; main does that, with a blank
// import, so each addon registers itself.
//
// Everything the library can do is used somewhere in this application. The
// comments on each page below say which features it demonstrates.
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/theme"
)

// Process is one row of the fake process table.
type Process struct {
	PID   int
	Name  string
	CPU   float64 // percent
	MemMB int
}

// State is everything the dashboard shows. Poll fills it in; addons only read
// it. It is synthetic data so the showcase runs anywhere.
type State struct {
	At        time.Time
	CPU       float64 // percent
	Mem       float64 // percent
	Disk      float64 // percent
	NetIn     float64 // bytes per second
	NetOut    float64 // bytes per second
	Uptime    time.Duration
	Requests  int
	Processes []Process
}

// Registry is the application's single registry. Addons register with it from
// init(); call its methods directly so errors name the addon's own file.
var Registry = dashygotui.NewRegistry[State]()

// Logs collects the application's log output. main diverts the standard logger
// into it before the dashboard starts, and the Logs view shows it.
var Logs = logs.NewBuffer(5000)

// Slog is a structured logger that writes into Logs, next to the standard one.
var Slog = slog.New(Logs.Handler(&slog.HandlerOptions{Level: slog.LevelDebug}))

// Short names for the generic types, so addon files stay terse.
type (
	// Ctx is the handle every addon callback receives.
	Ctx = dashygotui.Context[State]
	// Option is a numbered row in a section.
	Option = dashygotui.Option[State]
	// Widget is a read-only panel on a page.
	Widget = dashygotui.Widget[State]
	// View is a full-screen view.
	View = dashygotui.View[State]
	// Shortcut is a key binding that is not a menu row.
	Shortcut = dashygotui.Shortcut[State]
	// Modal is a pop-up.
	Modal = dashygotui.Modal[State]
)

// OutageKey is the Store key that makes Poll fail, to show a stale dashboard.
const OutageKey = "sim.outage"

// Options are the settings main takes from its command line.
type Options struct {
	Store      dashygotui.Store
	Glyphs     dashygotui.GlyphMode
	Background dashygotui.BackgroundMode
	Palette    string
	AltScroll  bool
}

// Solarized is a palette this application adds to the built-in ones. It sets a
// background, so choosing it recolours the whole terminal.
func Solarized() theme.Palette {
	return theme.Palette{
		ID: "solarized", Name: "Solarized Dark", Vibe: "Precision colours for machines and people",
		Border: "#268BD2", Cyan: "#2AA198", Green: "#859900", Mint: "#2AA198", Amber: "#B58900",
		Orange: "#CB4B16", Red: "#DC322F", Muted: "#586E75", White: "#EEE8D5", Track: "#073642",
		GradLow: [3]uint8{38, 139, 210}, GradMid: [3]uint8{133, 153, 0},
		GradWarn: [3]uint8{181, 137, 0}, GradPeak: [3]uint8{220, 50, 47},
		Background: "#002B36", Foreground: "#93A1A1",
	}
}

// Config returns the dashboard's configuration.
//
// Pages show different layouts, tab badges and short titles; sections show
// custom badges and nouns; the config shows a custom palette, a rebound key, a
// force-quit hook, glyph and background modes, and a custom header.
func Config(o Options) dashygotui.Config[State] {
	st := o.Store
	return dashygotui.Config[State]{
		Title:    "Showcase",
		Registry: Registry,
		Store:    st,
		Poll:     poll(time.Now(), st),
		Interval: time.Second,

		Glyphs:     o.Glyphs,
		Background: o.Background,
		Palette:    o.Palette,
		AltScroll:  o.AltScroll,
		Palettes:   []theme.Palette{Solarized()},

		// Backspace goes back too, as well as Esc and b.
		Keys: dashygotui.KeyMap{
			Back: key.NewBinding(key.WithKeys("esc", "b", "backspace"), key.WithHelp("esc", "back")),
		},

		// ctrl+c is the library's force-quit key. Ask before quitting. (ctrl+q is
		// a global shortcut of its own: see shortcut_stop.go.)
		OnForceQuit: func(c *Ctx) tea.Cmd {
			c.OpenModal(&modal.Confirm[State]{
				Title:    "Quit the dashboard?",
				Lines:    []string{"The service keeps running. Use ctrl+q to stop it as well."},
				Accept:   "Quit",
				OnAccept: func(c *Ctx) tea.Cmd { return c.Quit("", "Goodbye.") },
			})
			return nil
		},

		Header: func(c *Ctx) []panel.Segment {
			th := c.Theme()
			status := th.Green.Render(th.G.Bullet + " ONLINE")
			if c.Err() != nil {
				status = th.Red.Render(th.G.Cross + " STALE")
			}
			return []panel.Segment{
				{Text: th.White.Render("Showcase v1"), Prio: 100},
				{Text: status, Prio: 90},
				{Text: th.Cyan.Render("example.local"), Prio: 60},
				{Text: th.Green.Render(c.Now().Format("15:04:05")), Prio: 80},
			}
		},

		Pages: []dashygotui.Page[State]{
			// Four widgets in a 2x2 grid. The tab shows whether data is flowing.
			{ID: "monitor", Title: "Monitor", ShortTitle: "Mon", TinyTitle: "M",
				Layout: dashygotui.LayoutGrid2x2,
				Badge: func(c *Ctx) string {
					th := c.Theme()
					if c.Err() != nil {
						return th.Red.Render("STALE")
					}
					return th.Green.Render("LIVE")
				}},
			// Three chart panels stacked.
			{ID: "charts", Title: "Charts", ShortTitle: "Chrt", TinyTitle: "C", Layout: dashygotui.LayoutRows},
			// Three sections: Auto puts one above two.
			{ID: "actions", Title: "Actions", ShortTitle: "Act", TinyTitle: "A", Layout: dashygotui.LayoutAuto},
			// Two sections side by side.
			{ID: "popups", Title: "Pop-ups", ShortTitle: "Pop", TinyTitle: "P", Layout: dashygotui.LayoutColumns},
			// Two above one.
			{ID: "views", Title: "Views & Hooks", ShortTitle: "View", TinyTitle: "V", Layout: dashygotui.LayoutTwoTopOneBottom},
			// One section and two widgets.
			{ID: "theme", Title: "Theme", ShortTitle: "Thm", TinyTitle: "T", Layout: dashygotui.LayoutOneTopTwoBottom,
				Badge: func(c *Ctx) string { return c.Theme().Muted.Render(c.PaletteID()) }},
		},

		Sections: []dashygotui.SectionSpec[State]{
			// Noun changes the default "[3 options]" badge to "[3 actions]".
			{ID: "ops", Page: "actions", Order: 10, Title: "Quick Actions", Subtitle: "things to do", Noun: "action"},
			// Badge replaces the count badge entirely.
			{ID: "settings", Page: "actions", Order: 20, Title: "Settings", Subtitle: "saved between runs",
				Badge: func(c *Ctx) string { return c.Theme().Muted.Render("[persisted]") }},
			{ID: "danger", Page: "actions", Order: 30, Title: "Danger Zone", Subtitle: "think twice"},

			{ID: "standard", Page: "popups", Order: 10, Title: "Standard Pop-ups", Subtitle: "from the modal package"},
			{ID: "custom", Page: "popups", Order: 20, Title: "Custom Pop-ups", Subtitle: "written for this app"},

			{ID: "views", Page: "views", Order: 10, Title: "Full-screen Views", Subtitle: "Esc, b or Backspace returns"},
			{ID: "navigation", Page: "views", Order: 20, Title: "Navigation", Subtitle: "Order reorders; numbers do not"},
			{ID: "hooks", Page: "views", Order: 30, Title: "Hooks & Tasks", Subtitle: "messages, results, processes"},

			{ID: "appearance", Page: "theme", Order: 10, Title: "Appearance", Subtitle: "colours, spacing, symbols"},
		},
	}
}

// Run shows the dashboard and returns why it ended.
func Run(ctx context.Context, o Options) (dashygotui.ExitInfo, error) {
	d, err := dashygotui.New(Config(o))
	if err != nil {
		return dashygotui.ExitInfo{}, fmt.Errorf("building the dashboard: %w", err)
	}
	return d.Run(ctx)
}

// poll returns a function that produces gently moving synthetic readings. It
// fails while the simulated outage flag is set in the store.
func poll(start time.Time, st dashygotui.Store) func(context.Context) (State, error) {
	requests := 0
	return func(context.Context) (State, error) {
		var outage bool
		if st != nil {
			_, _ = st.Get(OutageKey, &outage)
		}
		if outage {
			return State{}, errors.New("backend unreachable (simulated outage)")
		}
		now := time.Now()
		t := now.Sub(start).Seconds()
		requests += 3 + int(2*math.Sin(t/3)+2)
		return State{
			At:        now,
			CPU:       percent(38 + 26*math.Sin(t/7) + 8*math.Sin(t*1.7)),
			Mem:       percent(54 + 6*math.Sin(t/19)),
			Disk:      percent(71 + 0.2*math.Sin(t/60)),
			NetIn:     math.Max(0, 420_000+380_000*math.Sin(t/5)+90_000*math.Sin(t*2.3)),
			NetOut:    math.Max(0, 120_000+100_000*math.Sin(t/6+1)),
			Uptime:    now.Sub(start),
			Requests:  requests,
			Processes: processes(t),
		}, nil
	}
}

// processes returns a fake process table whose numbers move with time.
func processes(t float64) []Process {
	base := []struct {
		name string
		mem  int
	}{
		{"api-gateway", 412}, {"postgres", 1280}, {"redis", 96}, {"worker-1", 220}, {"worker-2", 218},
		{"nginx", 64}, {"prometheus", 540}, {"grafana", 310}, {"cron", 4}, {"sshd", 8},
		{"backup-agent", 72}, {"log-shipper", 120},
	}
	out := make([]Process, len(base))
	for i, b := range base {
		phase := float64(i) * 0.9
		out[i] = Process{
			PID:   1000 + i*37,
			Name:  b.name,
			CPU:   percent(8 + 7*float64(len(base)-i)/float64(len(base))*(1+math.Sin(t/4+phase))),
			MemMB: b.mem + int(6*math.Sin(t/9+phase)),
		}
	}
	return out
}

// percent forces v into the range 0 to 100.
func percent(v float64) float64 { return math.Min(math.Max(v, 0), 100) }
