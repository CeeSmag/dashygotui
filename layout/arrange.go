package layout

import "github.com/CeeSmag/dashygotui/panel"

// Mode tells [Arrange] how to place a page's panels when the terminal has
// room. On narrow terminals every mode degrades to vertical stacking.
type Mode int

// The arrangement modes. Auto is the zero value.
const (
	// Auto picks the natural arrangement for the panel count:
	// 1 full, 2 side by side, 3 one on top and two below, 4 a 2x2 grid.
	Auto Mode = iota
	// Columns places all panels side by side.
	Columns
	// Rows stacks all panels vertically.
	Rows
	// OneTopTwoBottom is three panels: one full-width above two side by side.
	OneTopTwoBottom
	// TwoTopOneBottom is three panels: two side by side above one full-width.
	TwoTopOneBottom
	// Grid2x2 is four panels in a 2x2 grid.
	Grid2x2
)

// Panel is one panel to place. The panel package does the framing; Arrange only
// decides how many cells each panel gets.
type Panel struct {
	// Want reports how many rows the panel would like at width w, borders
	// included. A nil Want means "no preference".
	Want func(w int) int
	// Draw renders the panel into exactly w x h cells.
	Draw func(w, h int) []string
}

func (p Panel) want(w int) int {
	if p.Want == nil {
		return 0
	}
	return p.Want(w)
}

// Arrange places panels in a cols x body area using gap cells between
// neighbours, and returns exactly body lines of exactly cols cells.
func Arrange(panels []Panel, mode Mode, gap, cols, body int) []string {
	n := len(panels)
	if n == 0 || cols <= 0 || body <= 0 {
		return nil
	}
	return panel.Frame(arrange(panels, mode, max(gap, 0), cols, body), cols, body)
}

func arrange(panels []Panel, mode Mode, gap, cols, body int) []string {
	n := len(panels)
	draw := func(i, w, h int) []string { return panels[i].Draw(w, h) }
	if n == 1 {
		return draw(0, cols, body)
	}
	wideEnough := cols >= TightCols && body >= 8

	switch n {
	case 2:
		if mode != Rows && wideEnough {
			w0, w1 := split2(cols, gap)
			return panel.HJoin(gap, draw(0, w0, body), draw(1, w1, body))
		}
	case 3:
		switch effective(mode, OneTopTwoBottom) {
		case Columns:
			if cols >= TightCols && body >= 8 {
				ws := splitN(cols, gap, 3)
				return panel.HJoin(gap, draw(0, ws[0], body), draw(1, ws[1], body), draw(2, ws[2], body))
			}
		case OneTopTwoBottom:
			if wideEnough && body >= 6 {
				topH, botH := splitHeights(panels[0], cols, body)
				w1, w2 := split2(cols, gap)
				return panel.VStack(draw(0, cols, topH), panel.HJoin(gap, draw(1, w1, botH), draw(2, w2, botH)))
			}
		case TwoTopOneBottom:
			if wideEnough && body >= 6 {
				topH, botH := splitHeights(panels[2], cols, body)
				w0, w1 := split2(cols, gap)
				return panel.VStack(panel.HJoin(gap, draw(0, w0, topH), draw(1, w1, topH)), draw(2, cols, botH))
			}
		}
	case 4:
		switch effective4(mode) {
		case Columns:
			if cols >= WideCols && body >= 8 {
				ws := splitN(cols, gap, 4)
				return panel.HJoin(gap, draw(0, ws[0], body), draw(1, ws[1], body), draw(2, ws[2], body), draw(3, ws[3], body))
			}
		case Grid2x2:
			if wideEnough && body >= 6 {
				topH := max(body/2, 3)
				botH := max(body-topH, 3)
				w0, w1 := split2(cols, gap)
				return panel.VStack(
					panel.HJoin(gap, draw(0, w0, topH), draw(1, w1, topH)),
					panel.HJoin(gap, draw(2, w0, botH), draw(3, w1, botH)),
				)
			}
		}
	default:
		if mode == Columns && cols >= WideCols && body >= 8 {
			ws := splitN(cols, gap, n)
			blocks := make([][]string, n)
			for i := range blocks {
				blocks[i] = draw(i, ws[i], body)
			}
			return panel.HJoin(gap, blocks...)
		}
	}
	return stack(panels, cols, body)
}

// effective resolves Auto to the natural mode for three panels.
func effective(mode, auto Mode) Mode {
	if mode == Auto {
		return auto
	}
	return mode
}

// effective4 resolves the modes that only make sense for three panels to the
// grid, which is the natural arrangement of four.
func effective4(mode Mode) Mode {
	switch mode {
	case Auto, OneTopTwoBottom, TwoTopOneBottom:
		return Grid2x2
	}
	return mode
}

// split2 divides cols into two widths separated by gap.
func split2(cols, gap int) (int, int) {
	avail := max(cols-gap, 0)
	left := avail / 2
	return left, avail - left
}

// splitN divides cols into n widths separated by n-1 gaps. The remainder goes
// to the leftmost panels, one cell each.
func splitN(cols, gap, n int) []int {
	if n <= 0 {
		return nil
	}
	avail := max(cols-gap*(n-1), 0)
	base, rem := avail/n, avail%n
	out := make([]int, n)
	for i := range out {
		out[i] = base
		if i < rem {
			out[i]++
		}
	}
	return out
}

// splitHeights chooses the row counts of two stacked groups: the group
// described by top gets what it wants, capped at half the body.
func splitHeights(top Panel, cols, body int) (topH, botH int) {
	limit := max(body/2, 3)
	topH = limit
	if want := top.want(cols); want > 0 {
		topH = min(want, limit)
	}
	return topH, max(body-topH, 3)
}

// stack places panels one above another within body rows. Each panel gets what
// it wants up to a fair share; the last takes whatever is left.
func stack(panels []Panel, cols, body int) []string {
	n := len(panels)
	heights := make([]int, n)
	rem := body
	for i := range panels {
		if i == n-1 {
			heights[i] = max(rem, 3)
			break
		}
		fair := max(rem/(n-i), 3)
		h := fair
		if want := panels[i].want(cols); want > 0 {
			h = min(want, fair)
		}
		heights[i] = max(h, 3)
		rem -= heights[i]
	}
	blocks := make([][]string, n)
	for i := range panels {
		blocks[i] = panels[i].Draw(cols, heights[i])
	}
	return panel.VStack(blocks...)
}
