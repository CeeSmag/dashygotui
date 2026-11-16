package dashygotui

import (
	"cmp"
	"errors"
	"fmt"
	"strings"
)

// Map is what a dashboard contains and where: its pages, the panels on each
// page, the rows in each panel, and the keys and hooks that belong to the whole
// dashboard. An application writes it in one file (addons/addons.go), so the
// map is the only place anything is placed; an addon says only what it does.
// Give it to [New] in [Config.Map].
type Map[S any] struct {
	// Pages are the dashboard's tabs, in tab order. At least one is required.
	Pages []Page[S]
	// Shortcuts are the keys that work on every page, or everywhere.
	Shortcuts []Shortcut[S]
	// Hooks react to ticks, messages and task results for the whole dashboard.
	Hooks []Hook[S]
	// Views are views with no row of their own: opened by a shortcut, by an
	// option's Opens, by [Context.OpenView] or by [Config.StartView]. Each needs
	// an ID. A view placed as a row of a panel is not listed here.
	Views []View[S]
}

// Panel is one bordered panel of a page: one widget, or a list of numbered
// rows. Set Widget or Items, not both.
type Panel[S any] struct {
	// Title is shown in the panel's top border. Required for a list of rows; for
	// a widget it replaces the widget's own Title.
	Title string
	// Subtitle is shown in the bottom border. For a widget it replaces the
	// widget's own Subtitle. Optional.
	Subtitle string
	// Badge is shown in the top border, already styled. A list of rows shows a
	// count, "[3 options]", by default; for a widget it replaces the widget's own
	// Badge. Optional.
	Badge func(*Context[S]) string
	// Noun is the word the default count badge counts, "option" by default.
	Noun string
	// Number is the number of the first row; the rows after it count up from
	// there. Without it the panel's rows continue from the rows before them, or
	// start at 1. Optional.
	Number int
	// Widget is the panel's widget. Optional.
	Widget Widget[S]
	// Items are the panel's rows, top to bottom: an [Option], a [View] (the row
	// opens it), a [Page] (the row opens it as a sub-page), or a ready-made
	// setting such as one from package stdaddons.
	Items []Addon[S]
}

// Hotkey gives a row of the map a key that runs it without selecting it: from
// any tab page for a row of a tab, and only inside the window for a row of a
// window. The row is shown as "Label (key)". Like a row's number, its hotkey is
// the map's to give, never the feature's:
//
//	dashboard.Hotkey{Key: "d", Hint: "Diagnostics", Item: diagPage()}
//
// A hotkey may not be a navigation key, a digit, a shortcut or another row's
// hotkey in the same place.
type Hotkey[S any] struct {
	// Key is the key, as Bubble Tea names it: "d", "ctrl+o". Required.
	Key string
	// Display is how the key is written after the label and in the footer, such
	// as "Ctrl+O". It defaults to Key.
	Display string
	// Tone colours the key after the label and in the footer.
	Tone Tone
	// Hint, when set, also lists the key in the footer with this wording.
	Hint string
	// Item is the row: an option, a view, a window or a ready-made setting that
	// makes exactly one row. Required.
	Item Addon[S]
}

// Addons implements [Addon].
func (h Hotkey[S]) Addons() []Addon[S] { return []Addon[S]{h} }

// isWidget reports whether the panel holds a widget, as opposed to rows.
func (p Panel[S]) isWidget() bool {
	w := p.Widget
	return w.Render != nil || w.Title != "" || w.Tick != nil || w.Badge != nil
}

// Addons implements [Addon], so a page can be a row of a panel: a sub-page.
func (p Page[S]) Addons() []Addon[S] { return []Addon[S]{p} }

// mapped is what a Map compiles to: the pages the engine draws and what the
// addons add up to, as the v0.4 registry produced them.
type mapped[S any] struct {
	pages []pageDef[S]
	reg   *registry[S]
}

// mapper compiles a Map, recording every mistake with the place in the map.
type mapper[S any] struct {
	reg     *registry[S]
	hot     *Hotkey[S] // a hotkey waiting for the row it was given to
	pages   []pageDef[S]
	subRows []placedRow[S] // the rows of sub-pages, for checking what they open
}

