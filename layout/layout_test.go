package layout_test

import (
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/layout"
	"github.com/CeeSmag/dashygotui/text"
)

// fill is a panel that paints every cell with its own letter, so a test can see
// where Arrange put it.
func fill(letter string) layout.Panel {
	return layout.Panel{
		Draw: func(w, h int) []string {
			out := make([]string, h)
			for i := range out {
				out[i] = strings.Repeat(letter, w)
			}
			return out
		},
	}
}

func panels(letters string) []layout.Panel {
	var out []layout.Panel
	for _, r := range letters {
		out = append(out, fill(string(r)))
	}
	return out
}

func assertExact(t *testing.T, what string, got []string, cols, rows int) {
	t.Helper()
	if len(got) != rows {
		t.Fatalf("%s: %d rows, want %d", what, len(got), rows)
	}
	for i, line := range got {
		if w := text.Width(line); w != cols {
			t.Fatalf("%s: row %d width %d, want %d", what, i, w, cols)
		}
	}
}

func TestArrangeAlwaysFillsTheArea(t *testing.T) {
	modes := []layout.Mode{layout.Auto, layout.Columns, layout.Rows, layout.OneTopTwoBottom, layout.TwoTopOneBottom, layout.Grid2x2}
	for n := 1; n <= 6; n++ {
		for _, mode := range modes {
			for _, cols := range []int{1, 2, 7, 39, 40, 41, 75, 79, 80, 81, 99, 100, 101, 130} {
				for _, body := range []int{1, 2, 3, 5, 6, 7, 8, 9, 14, 30} {
					for _, gap := range []int{0, 2, 10} {
						got := layout.Arrange(panels("ABCDEF"[:n]), mode, gap, cols, body)
						assertExact(t, "Arrange", got, cols, body)
					}
				}
			}
		}
	}
}

func TestArrangeDegenerateInput(t *testing.T) {
	if got := layout.Arrange(nil, layout.Auto, 2, 80, 20); got != nil {
		t.Errorf("no panels: %v", got)
	}
	for _, sz := range [][2]int{{0, 10}, {10, 0}, {-1, 5}, {5, -1}} {
		if got := layout.Arrange(panels("AB"), layout.Auto, 2, sz[0], sz[1]); got != nil {
			t.Errorf("size %v: %v", sz, got)
		}
	}
	// A negative gap behaves as zero.
	assertExact(t, "negative gap", layout.Arrange(panels("AB"), layout.Auto, -5, 90, 12), 90, 12)
}

// firstRow and column helpers for placement assertions.
func row(lines []string, i int) string { return lines[i] }

