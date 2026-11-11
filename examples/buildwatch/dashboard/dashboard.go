// Package dashboard is the foundation of the buildwatch dashboard: the state, the
// pages and sections, the one Registry. Addons add everything else.
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

// Registry is the application's single registry.
var Registry = dashygotui.NewRegistry[State]()

// Short names for the generic types.
type (
	// Ctx is the handle every addon callback receives.
	Ctx = dashygotui.Context[State]
	// Option is a numbered row in a section.
	Option = dashygotui.Option[State]
	// Widget is a read-only panel on a page.
	Widget = dashygotui.Widget[State]
	// View is a full-screen view.
	View = dashygotui.View[State]
	// Entry is the menu option that opens a view.
	Entry = dashygotui.Entry[State]
	// Modal is a pop-up.
	Modal = dashygotui.Modal[State]
	// Shortcut is a key binding that is not a menu row.
	Shortcut = dashygotui.Shortcut[State]
)

// Config returns the dashboard's configuration. A nil st keeps settings in memory.
func Config(st dashygotui.Store) dashygotui.Config[State] {
	if st == nil {
		st = store.NewMemory()
	}
	return dashygotui.Config[State]{
		Title:    "Buildwatch",
		Registry: Registry,
		Store:    st,
		Pages: []dashygotui.Page[State]{
			{ID: "builds", Title: "Builds"},
			{ID: "settings", Title: "Settings"},
		},
		Sections: []dashygotui.SectionSpec[State]{
			{ID: "actions", Page: "builds", Title: "Actions"},
			{ID: "prefs", Page: "settings", Title: "Preferences", Subtitle: "saved between runs"},
		},
		Poll:     poll(time.Now()),
		Interval: 2 * time.Second,
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
