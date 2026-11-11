// Package dashboard is the foundation of the demo application's dashboard: it
// declares the pages and the sections on them, owns the one Registry, and
// builds the dashboard. It never imports the addons package; main does that,
// with a blank import, so each addon registers itself.
package dashboard

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/panel"
)

// State is everything the dashboard shows. Poll fills it in; addons only read
// it. Here it is synthetic data so the demo runs anywhere.
type State struct {
	At       time.Time
	CPU      float64 // percent
	Mem      float64 // percent
	Disk     float64 // percent
	NetIn    float64 // bytes per second
	NetOut   float64 // bytes per second
	Uptime   time.Duration
	Requests int
}

// Registry is the application's single registry. Addons register with it from
// init(); call its methods directly so errors name the addon's own file.
var Registry = dashygotui.NewRegistry[State]()

// Logs collects the application's log output. main diverts the standard logger
// into it before the dashboard starts, and the Logs addon shows it.
var Logs = logs.NewBuffer(2000)

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
)

// Config returns the dashboard's configuration. st keeps the user's settings.
func Config(st dashygotui.Store) dashygotui.Config[State] {
	return dashygotui.Config[State]{
		Title:    "Demo",
		Registry: Registry,
		Store:    st,
		Pages: []dashygotui.Page[State]{
			{ID: "monitor", Title: "Monitor", Layout: dashygotui.LayoutAuto},
			{ID: "control", Title: "Control", ShortTitle: "Ctl"},
			{ID: "theme", Title: "Theme"},
		},
		Sections: []dashygotui.SectionSpec[State]{
			{ID: "actions", Page: "control", Order: 10, Title: "Quick Actions", Subtitle: "things to do"},
			{ID: "settings", Page: "control", Order: 20, Title: "Settings", Subtitle: "saved between runs"},
			{ID: "appearance", Page: "theme", Title: "Appearance", Subtitle: "colours and spacing"},
		},
		Poll:     poll(time.Now()),
		Interval: time.Second,
		Header: func(c *Ctx) []panel.Segment {
			th := c.Theme()
			return []panel.Segment{
				{Text: th.White.Render("Demo v0.1"), Prio: 100},
				{Text: th.Cyan.Render("example.local"), Prio: 60},
				{Text: th.Green.Render(c.Now().Format("15:04:05")), Prio: 80},
			}
		},
	}
}

// Run shows the dashboard and returns why it ended.
func Run(ctx context.Context, st dashygotui.Store) (dashygotui.ExitInfo, error) {
	d, err := dashygotui.New(Config(st))
	if err != nil {
		return dashygotui.ExitInfo{}, fmt.Errorf("building the dashboard: %w", err)
	}
	return d.Run(ctx)
}

// poll returns a function that produces gently moving synthetic readings.
func poll(start time.Time) func(context.Context) (State, error) {
	requests := 0
	return func(context.Context) (State, error) {
		now := time.Now()
		t := now.Sub(start).Seconds()
		requests += 3 + int(2*math.Sin(t/3)+2)
		return State{
			At:       now,
			CPU:      clamp(38+26*math.Sin(t/7)+8*math.Sin(t*1.7), 0, 100),
			Mem:      clamp(54+6*math.Sin(t/19), 0, 100),
			Disk:     clamp(71+0.2*math.Sin(t/60), 0, 100),
			NetIn:    math.Max(0, 420_000+380_000*math.Sin(t/5)+90_000*math.Sin(t*2.3)),
			NetOut:   math.Max(0, 120_000+100_000*math.Sin(t/6+1)),
			Uptime:   now.Sub(start),
			Requests: requests,
		}, nil
	}
}

func clamp(v, lo, hi float64) float64 { return math.Min(math.Max(v, lo), hi) }
