package dashygotui

import (
	"cmp"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Registry collects the options, widgets, views, shortcuts and hooks that
// addons register. An application creates exactly one for its state type S,
// declares it as a package-level variable in its dashboard file, and addons
// register with it from init().
//
// A registry is safe for concurrent use. Registration never panics: a mistake
// is recorded together with the "file.go:line" of the offending call and
// reported by [Registry.Validate] and [New], before the UI starts.
//
// Call the registration methods directly, e.g. dashboard.Registry.Add(...).
// The recorded site is the immediate caller, so a wrapper function would hide
// the real location.
type Registry[S any] struct {
	mu     sync.Mutex
	frozen bool

	options   []Option[S]
	widgets   []Widget[S]
	views     []View[S]
	shortcuts []Shortcut[S]
	onTick    []func(*Context[S], time.Time) tea.Cmd
	onResult  []func(*Context[S], TaskResult) (bool, tea.Cmd)
	onMsg     []func(*Context[S], tea.Msg) tea.Cmd

	// Where each unique thing was first registered, for duplicate messages.
	optionKeys  map[string]string
	optionNums  map[int]string
	widgetKeys  map[string]string
	viewIDs     map[string]string
	shortcutKey map[string]string
	hotkeys     map[string]string

	errs []error
}

// NewRegistry returns an empty registry for state type S.
func NewRegistry[S any]() *Registry[S] {
	return &Registry[S]{
		optionKeys:  map[string]string{},
		optionNums:  map[int]string{},
		widgetKeys:  map[string]string{},
		viewIDs:     map[string]string{},
		shortcutKey: map[string]string{},
		hotkeys:     map[string]string{},
	}
}

// callerSite returns "file.go:line" of the code that called the Registry
// method calling callerSite, which is the addon's registration call.
func callerSite() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// begin locks the registry and reports whether registration is still allowed.
// The caller must unlock.
func (r *Registry[S]) begin(site, what string) bool {
	r.mu.Lock()
	if r.frozen {
		r.errs = append(r.errs, &RegistrationError{Site: site, Msg: what + " registered after the dashboard was created"})
		return false
	}
	return true
}

func (r *Registry[S]) fail(site, format string, args ...any) {
	r.errs = append(r.errs, &RegistrationError{Site: site, Msg: fmt.Sprintf(format, args...)})
}

// Option registers a numbered row for a section.
func (r *Registry[S]) Option(o Option[S]) { o.register(r, callerSite()) }

// Add registers any number of options, shortcuts, views and widgets in one
// call, so an addon that adds several things reads as one declaration:
//
//	dashboard.Registry.Add(dashboard.Option{...}, dashboard.View{...})
//
// It does the same as calling [Registry.Option], [Registry.Shortcut],
// [Registry.View] and [Registry.Widget] for each, and records the call site of
// Add itself for error messages.
func (r *Registry[S]) Add(items ...Addon[S]) {
	site := callerSite()
	for _, it := range items {
		r.registerAddon(it, site, 0)
	}
}

func (o Option[S]) register(r *Registry[S], site string) {
	if !r.begin(site, "option") {
		r.mu.Unlock()
		return
	}
	defer r.mu.Unlock()
	r.addOption(site, o)
}

// addOption validates and stores an option. The registry must be locked and
// not frozen.
func (r *Registry[S]) addOption(site string, o Option[S]) {
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

// Widget registers a read-only panel for a page.
func (r *Registry[S]) Widget(w Widget[S]) { w.register(r, callerSite()) }

func (w Widget[S]) register(r *Registry[S], site string) {
	if !r.begin(site, "widget") {
		r.mu.Unlock()
		return
	}
	defer r.mu.Unlock()

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

// View registers a full-screen view that [Context.OpenView] can open.
func (r *Registry[S]) View(v View[S]) { v.register(r, callerSite()) }

func (v View[S]) register(r *Registry[S], site string) {
	if !r.begin(site, "view") {
		r.mu.Unlock()
		return
	}
	defer r.mu.Unlock()

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

// Shortcut registers a global key binding for the dashboard pages.
func (r *Registry[S]) Shortcut(s Shortcut[S]) { s.register(r, callerSite()) }

func (s Shortcut[S]) register(r *Registry[S], site string) {
	if !r.begin(site, "shortcut") {
		r.mu.Unlock()
		return
	}
	defer r.mu.Unlock()

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

// OnTick registers a hook called on every refresh tick, for addons that sample
// data over time. The hook runs on the update loop and must not block.
func (r *Registry[S]) OnTick(fn func(*Context[S], time.Time) tea.Cmd) {
	site := callerSite()
	if !r.begin(site, "tick hook") {
		r.mu.Unlock()
		return
	}
	defer r.mu.Unlock()
	if fn == nil {
		r.fail(site, "tick hook is nil")
		return
	}
	r.onTick = append(r.onTick, fn)
}

// OnResult registers a hook that sees the result of every background task.
// Return handled == true to stop other hooks and the default footer message.
func (r *Registry[S]) OnResult(fn func(*Context[S], TaskResult) (handled bool, cmd tea.Cmd)) {
	site := callerSite()
	if !r.begin(site, "result hook") {
		r.mu.Unlock()
		return
	}
	defer r.mu.Unlock()
	if fn == nil {
		r.fail(site, "result hook is nil")
		return
	}
	r.onResult = append(r.onResult, fn)
}

// OnMsg registers a hook for messages the dashboard does not recognise, so an
// addon can handle the messages of its own commands. Return a non-nil command
// to claim the message and stop later hooks.
func (r *Registry[S]) OnMsg(fn func(*Context[S], tea.Msg) tea.Cmd) {
	site := callerSite()
	if !r.begin(site, "message hook") {
		r.mu.Unlock()
		return
	}
	defer r.mu.Unlock()
	if fn == nil {
		r.fail(site, "message hook is nil")
		return
	}
	r.onMsg = append(r.onMsg, fn)
}

// Validate returns every registration mistake recorded so far, joined, or nil.
func (r *Registry[S]) Validate() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return errors.Join(r.errs...)
}

// registered is an immutable copy of what was registered.
type registered[S any] struct {
	options   []Option[S]
	widgets   []Widget[S]
	views     []View[S]
	shortcuts []Shortcut[S]
	onTick    []func(*Context[S], time.Time) tea.Cmd
	onResult  []func(*Context[S], TaskResult) (bool, tea.Cmd)
	onMsg     []func(*Context[S], tea.Msg) tea.Cmd
}

// freeze stops further registration and returns a copy of the contents.
func (r *Registry[S]) freeze() registered[S] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frozen = true
	return registered[S]{
		options:   append([]Option[S](nil), r.options...),
		widgets:   append([]Widget[S](nil), r.widgets...),
		views:     append([]View[S](nil), r.views...),
		shortcuts: append([]Shortcut[S](nil), r.shortcuts...),
		onTick:    append([]func(*Context[S], time.Time) tea.Cmd(nil), r.onTick...),
		onResult:  append([]func(*Context[S], TaskResult) (bool, tea.Cmd)(nil), r.onResult...),
		onMsg:     append([]func(*Context[S], tea.Msg) tea.Cmd(nil), r.onMsg...),
	}
}
