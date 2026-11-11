package logs

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// prefixWidth is the width of "15:04:05 LEVEL " before an entry's text.
const prefixWidth = len("15:04:05 ERROR ")

// Viewer shows a [Buffer] newest entry first, with scrolling. It holds the
// reader's place: once you scroll away from the top, new entries do not push
// the line you are reading, because the view is anchored to an entry rather than
// to a row. Home returns to the live top, where the newest entry is always
// shown.
//
// A Viewer is not safe for concurrent use; the dashboard drives it from its
// update loop.
//
// When the buffer holds more than one category, or the viewer was given some,
// a strip of tabs runs along the top: "All" and then one tab per category, the
// selected one in brackets and highlighted. The developer picks the key that
// moves between them with [WithCategoryKey]. Each category keeps its own
// scroll position, and a tab you are not on counts the entries that arrived
// since you left it.
type Viewer struct {
	buf      *Buffer
	declared []string          // categories given up front, in tab order
	catKey   string            // moves to the next category; upper case moves back
	cat      string            // the selected category; "" is all of them
	anchors  map[string]uint64 // per category: Seq of the entry at the top while scrolled; 0 means live
	read     map[string]uint64 // per category: newest Seq seen when it was left
	min      Level             // entries below this level are hidden
}

// ViewerOption configures a [Viewer].
type ViewerOption func(*Viewer)

// WithCategories lists categories whose tabs are always shown, in this order,
// ahead of any other category the buffer meets later.
func WithCategories(names ...string) ViewerOption {
	return func(v *Viewer) { v.declared = append(v.declared, names...) }
}

// WithCategoryKey sets the key that selects the next category tab, wrapping
// round after the last; the same letter in upper case (or with shift) selects
// the previous one. It takes priority over the scrolling keys. Without it the
// tabs are shown but cannot be switched.
func WithCategoryKey(key string) ViewerOption {
	return func(v *Viewer) { v.catKey = key }
}

// NewViewer returns a viewer of buf, live, showing every level and category.
func NewViewer(buf *Buffer, opts ...ViewerOption) *Viewer {
	v := &Viewer{buf: buf, anchors: map[string]uint64{}, read: map[string]uint64{}}
	for _, o := range opts {
		o(v)
	}
	return v
}

// anchor is Seq of the entry at the top of the selected category while
// scrolled; 0 when live.
func (v *Viewer) anchor() uint64 { return v.anchors[v.cat] }

// Live reports whether the view is at the top, following the newest entry.
func (v *Viewer) Live() bool { return v.anchor() == 0 }

// Category is the selected category; the empty string means all of them.
func (v *Viewer) Category() string { return v.cat }

