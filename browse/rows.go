package browse

import (
	"strconv"
	"strings"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

const (
	itemsCol = 7
	sizeCol  = 9
	dateCol  = 16
	dateFmt  = "2006-01-02 15:04"
)

// Rows draws the header and visible entries for a body of cols x bodyRows
// cells. The cursor is kept in view.
func (n *Navigator) Rows(th *theme.Theme, cols, bodyRows int) []string {
	if bodyRows <= 0 || cols <= 0 {
		return nil
	}
	vis := n.Visible()
	n.clampCursor()
	showItems, showSize, showDate := columns(cols)
	header := formatHeader(th, cols, showItems, showSize, showDate)
	listRows := bodyRows - 1
	if listRows < 1 {
		return []string{header}
	}
	start := 0
	if len(vis) > listRows {
		start = max(n.cursor-listRows/2, 0)
		start = min(start, len(vis)-listRows)
	}
	out := make([]string, 0, bodyRows)
	out = append(out, header)
	for i := start; i < min(start+listRows, len(vis)); i++ {
		out = append(out, formatRow(th, vis[i], i == n.cursor, cols, showItems, showSize, showDate))
	}
	for len(out) < bodyRows {
		out = append(out, th.Fit("", cols))
	}
	return out
}

func columns(cols int) (items, size, date bool) {
	need := 2 + 8
	date = cols >= need+1+dateCol
	if date {
		need += 1 + dateCol
	}
	items = cols >= need+1+itemsCol
	if items {
		need += 1 + itemsCol
	}
	size = cols >= need+1+sizeCol
	return items, size, date
}

func formatHeader(th *theme.Theme, cols int, showItems, showSize, showDate bool) string {
	nameW := nameWidth(cols, showItems, showSize, showDate)
	line := "  " + padRight("NAME", nameW)
	if showItems {
		line += " " + padLeft("ITEMS", itemsCol)
	}
	if showSize {
		line += " " + padLeft("SIZE", sizeCol)
	}
	if showDate {
		line += " " + padRight("MODIFIED", dateCol)
	}
	return th.Muted.Render(th.Fit(line, cols))
}

func formatRow(th *theme.Theme, e Entry, selected bool, cols int, showItems, showSize, showDate bool) string {
	nameW := nameWidth(cols, showItems, showSize, showDate)
	ptr := "  "
	if selected {
		ptr = th.Key.Render(th.G.Pointer) + " "
	}
	namePart := padRight(th.Trunc(e.Name+"/", nameW), nameW)
	if selected {
		namePart = th.White.Render(namePart)
	}
	line := ptr + namePart
	if showItems {
		line += " " + padLeft(itemsText(e.Files), itemsCol)
	}
	if showSize {
		line += " " + padLeft(sizeText(e.Bytes), sizeCol)
	}
	if showDate {
		d := "-"
		if !e.ModTime.IsZero() {
			d = e.ModTime.Format(dateFmt)
		}
		line += " " + padRight(d, dateCol)
	}
	return th.Fit(line, cols)
}

func nameWidth(cols int, showItems, showSize, showDate bool) int {
	w := cols - 2
	if showItems {
		w -= 1 + itemsCol
	}
	if showSize {
		w -= 1 + sizeCol
	}
	if showDate {
		w -= 1 + dateCol
	}
	return max(w, 1)
}

func itemsText(n int64) string {
	if n < 0 {
		return "-"
	}
	return strconv.FormatInt(n, 10)
}

func sizeText(n int64) string {
	if n < 0 {
		return "-"
	}
	return text.Bytes(n)
}

func padRight(s string, w int) string {
	return text.Pad(text.TruncWith(s, w, ""), w)
}

func padLeft(s string, w int) string {
	s = text.TruncWith(s, w, "")
	if gap := w - text.Width(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}