// placedRow is a row with the place it was put.
type placedRow[S any] struct {
	path string
	opt  Option[S]
}

// scope is a numbering and hotkey space: the tab pages share one, and every
// sub-page has its own.
type scope[S any] struct {
	next    int               // the number the next row gets
	numbers map[int]string    // number -> place
	hotkeys map[string]string // hotkey -> place
	global  bool              // the tab pages' scope: hotkeys work from every page
}

func newScope[S any](global bool) *scope[S] {
	return &scope[S]{next: 1, numbers: map[int]string{}, hotkeys: map[string]string{}, global: global}
}

// compileMap checks mp and turns it into pages and registered addons.
func compileMap[S any](mp Map[S]) mapped[S] {
	c := &mapper[S]{reg: newRegistry[S]()}
	for i, s := range mp.Shortcuts {
		s.register(c.reg, fmt.Sprintf("Map.Shortcuts[%d]", i))
	}
	for i, h := range mp.Hooks {
		h.register(c.reg, fmt.Sprintf("Map.Hooks[%d]", i))
	}
	for i, v := range mp.Views {
		site := fmt.Sprintf("Map.Views[%d]", i)
		if strings.TrimSpace(v.ID) == "" {
			c.reg.fail(site, "view %q has no ID: a view with no row of its own is opened by its ID", v.Title)
			continue
		}
		c.view(v, site)
	}
	if len(mp.Pages) == 0 {
		c.reg.errs = append(c.reg.errs, errors.New("the Map has no Pages"))
	}
	tabs := newScope[S](true)
	ids := map[string]bool{}
	for i, p := range mp.Pages {
		path := cmp.Or(p.Title, fmt.Sprintf("Map.Pages[%d]", i))
		if p.ID != "" {
			if ids[p.ID] {
				c.reg.fail(path, "duplicate page ID %q", p.ID)
			}
			ids[p.ID] = true
		}
		if !rowZero(p.Row) {
			c.reg.fail(path, "a tab has no row: Row is for a page placed as a row of a panel")
		}
		c.pages = append(c.pages, c.page(p, path, tabs))
	}
	for _, r := range c.subRows {
		if r.opt.Opens != "" && c.reg.viewIDs[r.opt.Opens] == "" {
			c.reg.fail(r.path, "row %q opens unknown view %q", r.opt.Label, r.opt.Opens)
		}
	}
	return mapped[S]{pages: c.pages, reg: c.reg}
}

// page checks p and assembles its panels. path names it in errors; sc is the
// numbering and hotkey space its rows belong to.
func (c *mapper[S]) page(p Page[S], path string, sc *scope[S]) pageDef[S] {
	if strings.TrimSpace(p.Title) == "" {
		c.reg.fail(path, "page has no Title")
	}
	for _, err := range fallbackProblems(p) {
		c.reg.fail(path, "%s", err)
	}
	def := pageDef[S]{page: p}
	for i, pn := range p.Panels {
		ppath := path + " > " + panelName(pn, i)
		if pn.isWidget() {
			if w, ok := c.widget(pn, ppath); ok {
				def.panels = append(def.panels, panelDef[S]{widget: &w})
			}
			continue
		}
		if strings.TrimSpace(pn.Title) == "" {
			c.reg.fail(ppath, "panel has neither a Widget nor a Title for its rows")
		}
		if pn.Number < 0 {
			c.reg.fail(ppath, "panel Number %d is negative", pn.Number)
		} else if pn.Number > 0 {
			sc.next = pn.Number
		}
		var rows []Option[S]
		for j, it := range pn.Items {
			rows = append(rows, c.item(it, fmt.Sprintf("%s > row %d", ppath, j+1), path+" > "+panelName(pn, i), sc, 0)...)
		}
		def.panels = append(def.panels, panelDef[S]{opts: rows,
			sec: &section[S]{title: pn.Title, subtitle: pn.Subtitle, noun: pn.Noun, badge: pn.Badge}})
	}
	return def
}

