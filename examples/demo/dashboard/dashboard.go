// Package dashboard is the frame of the demo application's dashboard: the
// state it shows, the poll that fills it, the header, and the short type names
// the addon files use. What the dashboard contains, and where, is the map in
// addons/addons.go; main passes it in. This package never imports addons.
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

// Logs collects the application's log output. main diverts the standard logger
// into it before the dashboard starts, and the Logs addon shows it.
var Logs = logs.NewBuffer(2000)

// Short names for the generic types, so the addon files and the map stay terse.
type (
	// Map is what the dashboard contains and where: addons/addons.go.
	Map = dashygotui.Map[State]
	// Page is a tab, or a sub-page when it is a row of a panel.
	Page = dashygotui.Page[State]
	// Panel is one panel of a page: a widget, or a list of rows.
	Panel = dashygotui.Panel[State]
	// Addon is anything a panel's rows take.
	Addon = dashygotui.Addon[State]
	// Ctx is the handle every addon callback receives.
	Ctx = dashygotui.Context[State]
	// Option is a numbered row.
	Option = dashygotui.Option[State]
	// Widget is a read-only panel.
	Widget = dashygotui.Widget[State]
	// View is a full-screen view.
	View = dashygotui.View[State]
	// Modal is a pop-up.
	Modal = dashygotui.Modal[State]
	// Shortcut is a key that works on every page, or everywhere.
	Shortcut = dashygotui.Shortcut[State]
	// Hook reacts to ticks, messages and task results.
	Hook = dashygotui.Hook[State]
)

// Config returns the dashboard's configuration. st keeps the user's settings;
// mp is what the dashboard contains, addons.Map() in the application.
func Config(st dashygotui.Store, mp Map) dashygotui.Config[State] {
	return dashygotui.Config[State]{
		Title:    "Demo",
		Map:      mp,
		Store:    st,
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
func Run(ctx context.Context, st dashygotui.Store, mp Map) (dashygotui.ExitInfo, error) {
	d, err := dashygotui.New(Config(st, mp))
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
