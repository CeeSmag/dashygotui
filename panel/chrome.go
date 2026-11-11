package panel

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// LabelWidth is the width of the label column in [Row], so values line up down
// a panel.
const LabelWidth = 8

// Row is one status row: a muted label in a fixed-width column, then its value.
func Row(th *theme.Theme, label, value string) string {
	return th.Muted.Render(text.Pad(label, LabelWidth)) + " " + value
}

// Segment is one piece of the header strip. Lower priorities are dropped first
// when the strip does not fit.
type Segment struct {
	Text string
	Prio int
}

// Hint is one key hint in the footer, rendered as "[Key] Label". Lower
// priorities are dropped first when the strip does not fit.
type Hint struct {
	Key   string
	Label string
	Style lipgloss.Style // style of the "[Key]" part
	Prio  int
	// Tinted draws the label in Style too, instead of the muted style.
	Tinted bool
}

// FitSegments joins segs with " │ " separators, dropping the least important
// until the strip fits in budget cells. The first segment is never dropped, so
// the header always says which program this is.
func FitSegments(th *theme.Theme, segs []Segment, budget int) string {
	keep := make([]bool, len(segs))
	for i := range keep {
		keep[i] = true
	}
	for {
		strip := joinSegments(th, segs, keep)
		if text.Width(strip) <= budget || !dropWeakestSegment(segs, keep) {
			return strip
		}
	}
}

func joinSegments(th *theme.Theme, segs []Segment, keep []bool) string {
	parts := make([]string, 0, len(segs))
	for i, s := range segs {
		if keep[i] && s.Text != "" {
			parts = append(parts, s.Text)
		}
	}
	return strings.Join(parts, th.Border.Render(" │ "))
}

// dropWeakestSegment drops the lowest-priority segment still kept, except the
// first. It reports whether anything was dropped.
func dropWeakestSegment(segs []Segment, keep []bool) bool {
	worst, found := 0, false
	for i := 1; i < len(segs); i++ {
		if !keep[i] {
			continue
		}
		if !found || segs[i].Prio < segs[worst].Prio {
			worst, found = i, true
		}
	}
	if found {
		keep[worst] = false
	}
	return found
}

// Header draws the strip along the top of a page: the segments, fitted into a
// cols-wide border.
func Header(th *theme.Theme, segs []Segment, cols int) string {
	return top(th, "", FitSegments(th, segs, cols-8), cols)
}

// FitHints renders the hint strip, dropping the least important hints until it
// fits in budget cells. Give the hint that must survive the highest priority.
func FitHints(th *theme.Theme, hints []Hint, budget int) string {
	keep := make([]bool, len(hints))
	for i := range keep {
		keep[i] = true
	}
	for {
		strip := joinHints(th, hints, keep)
		if text.Width(strip) <= budget {
			return strip
		}
		worst, found := -1, false
		for i := range hints {
			if !keep[i] {
				continue
			}
			if !found || hints[i].Prio < hints[worst].Prio {
				worst, found = i, true
			}
		}
		if !found {
			return strip
		}
		keep[worst] = false
	}
}

func joinHints(th *theme.Theme, hints []Hint, keep []bool) string {
	var sb strings.Builder
	for i, h := range hints {
		if !keep[i] {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		label := th.Muted
		if h.Tinted {
			label = h.Style
		}
		sb.WriteString(h.Style.Render("["+h.Key+"]") + " " + label.Render(h.Label))
	}
	return sb.String()
}
