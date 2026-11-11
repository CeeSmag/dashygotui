package panel

import (
	"fmt"
	"strings"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// Blank returns an empty block of w cells by h rows, for a layout slot with
// nothing in it.
func Blank(w, h int) []string {
	out := make([]string, 0, max(h, 0))
	for range h {
		out = append(out, strings.Repeat(" ", max(w, 0)))
	}
	return out
}

// HJoin places blocks side by side with gap spaces between them, padding short
// blocks so every result row has the same width.
func HJoin(gap int, blocks ...[]string) []string {
	height := 0
	widths := make([]int, len(blocks))
	for i, b := range blocks {
		height = max(height, len(b))
		for _, line := range b {
			widths[i] = max(widths[i], text.Width(line))
		}
	}
	sep := strings.Repeat(" ", max(gap, 0))
	out := make([]string, 0, height)
	for r := range height {
		var sb strings.Builder
		for i, b := range blocks {
			if i > 0 {
				sb.WriteString(sep)
			}
			line := ""
			if r < len(b) {
				line = b[r]
			}
			sb.WriteString(text.Fit(line, widths[i]))
		}
		out = append(out, sb.String())
	}
	return out
}

// VStack stacks blocks top to bottom.
func VStack(blocks ...[]string) []string {
	out := []string{}
	for _, b := range blocks {
		out = append(out, b...)
	}
	return out
}

// Frame normalises a block to exactly rows lines of exactly cols cells, which
// is what the renderer hands to Bubble Tea. A non-positive rows or cols yields
// lines that are empty or no lines at all.
func Frame(lines []string, cols, rows int) []string {
	out := make([]string, 0, max(rows, 0))
	for i := range rows {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out = append(out, text.Fit(line, cols))
	}
	return out
}

// ClipHeight trims a block to at most h rows, keeping the top and the last
// two rows. Those two hold the keys and the bottom border of a pop-up, and a
// clipped pop-up with no visible keys is a dead end.
func ClipHeight(block []string, h int) []string {
	if h <= 0 || len(block) <= h {
		return block
	}
	if h <= 2 {
		return block[len(block)-h:]
	}
	out := append([]string(nil), block[:h-2]...)
	return append(out, block[len(block)-2:]...)
}

// Overlay centres box over base (a cols x rows frame) and returns the result.
// A box taller than the screen is clipped with [ClipHeight].
func Overlay(base, box []string, cols, rows int) []string {
	if len(box) == 0 {
		return base
	}
	box = ClipHeight(box, rows)
	w := 0
	for _, line := range box {
		w = max(w, text.Width(line))
	}
	left := max((cols-w)/2, 0)
	topRow := max((rows-len(box))/2, 0)
	out := append([]string(nil), base...)
	for i, line := range box {
		r := topRow + i
		if r < 0 || r >= len(out) {
			continue
		}
		prefix := text.Fit(text.Slice(out[r], 0, left), left)
		suffix := text.Slice(out[r], left+w, cols)
		out[r] = text.Fit(prefix+text.Fit(line, w)+suffix, cols)
	}
	return out
}

// TooSmall is what a dashboard shows below its minimum size: the size it has,
// the size it needs, and the one key that still works. The result is exactly
// rows lines of cols cells.
func TooSmall(th *theme.Theme, cols, rows, minCols, minRows int) []string {
	lines := []string{
		th.Amber.Render("Terminal too small"),
		th.Muted.Render(fmt.Sprintf("%dx%d, need %dx%d", cols, rows, minCols, minRows)),
		th.Muted.Render("q quits"),
	}
	out := make([]string, 0, max(rows, 0))
	topRow := (rows - len(lines)) / 2
	for i := range rows {
		if j := i - topRow; j >= 0 && j < len(lines) {
			out = append(out, text.Center(lines[j], cols))
			continue
		}
		out = append(out, strings.Repeat(" ", max(cols, 0)))
	}
	return out
}