// panelName names a panel in errors: its title, or its position.
func panelName[S any](pn Panel[S], i int) string {
	return cmp.Or(pn.Title, pn.Widget.Title, fmt.Sprintf("panel %d", i+1))
}

// widget checks the widget of a panel and applies the panel's title, subtitle
// and badge to it.
func (c *mapper[S]) widget(pn Panel[S], path string) (Widget[S], bool) {
	w := pn.Widget
	ok := true
	if len(pn.Items) > 0 {
		c.reg.fail(path, "panel has both a Widget and Items: a widget is a panel of its own")
		ok = false
	}
	if pn.Number != 0 || pn.Noun != "" {
		c.reg.fail(path, "a widget panel has no rows to number or count: Number and Noun are for Items")
		ok = false
	}
	w.Title = cmp.Or(pn.Title, w.Title)
	w.Subtitle = cmp.Or(pn.Subtitle, w.Subtitle)
	if pn.Badge != nil {
		w.Badge = pn.Badge
	}
	if strings.TrimSpace(w.Title) == "" {
		c.reg.fail(path, "widget has no Title")
		ok = false
	}
	if w.Render == nil {
		c.reg.fail(path, "widget %q has no Render", w.Title)
		ok = false
	}
	if !ok {
		return w, false
	}
	if w.Tick != nil {
		c.reg.onTick = append(c.reg.onTick, w.Tick)
	}
	return w, true
}

// item turns one entry of a panel's Items into its rows, numbered from sc.
// parent names the panel, for the path of a sub-page.
func (c *mapper[S]) item(it Addon[S], path, parent string, sc *scope[S], depth int) []Option[S] {
	switch v := it.(type) {
	case nil:
		c.reg.fail(path, "nil row")
	case Option[S]:
		return c.row(v, path, sc)
	case Hotkey[S]:
		if c.hot != nil {
			c.reg.fail(path, "hotkey %q is given to another hotkey", v.Key)
			return nil
		}
		if strings.TrimSpace(v.Key) == "" {
			c.reg.fail(path, "hotkey has no Key")
		}
		c.hot = &v
		rows := c.item(v.Item, path, parent, sc, depth)
		if c.hot != nil {
			c.hot = nil
			c.reg.fail(path, "hotkey %q has no row to run", v.Key)
			return rows
		}
		if len(rows) != 1 {
			c.reg.fail(path, "hotkey %q is given to %d rows; give it one", v.Key, len(rows))
		}
		return rows
	case View[S]:
		if v.ID == "" {
			v.ID = path
		}
		row := v.Row
		v.Row = Option[S]{}
		if !c.view(v, path) {
			return nil
		}
		return c.opener(row, v.Title, v.ID, path, sc)
	case Page[S]:
		v.Title = cmp.Or(v.Title, v.Row.Label)
		spath := parent + " > " + cmp.Or(v.Title, "page")
		hot := c.hot // the window's own rows are not the one the hotkey is for
		c.hot = nil
		def := c.page(v, spath, newScope[S](false))
		c.hot = hot
		id := cmp.Or(v.ID, spath)
		if !c.view(View[S]{ID: id, Title: cmp.Or(v.Title, "page"), Badge: v.Badge, Session: v.Session, sub: &def}, path) {
			return nil
		}
		return c.opener(v.Row, v.Title, id, path, sc)
	case Widget[S]:
		c.reg.fail(path, "widget %q is a panel of its own: {Widget: ...}, not a row", v.Title)
	case Shortcut[S]:
		c.reg.fail(path, "shortcut %q goes in Map.Shortcuts, not a panel", v.Key)
	case Hook[S]:
		c.reg.fail(path, "a hook goes in Map.Hooks, not a panel")
	case Invalid[S]:
		c.reg.fail(path, "%s", v.Reason)
	default:
		// A ready-made addon: its options are rows here, a view with a Row is a
		// row that opens it, and any other view, shortcut or hook it brings along
		// is registered without a row of its own.
		if depth >= maxAddonDepth {
			c.reg.fail(path, "addon %T expands into addons more than %d levels deep", it, maxAddonDepth)
			return nil
		}
		var rows []Option[S]
		for _, sub := range it.Addons() {
			switch x := sub.(type) {
			case View[S]:
				if !rowZero(x.Row) {
					rows = append(rows, c.item(x, path, parent, sc, depth+1)...)
					continue
				}
				c.view(x, path)
			case Shortcut[S]:
				x.register(c.reg, path)
			case Hook[S]:
				x.register(c.reg, path)
			default:
				rows = append(rows, c.item(sub, path, parent, sc, depth+1)...)
			}
		}
		return rows
	}
	return nil
}

