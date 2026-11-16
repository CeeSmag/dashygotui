// Package dashboard is the frame of the buildwatch dashboard: the state, the poll
// and the short type names. What it contains, and where, is the map in
// addons/addons.go, which main passes in.
package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/store"
)

// Build is one run of a pipeline.
type Build struct {
	Pipeline string
	Number   int
	Status   string // "running", "passed" or "failed"
	Took     time.Duration
}

// State is what the dashboard shows. Poll fills it in.
type State struct {
	Builds []Build
}

// Failed counts the failed builds.
func (s State) Failed() int {
	n := 0
	for _, b := range s.Builds {
		if b.Status == "failed" {
			n++
		}
	}
	return n
}

// Short names for the generic types.
type (
	// Map is what the dashboard contains and where: addons/addons.go.
	Map = dashygotui.Map[State]
	// Page is a tab, or a sub-page when it is a row of a panel.
	Page = dashygotui.Page[State]
	// Panel is one panel of a page: a widget, or a list of rows.
	Panel = dashygotui.Panel[State]
	// Ctx is the handle every addon callback receives.
	Ctx = dashygotui.Context[State]
	// Option is a numbered row.
	Option = dashygotui.Option[State]
	// Widget is a read-only panel.
	Widget = dashygotui.Widget[State]
	// View is a full-screen view.
	View = dashygotui.View[State]
	// Entry is how the row that opens a view or a sub-page looks.
	Entry = dashygotui.Entry[State]
	// Modal is a pop-up.
	Modal = dashygotui.Modal[State]
	// Shortcut is a key binding that is not a menu row.
	Shortcut = dashygotui.Shortcut[State]
	// Addon is anything a panel's rows take.
	Addon = dashygotui.Addon[State]
	// Hook reacts to ticks, messages and task results.
	Hook = dashygotui.Hook[State]
)

// Config returns the dashboard's configuration. A nil st keeps settings in
// memory; mp is what the dashboard contains, addons.Map() in the application.
func Config(st dashygotui.Store, mp Map) dashygotui.Config[State] {
	if st == nil {
		st = store.NewMemory()
	}
	return dashygotui.Config[State]{
		Title:    "Buildwatch",
		Map:      mp,
		Store:    st,
		Poll:     poll(time.Now()),
		Interval: 2 * time.Second,
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

// poll fakes a few pipelines whose latest builds come and go.
func poll(start time.Time) func(context.Context) (State, error) {
	return func(context.Context) (State, error) {
		n := int(time.Since(start) / (2 * time.Second))
		names := []string{"api", "web", "worker", "docs"}
		var s State
		for i, name := range names {
			status := "passed"
			switch (n + i) % 5 {
			case 0:
				status = "running"
			case 3:
				status = "failed"
			}
			s.Builds = append(s.Builds, Build{Pipeline: name, Number: 100 + n + i, Status: status,
				Took: time.Duration(40+i*17) * time.Second})
		}
		return s, nil
	}
}
