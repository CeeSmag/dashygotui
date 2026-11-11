package chart

import (
	"math"
	"strings"

	"github.com/CeeSmag/dashygotui/theme"
)

// shades maps (left dots, right dots), each 0..4 as index left*5+right, onto
// the shading glyphs that exist in every VGA/Linux console font. It is used
// instead of Braille when the theme is in ASCII mode.
const shades = " ░░▒▒" + "░░▒▒█" + "░▒▒▒█" + "▒▒▒██" + "▒████"

// ClampDots limits a dot count to the 0..4 a Braille cell column can hold.
func ClampDots(n int) int { return min(max(n, 0), 4) }

// UpCell returns the glyph for a pair of Braille columns filled from the
// bottom up (a download or read stream): the left column gets l dots and the
// right r dots, each 0..4. With ascii set it returns a shade block instead.
func UpCell(ascii bool, l, r int) rune {
	return cell(ascii, l, r, true)
}

// DownCell is like [UpCell] but fills from the top down, for the mirrored
// half of a two-stream waveform (an upload or write stream).
func DownCell(ascii bool, l, r int) rune {
	return cell(ascii, l, r, false)
}

func cell(ascii bool, l, r int, fromBottom bool) rune {
	cl, cr := ClampDots(l), ClampDots(r)
	if cl == 0 && cr == 0 {
		return ' '
	}
	if ascii {
		return []rune(shades)[cl*5+cr]
	}
	// Dot bits by row, top to bottom, for the left and right columns.
	leftBits := [4]rune{0x01, 0x02, 0x04, 0x40}
	rightBits := [4]rune{0x08, 0x10, 0x20, 0x80}
	var mask rune
	for i := range cl {
		if fromBottom {
			mask |= leftBits[3-i]
		} else {
			mask |= leftBits[i]
		}
	}
	for i := range cr {
		if fromBottom {
			mask |= rightBits[3-i]
		} else {
			mask |= rightBits[i]
		}
	}
	return 0x2800 + mask
}

// TotalDots maps a value onto the dots available in a graph. A non-zero
// positive value always lights at least one dot, so a trickle stays visible.
// NaN, non-positive values and non-positive scales or sizes give 0.
func TotalDots(v, scale float64, available int) int {
	if math.IsNaN(v) || v <= 0 || scale <= 0 || available <= 0 {
		return 0
	}
	d := int(math.Min(1, v/scale)*float64(available) + 0.5)
	return min(max(d, 1), available)
}

// rowDots is how many of a row's four dots a total fills, for the row at
// index row counting from the top of an h-row graph.
func rowDots(total, row, h int) int {
	return total - (h-1-row)*4
}

// Braille draws vals as an h-row Braille graph w cells wide. It expects 2*w
// values (two samples per cell, oldest to newest); extra samples are dropped
// from the old end and missing ones leave a gap at the left, so the right edge
// is always the most recent moment. NaN draws as a gap. scale is the value
// that fills the full height.
//
// A multi-row graph is coloured per row, cool at the bottom and hot at the
// top, so height reads as intensity; a single-row sparkline takes the colour
// of each cell's own peak value.
func Braille(th *theme.Theme, vals []float64, scale float64, w, h int) []string {
	if w <= 0 || h <= 0 {
		return nil
	}
	n := 2 * w
	if len(vals) > n {
		vals = vals[len(vals)-n:]
	}
	totals := make([]int, n)
	offset := n - len(vals)
	for i, v := range vals {
		totals[offset+i] = TotalDots(v, scale, h*4)
	}

	out := make([]string, 0, h)
	for row := range h {
		var sb strings.Builder
		rowStyle := th.Gradient(float64(h-row) / float64(h))
		for c := range w {
			l := rowDots(totals[2*c], row, h)
			r := rowDots(totals[2*c+1], row, h)
			glyph := UpCell(th.ASCII, l, r)
			switch {
			case glyph == ' ':
				sb.WriteRune(' ')
			case h == 1:
				peak := math.Max(float64(max(l, 0)), float64(max(r, 0))) / 4
				sb.WriteString(th.Gradient(peak).Render(string(glyph)))
			default:
				sb.WriteString(rowStyle.Render(string(glyph)))
			}
		}
		out = append(out, sb.String())
	}
	return out
}
