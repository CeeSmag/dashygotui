// Package text provides string helpers that measure and cut text in terminal
// cells rather than bytes or runes.
//
// Every row a dashboard draws must be exactly as wide as its allotted space,
// yet the strings are full of ANSI styling, CJK characters and emoji. All
// measuring and cutting therefore goes through x/ansi, which counts terminal
// cells and treats escape sequences as zero width. That is the same width
// method lipgloss uses, so a string measured here lays out the way lipgloss
// joins it.
package text

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Ellipsis marks text that [Trunc] cut short.
const Ellipsis = "…"

// Width returns the width of s in terminal cells, ignoring ANSI sequences.
func Width(s string) int { return ansi.StringWidth(s) }

// Trunc cuts s to at most w cells, ending in [Ellipsis] when something was cut.
// A non-positive w yields the empty string.
func Trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if Width(s) <= w {
		return s
	}
	if w == 1 {
		return Ellipsis
	}
	return ansi.Truncate(s, w, Ellipsis)
}

// Pad right-pads s with spaces to exactly w cells. It does nothing when s is
// already at least w cells wide.
func Pad(s string, w int) string {
	if gap := w - Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

// Fit makes s exactly w cells wide, truncating with [Ellipsis] or padding with
// spaces. It is the last thing every row should pass through. A non-positive w
// yields the empty string.
func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return Pad(Trunc(s, w), w)
}

// Center pads s on both sides so that it sits in the middle of w cells. Text
// wider than w is truncated.
func Center(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = Trunc(s, w)
	left := max((w-Width(s))/2, 0)
	return Pad(strings.Repeat(" ", left)+s, w)
}

// Wrap breaks s into lines of at most w cells, preferring breaks at spaces and
// hyphens and falling back to a hard break for a word too long on its own. A
// non-positive w yields nil.
func Wrap(s string, w int) []string {
	if w <= 0 {
		return nil
	}
	return strings.Split(ansi.Wrap(s, w, " -"), "\n")
}

// Slice returns the cells [left, right) of s, keeping the styling that was
// active at the cut. It is how a wide row is sliced into columns.
func Slice(s string, left, right int) string {
	if right <= left {
		return ""
	}
	return ansi.Cut(s, left, right)
}

// OneLine flattens s onto a single line by collapsing every run of whitespace,
// newlines included, into one space. Use it on messages shown in a one-row
// footer.
func OneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
