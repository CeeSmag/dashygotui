package chart

import (
	"math"
	"strings"

	"github.com/CeeSmag/dashygotui/theme"
)

// The renderers in this file draw a series of values as a waveform. They share
// a convention: values run oldest to newest and are right-aligned, so the right
// edge is always the most recent moment; missing history leaves a gap at the
// left; NaN and non-positive values draw as a gap; and scale is the value that
// fills the full height. In TTY mode (theme.ASCII) they fall back to the
// theme's ASCII ramp and shade blocks.

// tail returns exactly n values: the newest n of vals, right-aligned, with NaN
// padding the left when vals is shorter.
func tail(vals []float64, n int) []float64 {
	out := make([]float64, max(n, 0))
	for i := range out {
		out[i] = math.NaN()
	}
	if len(vals) >= n {
		copy(out, vals[len(vals)-n:])
	} else {
		copy(out[n-len(vals):], vals)
	}
	return out
}

// fraction maps v onto 0..1 against scale. It reports false for a gap: NaN, a
// non-positive value, or a non-positive scale.
func fraction(v, scale float64) (float64, bool) {
	if math.IsNaN(v) || v <= 0 || scale <= 0 {
		return 0, false
	}
	return math.Min(1, v/scale), true
}

// rampLevel picks one of the theme's eight ramp glyphs for a fraction in (0, 1].
// A trickle still shows the lowest glyph.
func rampLevel(f float64) int {
	return min(max(int(math.Ceil(f*8))-1, 0), 7)
}

// Sparkline draws vals as one row w cells wide, each cell a block of height
// proportional to its value. It is the most compact waveform, good beside a
// label. It returns "" when w is not positive.
func Sparkline(th *theme.Theme, vals []float64, scale float64, w int) string {
	if w <= 0 {
		return ""
	}
	var sb strings.Builder
	for _, v := range tail(vals, w) {
		f, ok := fraction(v, scale)
		if !ok {
			sb.WriteByte(' ')
			continue
		}
		sb.WriteString(th.Gradient(f).Render(th.G.Ramp[rampLevel(f)]))
	}
	return sb.String()
}

// BarGraph draws vals as vertical bars, one column per value, h rows tall, with
// eighth-block precision at the top of each bar. It reads like a spectrum or a
// histogram. A multi-row graph is coloured per row, cool at the bottom and hot
// at the top. It returns nil when w or h is not positive.
func BarGraph(th *theme.Theme, vals []float64, scale float64, w, h int) []string {
	if w <= 0 || h <= 0 {
		return nil
	}
	cols := tail(vals, w)
	eighths := make([]int, w)
	for i, v := range cols {
		if f, ok := fraction(v, scale); ok {
			eighths[i] = min(max(int(f*float64(h*8)+0.5), 1), h*8)
		}
	}
	out := make([]string, 0, h)
	for row := range h {
		var sb strings.Builder
		style := th.Gradient(float64(h-row) / float64(h))
		base := (h - 1 - row) * 8 // eighths below this row
		for _, n := range eighths {
			switch fill := n - base; {
			case fill >= 8:
				sb.WriteString(style.Render(th.G.Ramp[7]))
			case fill > 0:
				sb.WriteString(style.Render(th.G.Ramp[fill-1]))
			default:
				sb.WriteByte(' ')
			}
		}
		out = append(out, sb.String())
	}
	return out
}

// LineTrace draws vals as a thin line, like an oscilloscope trace, using Braille
// dots (two samples per cell horizontally, four dots per row vertically).
// Jumps between neighbouring samples are joined so a spike is drawn as a
// stroke, not two dots. In TTY mode it draws one '*' per column. It returns nil
// when w or h is not positive.
func LineTrace(th *theme.Theme, vals []float64, scale float64, w, h int) []string {
	if w <= 0 || h <= 0 {
		return nil
	}
	if th.ASCII {
		return lineTraceASCII(th, vals, scale, w, h)
	}
	n, dotRows := 2*w, 4*h
	samples := tail(vals, n)
	// dots[y][x] is true where a dot is lit.
	dots := make([][]bool, dotRows)
	for y := range dots {
		dots[y] = make([]bool, n)
	}
	prevY := -1
	for x, v := range samples {
		f, ok := fraction(v, scale)
		if math.IsNaN(v) {
			prevY = -1 // a gap breaks the line
			continue
		}
		if !ok {
			f = 0 // zero or below the floor sits on the bottom edge
		}
		y := int(math.Round((1 - f) * float64(dotRows-1)))
		dots[y][x] = true
		if prevY >= 0 {
			for yy := min(prevY, y); yy <= max(prevY, y); yy++ {
				dots[yy][x] = true
			}
		}
		prevY = y
	}

	leftBits := [4]rune{0x01, 0x02, 0x04, 0x40}
	rightBits := [4]rune{0x08, 0x10, 0x20, 0x80}
	out := make([]string, 0, h)
	for cy := range h {
		var sb strings.Builder
		style := th.Gradient(1 - (float64(cy)+0.5)/float64(h))
		for cx := range w {
			var mask rune
			for dy := range 4 {
				if dots[cy*4+dy][2*cx] {
					mask |= leftBits[dy]
				}
				if dots[cy*4+dy][2*cx+1] {
					mask |= rightBits[dy]
				}
			}
			if mask == 0 {
				sb.WriteByte(' ')
				continue
			}
			sb.WriteString(style.Render(string(0x2800 + mask)))
		}
		out = append(out, sb.String())
	}
	return out
}