// Categories lists the category tabs after "All": the declared ones, then any
// others the buffer has met, in order of appearance.
func (v *Viewer) Categories() []string {
	out := slices.Clone(v.declared)
	for _, c := range v.buf.Categories() {
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

// showTabs reports whether the tab strip is drawn: when categories were
// declared, a key was given, or the buffer holds any besides the default.
func (v *Viewer) showTabs() bool {
	if len(v.declared) > 0 || v.catKey != "" {
		return true
	}
	for _, c := range v.buf.Categories() {
		if c != DefaultCategory {
			return true
		}
	}
	return false
}

// SetCategory selects a category by name; "" selects all. An unknown name is
// ignored. It leaves the other categories' scroll positions alone.
func (v *Viewer) SetCategory(name string) {
	if name == v.cat || (name != "" && !slices.Contains(v.Categories(), name)) {
		return
	}
	// Leaving a category you were following means you read up to now.
	if v.cat != "" && v.Live() {
		v.read[v.cat] = v.newest(v.cat)
	}
	v.cat = name
}

// stepCategory selects the next (delta 1) or previous (delta -1) tab, wrapping.
func (v *Viewer) stepCategory(delta int) {
	tabs := append([]string{""}, v.Categories()...)
	i := slices.Index(tabs, v.cat)
	v.SetCategory(tabs[((i+delta)%len(tabs)+len(tabs))%len(tabs)])
}

// newest is the Seq of the newest entry in a category.
func (v *Viewer) newest(category string) uint64 {
	for _, e := range v.buf.Snapshot() { // newest first
		if e.Category == category {
			return e.Seq
		}
	}
	return 0
}

// MinLevel is the lowest level shown.
func (v *Viewer) MinLevel() Level { return v.min }

// entries is the visible entries of the selected category, newest first.
func (v *Viewer) entries() []Entry { return v.visible(v.buf.Snapshot(), v.cat) }

// visible filters a snapshot to one category ("" is all) and the level filter.
func (v *Viewer) visible(all []Entry, category string) []Entry {
	if v.min == Debug && category == "" {
		return all
	}
	out := all[:0:0]
	for _, e := range all {
		if e.Level >= v.min && (category == "" || e.Category == category) {
			out = append(out, e)
		}
	}
	return out
}

// top is the index, into es, of the entry at the top of the view.
func (v *Viewer) top(es []Entry) int {
	anchor := v.anchor()
	if anchor == 0 || len(es) == 0 {
		return 0
	}
	// es is newest first, so Seq decreases: find the first entry at or older
	// than the anchor. If the anchor was evicted that is simply the oldest kept.
	i := sort.Search(len(es), func(i int) bool { return es[i].Seq <= anchor })
	return min(i, len(es)-1)
}

// Newer reports how many visible entries are above the top of the view: zero
// when live.
func (v *Viewer) Newer() int { return v.top(v.entries()) }

// scrollTo moves the top of the view to index i of the visible entries.
func (v *Viewer) scrollTo(es []Entry, i int) {
	i = min(max(i, 0), max(len(es)-1, 0))
	if i == 0 || len(es) == 0 {
		delete(v.anchors, v.cat)
		return
	}
	v.anchors[v.cat] = es[i].Seq
}

// HandleKey scrolls on the named key and reports whether it used it. rows is
// the height of the view, used as the page size. The keys are up/k (newer),
// down/j (older), pgup, pgdown, home (back to live), end (oldest) and f, which
// cycles the minimum level shown, plus the category key if one was set. Pass
// the key name as the terminal reports it ("c", "C", "up", "pgdown"): the case
// of a letter tells the category key from its reverse.
func (v *Viewer) HandleKey(name string, rows int) bool {
	if v.catKey != "" {
		switch name {
		case v.catKey:
			v.stepCategory(1)
			return true
		case strings.ToUpper(v.catKey), "shift+" + v.catKey:
			v.stepCategory(-1)
			return true
		}
	}
	name = strings.ToLower(name)
	es := v.entries()
	cur := v.top(es)
	page := max(rows-1, 1)
	switch name {
	case "up", "k":
		v.scrollTo(es, cur-1)
	case "down", "j":
		v.scrollTo(es, cur+1)
	case "pgup":
		v.scrollTo(es, cur-page)
	case "pgdown":
		v.scrollTo(es, cur+page)
	case "home":
		delete(v.anchors, v.cat)
	case "end":
		v.scrollTo(es, len(es)-1)
	case "f":
		v.min = (v.min + 1) % (Error + 1)
		clear(v.anchors) // an old anchor may now be filtered out
	default:
		return false
	}
	return true
}

// Badge describes the view's position: live, or how many entries are newer than
// the top.
func (v *Viewer) Badge(th *theme.Theme) string {
	if v.Live() {
		return th.Green.Render(th.G.Bullet + " LIVE")
	}
	return th.Amber.Render(fmt.Sprintf("[%d newer]", v.Newer()))
}

// Rows renders the view as exactly rows lines of exactly cols cells, newest
// entry first starting at the top of the view. When the category tabs are shown
// they take the first row.
func (v *Viewer) Rows(th *theme.Theme, cols, rows int) []string {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	all := v.buf.Snapshot()
	es := v.visible(all, v.cat)
	out := make([]string, 0, rows)
	if v.showTabs() {
		out = append(out, v.tabs(th, cols, all))
	}
	head := len(out)
	switch {
	case len(es) == 0 && v.buf.Len() == 0:
		out = append(out, th.Muted.Render("no log entries yet"))
	case len(es) == 0 && v.cat != "":
		msg := "no entries in " + v.cat
		if v.min > Debug {
			msg += " at " + v.min.String() + " or above; press f to change the filter"
		}
		out = append(out, th.Muted.Render(msg))
	case len(es) == 0:
		out = append(out, th.Muted.Render("no entries at "+v.min.String()+" or above; press f to change the filter"))
	default:
		for i := v.top(es); i < len(es) && len(out) < rows; i++ {
			out = append(out, v.entryRows(th, es[i], cols)...)
		}
	}
	for len(out) < rows {
		out = append(out, "")
	}
	out = out[:rows]
	for i := head; i < len(out); i++ {
		out[i] = th.Fit(out[i], cols)
	}
	return out
}

// tabs draws the category strip: "All", then a tab per category. The selected
// one is in brackets and highlighted; the others are muted and show how many
// entries arrived since you left them.
func (v *Viewer) tabs(th *theme.Theme, cols int, all []Entry) string {
	names := append([]string{""}, v.Categories()...)
	cells := make([]string, len(names))
	active := 0
	for i, name := range names {
		label := name
		if name == "" {
			label = "All"
		}
		if name == v.cat {
			active = i
			cells[i] = th.Key.Render("[" + label + "]")
			continue
		}
		if name != "" {
			unread := 0
			for _, e := range v.visible(all, name) {
				if e.Seq > v.read[name] {
					unread++
				}
			}
			if unread > 0 {
				label += " (" + strconv.Itoa(unread) + ")"
			}
		}
		cells[i] = th.Muted.Render(" " + label + " ")
	}
	// Too many tabs for the width: drop them from the left until the selected
	// one fits, so it is always on screen.
	lo := 0
	for lo < active && text.Width(strings.Join(cells[lo:], " ")) > cols {
		lo++
	}
	row := strings.Join(cells[lo:], " ")
	if lo > 0 {
		row = th.Muted.Render(th.G.Ellipsis) + " " + row
	}
	return th.Fit(row, cols)
}

// entryRows renders one entry: the first row carries the time and level, and
// wrapped continuation rows are indented under the text.
func (v *Viewer) entryRows(th *theme.Theme, e Entry, cols int) []string {
	tag := th.Muted
	switch e.Level {
	case Error:
		tag = th.Red
	case Warn:
		tag = th.Amber
	case Info:
		tag = th.Cyan
	}
	head := th.Muted.Render(e.Time.Format("15:04:05")) + " " + tag.Render(fmt.Sprintf("%-5s", e.Level)) + " "
	body := sanitize(e.Text)
	width := cols - prefixWidth
	if width < 8 {
		return []string{th.Trunc(head+body, cols)}
	}
	lines := text.Wrap(body, width)
	if len(lines) == 0 {
		lines = []string{""}
	}
	out := make([]string, len(lines))
	indent := strings.Repeat(" ", prefixWidth)
	for i, l := range lines {
		if i == 0 {
			out[i] = head + l
		} else {
			out[i] = indent + l
		}
	}
	return out
}

// sanitize replaces control characters, including escape sequences, with a
// space, so a log message can never move the cursor or restyle the screen.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
