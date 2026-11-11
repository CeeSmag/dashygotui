package logs

import (
	"fmt"
	"sort"
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
type Viewer struct {
	buf    *Buffer
	anchor uint64 // Seq of the entry at the top while scrolled; 0 means live
	min    Level  // entries below this level are hidden
}

// NewViewer returns a viewer of buf, live and showing every level.
func NewViewer(buf *Buffer) *Viewer { return &Viewer{buf: buf} }

// Live reports whether the view is at the top, following the newest entry.
func (v *Viewer) Live() bool { return v.anchor == 0 }

// MinLevel is the lowest level shown.
func (v *Viewer) MinLevel() Level { return v.min }

// entries is the visible entries, newest first.
func (v *Viewer) entries() []Entry {
	all := v.buf.Snapshot()
	if v.min == Debug {
		return all
	}
	out := all[:0]
	for _, e := range all {
		if e.Level >= v.min {
			out = append(out, e)
		}
	}
	return out
}

// top is the index, into es, of the entry at the top of the view.
func (v *Viewer) top(es []Entry) int {
	if v.anchor == 0 || len(es) == 0 {
		return 0
	}
	// es is newest first, so Seq decreases: find the first entry at or older
	// than the anchor. If the anchor was evicted that is simply the oldest kept.
	i := sort.Search(len(es), func(i int) bool { return es[i].Seq <= v.anchor })
	return min(i, len(es)-1)
}

// Newer reports how many visible entries are above the top of the view: zero
// when live.
func (v *Viewer) Newer() int { return v.top(v.entries()) }

// scrollTo moves the top of the view to index i of the visible entries.
func (v *Viewer) scrollTo(es []Entry, i int) {
	i = min(max(i, 0), max(len(es)-1, 0))
	if i == 0 || len(es) == 0 {
		v.anchor = 0
		return
	}
	v.anchor = es[i].Seq
}

// HandleKey scrolls on the named key and reports whether it used it. rows is
// the height of the view, used as the page size. The keys are up/k (newer),
// down/j (older), pgup, pgdown, home (back to live), end (oldest) and f, which
// cycles the minimum level shown.
func (v *Viewer) HandleKey(name string, rows int) bool {
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
		v.anchor = 0
	case "end":
		v.scrollTo(es, len(es)-1)
	case "f":
		v.min = (v.min + 1) % (Error + 1)
		v.anchor = 0 // the old anchor may now be filtered out
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
// entry first starting at the top of the view.
func (v *Viewer) Rows(th *theme.Theme, cols, rows int) []string {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	es := v.entries()
	out := make([]string, 0, rows)
	switch {
	case len(es) == 0 && v.buf.Len() == 0:
		out = append(out, th.Muted.Render("no log entries yet"))
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
	for i, l := range out {
		out[i] = th.Fit(l, cols)
	}
	return out
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