// lineTraceASCII is [LineTrace] for TTY mode: one '*' per column, at the row of
// the column's value.
func lineTraceASCII(th *theme.Theme, vals []float64, scale float64, w, h int) []string {
	grid := make([][]byte, h)
	for r := range grid {
		grid[r] = []byte(strings.Repeat(" ", w))
	}
	for x, v := range tail(vals, w) {
		if math.IsNaN(v) {
			continue
		}
		f, _ := fraction(v, scale)
		grid[int(math.Round((1-f)*float64(h-1)))][x] = '*'
	}
	out := make([]string, h)
	for r, line := range grid {
		out[r] = th.Gradient(1 - (float64(r)+0.5)/float64(h)).Render(string(line))
	}
	return out
}

// DualWave draws two streams mirrored around a centre axis, h rows tall and w
// cells wide: up grows upward from the axis (a download, a read) and down grows
// downward (an upload, a write), both filled with Braille dots. It is the
// classic bidirectional traffic graph. Both streams share scale so their
// heights compare. The axis takes the middle row; with h of 1 only the axis is
// drawn. It returns nil when w or h is not positive.
func DualWave(th *theme.Theme, up, down []float64, scale float64, w, h int) []string {
	if w <= 0 || h <= 0 {
		return nil
	}
	axis := h / 2
	topRows, bottomRows := axis, h-axis-1
	n := 2 * w

	totals := func(vals []float64, rows int) []int {
		out := make([]int, n)
		for i, v := range tail(vals, n) {
			out[i] = TotalDots(v, scale, rows*4)
		}
		return out
	}
	upTotals, downTotals := totals(up, topRows), totals(down, bottomRows)

	out := make([]string, 0, h)
	for row := range topRows {
		var sb strings.Builder
		style := th.Gradient(float64(topRows-row) / float64(topRows))
		for c := range w {
			glyph := UpCell(th.ASCII, rowDots(upTotals[2*c], row, topRows), rowDots(upTotals[2*c+1], row, topRows))
			sb.WriteString(paintCell(style.Render, glyph))
		}
		out = append(out, sb.String())
	}
	out = append(out, th.Track.Render(strings.Repeat(th.G.SliderRail, w)))
	for row := range bottomRows {
		var sb strings.Builder
		style := th.Gradient(float64(row+1) / float64(bottomRows))
		for c := range w {
			l := min(max(downTotals[2*c]-row*4, 0), 4)
			r := min(max(downTotals[2*c+1]-row*4, 0), 4)
			sb.WriteString(paintCell(style.Render, DownCell(th.ASCII, l, r)))
		}
		out = append(out, sb.String())
	}
	return out
}

// paintCell styles a glyph, leaving blanks unstyled.
func paintCell(render func(...string) string, glyph rune) string {
	if glyph == ' ' {
		return " "
	}
	return render(string(glyph))
}

// HeatStrip draws vals as one row w cells wide in which only colour (and, in TTY
// mode, ramp weight) carries the value: a compact way to show many series
// stacked, such as per-core load. Gaps draw as the theme's empty-meter glyph.
// It returns "" when w is not positive.
func HeatStrip(th *theme.Theme, vals []float64, scale float64, w int) string {
	if w <= 0 {
		return ""
	}
	var sb strings.Builder
	for _, v := range tail(vals, w) {
		f, ok := fraction(v, scale)
		switch {
		case !ok:
			sb.WriteString(th.Track.Render(th.G.MeterEmpty))
		case th.ASCII:
			sb.WriteString(th.Gradient(f).Render(th.G.Ramp[rampLevel(f)]))
		default:
			sb.WriteString(th.Gradient(f).Render("█"))
		}
	}
	return sb.String()
}