// opener checks and numbers r, the row that opens view or sub-page id: its
// label defaults to title, and its action is opening it.
func (c *mapper[S]) opener(r Option[S], title, id, path string, sc *scope[S]) []Option[S] {
	if r.actions() > 0 {
		c.reg.fail(path, "the row %q opens %q: it sets no Run, Do, Opens or Modal of its own", cmp.Or(r.Label, title), title)
		return nil
	}
	r.Label = cmp.Or(r.Label, title)
	r.Opens = id
	return c.row(r, path, sc)
}

// rowZero reports whether r says nothing about how a row looks or acts.
func rowZero[S any](r Option[S]) bool {
	return r.Label == "" && r.Badge == nil && r.Note == "" && r.actions() == 0
}

// view registers a view and reports whether it was valid.
func (c *mapper[S]) view(v View[S], path string) bool {
	if !rowZero(v.Row) {
		c.reg.fail(path, "view %q has a Row but is not placed as one: place the view in a panel's Items instead", v.ID)
		return false
	}
	before := len(c.reg.errs)
	v.register(c.reg, path)
	return len(c.reg.errs) == before
}

// row checks one option and numbers it.
func (c *mapper[S]) row(o Option[S], path string, sc *scope[S]) []Option[S] {
	ok := true
	fail := func(format string, args ...any) {
		c.reg.fail(path, format, args...)
		ok = false
	}
	if strings.TrimSpace(o.Label) == "" {
		fail("option has no Label")
	}
	if n := o.actions(); n != 1 {
		fail("option %q needs exactly one of Run, Do, Opens and Modal, has %d", o.Label, n)
	}
	if c.hot != nil {
		o.hot = *c.hot
		o.hot.Item = nil
		c.hot = nil
	}
	if hk := o.hot.Key; hk != "" {
		switch {
		case sc.hotkeys[hk] != "":
			fail("option %q reuses hotkey %q (first used at %s)", o.Label, hk, sc.hotkeys[hk])
		case sc.global && c.reg.shortcutKey[hk] != "":
			fail("option %q hotkey %q is already a shortcut (%s)", o.Label, hk, c.reg.shortcutKey[hk])
		default:
			sc.hotkeys[hk] = path
			if sc.global {
				c.reg.hotkeys[hk] = path
			}
		}
	}
	n := sc.next
	sc.next++
	if first := sc.numbers[n]; first != "" {
		fail("option %q gets number %d, which %s already has: raise the panel's Number", o.Label, n, first)
	}
	if !ok {
		return nil
	}
	sc.numbers[n] = fmt.Sprintf("%s (%q)", path, o.Label)
	o.number = n
	if sc.global {
		c.reg.options = append(c.reg.options, o)
	} else {
		c.subRows = append(c.subRows, placedRow[S]{path: path, opt: o})
	}
	return []Option[S]{o}
}

// fallbackProblems checks a page's Fallback settings.
func fallbackProblems[S any](p Page[S]) []string {
	below := p.FallbackBelow.Cols > 0 || p.FallbackBelow.Rows > 0
	switch {
	case p.Fallback != nil && !below && p.FallbackWhen == nil:
		return []string{"page has a Fallback but no FallbackBelow size or FallbackWhen"}
	case p.Fallback == nil && (below || p.FallbackWhen != nil):
		return []string{"page sets FallbackBelow or FallbackWhen but has no Fallback"}
	case below && p.FallbackWhen != nil:
		return []string{"page sets both FallbackBelow and FallbackWhen; use one"}
	}
	return nil
}
