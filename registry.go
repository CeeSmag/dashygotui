package dashygotui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// registry collects and checks the views, shortcuts and hooks of one dashboard
// while [New] compiles its map, and the rows of its tab pages. Each is recorded
// with its place in the map, so a mistake is reported with that place, and every
// mistake is kept rather than the first: [New] reports them all at once.
type registry[S any] struct {
	options   []Option[S] // the rows of the tab pages, whose hotkeys work on every page
	views     []View[S]
	shortcuts []Shortcut[S]
	onTick    []func(*Context[S], time.Time) tea.Cmd
	onResult  []func(*Context[S], TaskResult) (bool, tea.Cmd)
	onMsg     []func(*Context[S], tea.Msg) tea.Cmd

	// Where each unique thing was first placed, for duplicate messages.
	viewIDs     map[string]string
	shortcutKey map[string]string
	hotkeys     map[string]string // the hotkeys of the tab pages' rows

	errs []error
}

// newRegistry returns an empty registry.
func newRegistry[S any]() *registry[S] {
	return &registry[S]{viewIDs: map[string]string{}, shortcutKey: map[string]string{}, hotkeys: map[string]string{}}
}

func (r *registry[S]) fail(site, format string, args ...any) {
	r.errs = append(r.errs, &AddonError{Site: site, Msg: fmt.Sprintf(format, args...)})
}

func (v View[S]) register(r *registry[S], site string) {
	ok := true
	switch {
	case strings.TrimSpace(v.ID) == "":
		r.fail(site, "view has no ID")
		ok = false
	case r.viewIDs[v.ID] != "":
		r.fail(site, "duplicate view ID %q (first registered at %s)", v.ID, r.viewIDs[v.ID])
		ok = false
	}
	if strings.TrimSpace(v.Title) == "" {
		r.fail(site, "view %q has no Title", v.ID)
		ok = false
	}
	if v.Body == nil && v.sub == nil {
		r.fail(site, "view %q has no Body to draw", v.ID)
		ok = false
	}
	if (v.Follow || v.Framed) && v.Body == nil {
		r.fail(site, "view %q sets Follow or Framed but has no Body to frame", v.ID)
		ok = false
	}
	if v.Follow {
		v.Scroll, v.Tail = true, true
	}
	if (v.Follow || v.Framed) && v.Panel == nil {
		v.Panel = &ViewPanel[S]{}
	}
	if v.Scroll && v.Body == nil {
		r.fail(site, "view %q sets Scroll but has no Body to scroll", v.ID)
		ok = false
	}
	if v.Panel != nil && v.Body == nil {
		r.fail(site, "view %q sets Panel but has no Body to frame", v.ID)
		ok = false
	}
	if v.Appended != nil && !v.Tail {
		r.fail(site, "view %q sets Appended but not Tail", v.ID)
		ok = false
	}
	if v.Tail && !v.Scroll {
		r.fail(site, "view %q sets Tail but not Scroll", v.ID)
		ok = false
	}
	if !ok {
		return
	}
	r.viewIDs[v.ID] = site
	r.views = append(r.views, v)
}

func (s Shortcut[S]) register(r *registry[S], site string) {
	ok := true
	switch {
	case s.Key == "":
		r.fail(site, "shortcut has no Key")
		ok = false
	case r.shortcutKey[s.Key] != "":
		r.fail(site, "duplicate shortcut key %q (first registered at %s)", s.Key, r.shortcutKey[s.Key])
		ok = false
	}
	if r.hotkeys[s.Key] != "" {
		r.fail(site, "shortcut %q is already a row's hotkey (%s)", s.Key, r.hotkeys[s.Key])
		ok = false
	}
	if n := s.actions(); n != 1 {
		r.fail(site, "shortcut %q needs exactly one of Run, Do, Opens and Modal, has %d", s.Key, n)
		ok = false
	}
	if !ok {
		return
	}
	r.shortcutKey[s.Key] = site
	r.shortcuts = append(r.shortcuts, s)
}

func (h Hook[S]) register(r *registry[S], site string) {
	if h.Tick == nil && h.Msg == nil && h.Result == nil {
		r.fail(site, "hook sets none of Tick, Msg and Result")
		return
	}
	if h.Tick != nil {
		r.onTick = append(r.onTick, h.Tick)
	}
	if h.Msg != nil {
		r.onMsg = append(r.onMsg, h.Msg)
	}
	if h.Result != nil {
		r.onResult = append(r.onResult, h.Result)
	}
}

// err returns every mistake recorded, joined, or nil.
func (r *registry[S]) err() error { return errors.Join(r.errs...) }

// registered is what the map added up to.
type registered[S any] struct {
	options   []Option[S]
	views     []View[S]
	shortcuts []Shortcut[S]
	onTick    []func(*Context[S], time.Time) tea.Cmd
	onResult  []func(*Context[S], TaskResult) (bool, tea.Cmd)
	onMsg     []func(*Context[S], tea.Msg) tea.Cmd
}

// contents returns what was collected.
func (r *registry[S]) contents() registered[S] {
	return registered[S]{
		options: r.options, views: r.views, shortcuts: r.shortcuts,
		onTick: r.onTick, onResult: r.onResult, onMsg: r.onMsg,
	}
}