func TestArrangePlacement(t *testing.T) {
	const cols, body = 100, 20
	tests := []struct {
		name    string
		letters string
		mode    layout.Mode
		cols    int
		body    int
		check   func(t *testing.T, got []string)
	}{
		{
			name: "one panel takes everything", letters: "A", mode: layout.Auto, cols: cols, body: body,
			check: func(t *testing.T, got []string) {
				if strings.Contains(strings.Join(got, ""), " ") {
					t.Error("a single panel must fill every cell")
				}
			},
		},
		{
			name: "two wide are side by side", letters: "AB", mode: layout.Auto, cols: cols, body: body,
			check: func(t *testing.T, got []string) {
				r := row(got, 0)
				if !strings.Contains(r, "A") || !strings.Contains(r, "B") {
					t.Errorf("row 0 = %q, want both panels", r)
				}
			},
		},
		{
			name: "two narrow are stacked", letters: "AB", mode: layout.Auto, cols: 60, body: body,
			check: func(t *testing.T, got []string) {
				if strings.Contains(row(got, 0), "B") || !strings.Contains(row(got, len(got)-1), "B") {
					t.Errorf("want A above B, got first=%q last=%q", row(got, 0), row(got, len(got)-1))
				}
			},
		},
		{
			name: "rows mode stacks even when wide", letters: "AB", mode: layout.Rows, cols: cols, body: body,
			check: func(t *testing.T, got []string) {
				if strings.Contains(row(got, 0), "B") {
					t.Errorf("row 0 = %q, want only A", row(got, 0))
				}
			},
		},
		{
			name: "three auto: one on top, two below", letters: "ABC", mode: layout.Auto, cols: cols, body: body,
			check: func(t *testing.T, got []string) {
				if strings.ContainsAny(row(got, 0), "BC") {
					t.Errorf("row 0 = %q, want only A", row(got, 0))
				}
				last := row(got, len(got)-1)
				if !strings.Contains(last, "B") || !strings.Contains(last, "C") {
					t.Errorf("last row = %q, want B and C side by side", last)
				}
			},
		},
		{
			name: "three two-top-one-bottom", letters: "ABC", mode: layout.TwoTopOneBottom, cols: cols, body: body,
			check: func(t *testing.T, got []string) {
				first := row(got, 0)
				if !strings.Contains(first, "A") || !strings.Contains(first, "B") || strings.Contains(first, "C") {
					t.Errorf("row 0 = %q, want A and B", first)
				}
				if strings.ContainsAny(row(got, len(got)-1), "AB") {
					t.Errorf("last row = %q, want only C", row(got, len(got)-1))
				}
			},
		},
		{
			name: "three columns", letters: "ABC", mode: layout.Columns, cols: cols, body: body,
			check: func(t *testing.T, got []string) {
				r := row(got, 0)
				a, b, c := strings.Index(r, "A"), strings.Index(r, "B"), strings.Index(r, "C")
				if a >= b || b >= c {
					t.Errorf("row 0 = %q, want A B C left to right", r)
				}
			},
		},
		{
			name: "four auto is a 2x2 grid", letters: "ABCD", mode: layout.Auto, cols: cols, body: body,
			check: func(t *testing.T, got []string) {
				first, last := row(got, 0), row(got, len(got)-1)
				if !strings.Contains(first, "A") || !strings.Contains(first, "B") || strings.ContainsAny(first, "CD") {
					t.Errorf("row 0 = %q, want A and B", first)
				}
				if !strings.Contains(last, "C") || !strings.Contains(last, "D") || strings.ContainsAny(last, "AB") {
					t.Errorf("last row = %q, want C and D", last)
				}
			},
		},
		{
			name: "four columns needs wide terminal", letters: "ABCD", mode: layout.Columns, cols: 120, body: body,
			check: func(t *testing.T, got []string) {
				r := row(got, 0)
				for _, l := range "ABCD" {
					if !strings.ContainsRune(r, l) {
						t.Errorf("row 0 = %q, missing %c", r, l)
					}
				}
			},
		},
		{
			name: "five columns", letters: "ABCDE", mode: layout.Columns, cols: 130, body: body,
			check: func(t *testing.T, got []string) {
				for _, l := range "ABCDE" {
					if !strings.ContainsRune(row(got, 0), l) {
						t.Errorf("row 0 = %q, missing %c", row(got, 0), l)
					}
				}
			},
		},
		{
			name: "five auto stacks", letters: "ABCDE", mode: layout.Auto, cols: cols, body: 30,
			check: func(t *testing.T, got []string) {
				if strings.Contains(row(got, 0), "B") {
					t.Errorf("row 0 = %q, want only A", row(got, 0))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layout.Arrange(panels(tt.letters), tt.mode, 2, tt.cols, tt.body)
			assertExact(t, tt.name, got, tt.cols, tt.body)
			tt.check(t, got)
		})
	}
}

func TestArrangeHonoursWantedHeights(t *testing.T) {
	short := layout.Panel{Want: func(int) int { return 4 }, Draw: fill("S").Draw}
	got := layout.Arrange([]layout.Panel{short, fill("L")}, layout.Rows, 2, 50, 20)
	var s int
	for _, line := range got {
		if strings.HasPrefix(line, "S") {
			s++
		}
	}
	if s != 4 {
		t.Errorf("short panel got %d rows, want the 4 it asked for", s)
	}
}

func TestArrangeStackNeverStarvesAPanel(t *testing.T) {
	// Even when the body is too short, each stacked panel gets at least three rows
	// (its two borders and one line of content); the frame then clips the excess.
	var seen []int
	probe := func(letter string) layout.Panel {
		return layout.Panel{Draw: func(w, h int) []string {
			seen = append(seen, h)
			return fill(letter).Draw(w, h)
		}}
	}
	layout.Arrange([]layout.Panel{probe("A"), probe("B"), probe("C")}, layout.Rows, 2, 50, 5)
	for _, h := range seen {
		if h < 3 {
			t.Errorf("panel was drawn %d rows tall, want at least 3", h)
		}
	}
}

func TestSpacingClamp(t *testing.T) {
	got := layout.Spacing{TerminalPage: 99, PageSection: -4, Section: 50, SectionOption: 7}.Clamp()
	want := layout.Spacing{TerminalPage: 6, PageSection: 0, Section: 10, SectionOption: 6}
	if got != want {
		t.Errorf("Clamp = %+v, want %+v", got, want)
	}
	if d := layout.DefaultSpacing(); d != d.Clamp() {
		t.Errorf("the default spacing %+v is outside its own bounds", d)
	}
	if d := layout.DefaultSpacing(); d.Section != 2 {
		t.Errorf("default section gap = %d, want 2", d.Section)
	}
}

func TestInsetIsExactAndClamps(t *testing.T) {
	for cols := 1; cols <= 60; cols++ {
		for rows := 1; rows <= 24; rows++ {
			for _, gap := range []int{0, 1, 6, 50} {
				got := layout.Inset(cols, rows, gap, 20, 5, func(w, h int) []string {
					if w < 1 || h < 1 {
						t.Fatalf("content given a non-positive area %dx%d", w, h)
					}
					return []string{"content"}
				})
				assertExact(t, "Inset", got, cols, rows)
			}
		}
	}
}

func TestInsetMargins(t *testing.T) {
	got := layout.Inset(20, 9, 2, 5, 3, func(w, h int) []string {
		if w != 16 || h != 5 {
			t.Errorf("content area = %dx%d, want 16x5", w, h)
		}
		return []string{"hello"}
	})
	assertExact(t, "Inset", got, 20, 9)
	if strings.TrimSpace(got[0]) != "" || strings.TrimSpace(got[1]) != "" {
		t.Error("the top margin must be blank")
	}
	if !strings.HasPrefix(got[2], "  hello") {
		t.Errorf("row 2 = %q, want it indented by the left margin", got[2])
	}
}

func TestInsetShrinksMarginToKeepContentRoom(t *testing.T) {
	var gotW, gotH int
	layout.Inset(44, 12, 6, 40, 10, func(w, h int) []string {
		gotW, gotH = w, h
		return nil
	})
	if gotW < 40 || gotH < 10 {
		t.Errorf("content got %dx%d, want at least the 40x10 minimum", gotW, gotH)
	}
	if got := layout.Inset(0, 5, 2, 1, 1, nil); got != nil {
		t.Errorf("zero width = %v, want nil", got)
	}
}
