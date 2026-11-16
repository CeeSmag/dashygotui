package dashygotui

import (
	"cmp"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// registry collects and checks the addons of one dashboard while [New] builds
// it. Each addon is recorded with the place [At] says it is declared, so a
// mistake is reported with that place, and every mistake is kept rather than
// the first: [New] reports them all at once.
type registry[S any] struct {
	options   []Option[S]
	widgets   []Widget[S]
	views     []View[S]
	shortcuts []Shortcut[S]
	onTick    []func(*Context[S], time.Time) tea.Cmd
	onResult  []func(*Context[S], TaskResult) (bool, tea.Cmd)
	onMsg     []func(*Context[S], tea.Msg) tea.Cmd

	// Where each unique thing was first declared, for duplicate messages.
	optionKeys  map[string]string
	optionNums  map[int]string
	widgetKeys  map[string]string
	viewIDs     map[string]string
	shortcutKey map[string]string
	hotkeys     map[string]string

	errs []error
}

// collect checks items and returns what they add up to. An item without [At]
// is named by its index in the list.
func collect[S any](items []Addon[S]) *registry[S] {
	r := &registry[S]{
		optionKeys:  map[string]string{},
		optionNums:  map[int]string{},
		widgetKeys:  map[string]string{},
		viewIDs:     map[string]string{},
		shortcutKey: map[string]string{},
		hotkeys:     map[string]string{},
	}
	for i, it := range items {
		r.registerAddon(it, fmt.Sprintf("Config.Addons[%d]", i), 0)
	}
	return r
}

func (r *registry[S]) fail(site, format string, args ...any) {
	r.errs = append(r.errs, &AddonError{Site: site, Msg: fmt.Sprintf(format, args...)})
}

func (o Option[S]) register(r *registry[S], site string) {
	r.addOption(site, o)
}

// addOption validates and stores an option.
func (r *registry[S]) addOption(site string, o Option[S]) {
	ok := true
	switch {
	case strings.TrimSpace(o.Key) == "":
		r.fail(site, "option has no Key")
		ok = false
	case r.optionKeys[o.Key] != "":
		r.fail(site, "duplicate option key %q (first registered at %s)", o.Key, r.optionKeys[o.Key])
		ok = false
	}
	if strings.HasPrefix(o.Key, ReservedKeyPrefix) {
		r.fail(site, "option key %q starts with %q, which is reserved for the library's own settings", o.Key, ReservedKeyPrefix)
		ok = false
	}
	if strings.TrimSpace(o.Section) == "" {
		r.fail(site, "option %q has no Section", o.Key)
		ok = false
	}
	if strings.TrimSpace(o.Label) == "" {
		r.fail(site, "option %q has no Label", o.Key)
		ok = false
	}
	if n := o.actions(); n != 1 {
		r.fail(site, "option %q needs exactly one of Run, Do, Opens and Modal, has %d", o.Key, n)
		ok = false
	}
	switch {
	case o.Number <= 0:
		r.fail(site, "option %q has Number %d; numbers must be positive", o.Key, o.Number)
		ok = false
	case r.optionNums[o.Number] != "":
		r.fail(site, "option %q reuses number %d (first used at %s)", o.Key, o.Number, r.optionNums[o.Number])
		ok = false
	}
	if hk := o.Hotkey; hk != "" {
		switch {
		case r.hotkeys[hk] != "":
			r.fail(site, "option %q reuses hotkey %q (first used at %s)", o.Key, hk, r.hotkeys[hk])
			ok = false
		case r.shortcutKey[hk] != "":
			r.fail(site, "option %q hotkey %q is already a shortcut (registered at %s)", o.Key, hk, r.shortcutKey[hk])
			ok = false
		}
	}
	if !ok {
		return
	}
	r.optionKeys[o.Key] = site
	r.optionNums[o.Number] = site
	if o.Hotkey != "" {
		r.hotkeys[o.Hotkey] = site
	}
	r.options = append(r.options, o)
}

func (w Widget[S]) register(r *registry[S], site string) {
	ok := true
	switch {
	case strings.TrimSpace(w.Key) == "":
		r.fail(site, "widget has no Key")
		ok = false
	case r.widgetKeys[w.Key] != "":
		r.fail(site, "duplicate widget key %q (first registered at %s)", w.Key, r.widgetKeys[w.Key])
		ok = false
	}
	if strings.TrimSpace(w.Page) == "" {
		r.fail(site, "widget %q has no Page", w.Key)
		ok = false
	}
	if strings.TrimSpace(w.Title) == "" {
		r.fail(site, "widget %q has no Title", w.Key)
		ok = false
	}
	if w.Render == nil {
		r.fail(site, "widget %q has no Render", w.Key)
		ok = false
	}
	if !ok {
		return
	}
	r.widgetKeys[w.Key] = site
	r.widgets = append(r.widgets, w)
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
	if v.Options != nil && len(v.Items) > 0 {
		r.fail(site, "view %q sets both Options and Items; use one", v.ID)
		ok = false
	}
	for i, it := range v.Items {
		if strings.TrimSpace(it.Label) == "" {
			r.fail(site, "view %q item #%d has no Label", v.ID, i+1)
			ok = false
		}
		if n := it.actions(); n != 1 {
			r.fail(site, "view %q item %q needs exactly one of Run, Do, Opens and Modal, has %d", v.ID, it.Label, n)
			ok = false
		}
	}
	seen := map[int]string{}
	for i, it := range v.Items {
		n := it.Number
		if n == 0 {
			n = i + 1 // numbered by position
		}
		if first, dup := seen[n]; dup {
			r.fail(site, "view %q item %q has number %d, which %q already has (rows without a Number are numbered by position)", v.ID, it.Label, n, first)
			ok = false
		}
		seen[n] = it.Label
	}
	if v.Body == nil && !v.hasOptions() && v.Status == nil {
		r.fail(site, "view %q draws nothing: set Body, Status, Options or Items", v.ID)
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
	if e := v.Entry; e != nil {
		r.addOption(site, Option[S]{
			Key: "view." + v.ID, Section: e.Section, Number: e.Number, Order: e.Order,
			Label: cmp.Or(e.Label, v.Title), Hotkey: e.Hotkey, HotkeyDisplay: e.HotkeyDisplay,
			HotkeyTone: e.HotkeyTone, HotkeyHint: e.HotkeyHint, Badge: e.Badge, Note: e.Note,
			Opens: v.ID,
		})
	}
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
		r.fail(site, "shortcut %q is already an option hotkey (registered at %s)", s.Key, r.hotkeys[s.Key])
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

// registered is what the addons added up to.
type registered[S any] struct {
	options   []Option[S]
	widgets   []Widget[S]
	views     []View[S]
	shortcuts []Shortcut[S]
	onTick    []func(*Context[S], time.Time) tea.Cmd
	onResult  []func(*Context[S], TaskResult) (bool, tea.Cmd)
	onMsg     []func(*Context[S], tea.Msg) tea.Cmd
}

// contents returns what was collected.
func (r *registry[S]) contents() registered[S] {
	return registered[S]{
		options: r.options, widgets: r.widgets, views: r.views, shortcuts: r.shortcuts,
		onTick: r.onTick, onResult: r.onResult, onMsg: r.onMsg,
	}
}
