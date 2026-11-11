package panel_test

import (
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

func newTheme() *theme.Theme { return theme.New(theme.Default(), false) }

// assertBlock fails unless block is exactly h lines of exactly w cells.
func assertBlock(t *testing.T, what string, block []string, w, h int) {
	t.Helper()
	if len(block) != h {
		t.Fatalf("%s: got %d rows, want %d", what, len(block), h)
	}
	for i, line := range block {
		if got := text.Width(line); got != w {
			t.Fatalf("%s: row %d has width %d, want %d: %q", what, i, got, w, line)
		}
	}
}

func TestPanelIsExactlyTheRequestedSize(t *testing.T) {
	th := newTheme()
	titles := []string{"", "Sys", "A rather long panel title indeed", "日本語のタイトル"}
	badges := []string{"", "● ON", "[a very long badge text]"}
	subtitles := []string{"", "sub", "a very long subtitle that will not fit"}
	body := []string{"first row", panel.Divider("SECTION"), "日本語 row", strings.Repeat("x", 100)}

	for _, title := range titles {
		for _, badge := range badges {
			for _, subtitle := range subtitles {
				for w := 1; w <= 60; w++ {
					for h := 1; h <= 8; h++ {
						got := panel.Panel(th, title, badge, subtitle, w, h, body)
						assertBlock(t, "Panel", got, w, h)
					}
				}
			}
		}
	}
}

func TestPanelNonPositiveSizeIsNil(t *testing.T) {
	th := newTheme()
	for _, sz := range [][2]int{{0, 5}, {5, 0}, {-1, 5}, {5, -1}, {0, 0}} {
		if got := panel.Panel(th, "t", "b", "s", sz[0], sz[1], nil); got != nil {
			t.Errorf("Panel(w=%d,h=%d) = %v, want nil", sz[0], sz[1], got)
		}
	}
}

func TestPanelContent(t *testing.T) {
	th := newTheme()
	got := panel.Panel(th, "System", "ONLINE", "sub", 60, 5, []string{"hello", panel.Divider("NET"), "world"})
	joined := strings.Join(got, "\n")
	for _, want := range []string{"System", "ONLINE", "sub", "hello", "NET", "world", "╭", "╮", "╰", "╯", "├", "┤"} {
		if !strings.Contains(joined, want) {
			t.Errorf("panel is missing %q:\n%s", want, joined)
		}
	}
}

func TestPanelClipsAndPadsBody(t *testing.T) {
	th := newTheme()
	long := make([]string, 20)
	for i := range long {
		long[i] = "row"
	}
	assertBlock(t, "clipped", panel.Panel(th, "t", "", "", 20, 5, long), 20, 5)
	assertBlock(t, "padded", panel.Panel(th, "t", "", "", 20, 9, []string{"one"}), 20, 9)
}

func TestPanelTitleDegradesBeforeBadge(t *testing.T) {
	th := newTheme()
	got := strings.Join(panel.Panel(th, "A very very long title", "BADGE", "", 30, 3, nil), "\n")
	if !strings.Contains(got, "BADGE") {
		t.Errorf("badge was dropped before the title:\n%s", got)
	}
}

func TestHJoin(t *testing.T) {
	got := panel.HJoin(2, []string{"ab", "cd", "ef"}, []string{"1"})
	want := []string{"ab  1", "cd   ", "ef   "}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("HJoin = %q, want %q", got, want)
	}
	if got := panel.HJoin(-3, []string{"a"}, []string{"b"}); got[0] != "ab" {
		t.Errorf("negative gap = %q, want %q", got[0], "ab")
	}
	if got := panel.HJoin(1); len(got) != 0 {
		t.Errorf("no blocks = %v, want empty", got)
	}
}

func TestVStackAndBlank(t *testing.T) {
	got := panel.VStack([]string{"a"}, nil, []string{"b", "c"})
	if strings.Join(got, "") != "abc" {
		t.Errorf("VStack = %q", got)
	}
	assertBlock(t, "Blank", panel.Blank(7, 3), 7, 3)
	if got := panel.Blank(-1, -1); len(got) != 0 {
		t.Errorf("Blank(-1,-1) = %v, want empty", got)
	}
}

func TestFrameIsExact(t *testing.T) {
	in := []string{"short", strings.Repeat("w", 50), "日本語"}
	for cols := 0; cols <= 20; cols++ {
		for rows := 0; rows <= 8; rows++ {
			got := panel.Frame(in, cols, rows)
			if len(got) != rows {
				t.Fatalf("Frame rows = %d, want %d", len(got), rows)
			}
			for _, line := range got {
				if text.Width(line) != cols {
					t.Fatalf("Frame cols = %d, want %d (%q)", text.Width(line), cols, line)
				}
			}
		}
	}
}

func TestClipHeight(t *testing.T) {
	block := []string{"1", "2", "3", "4", "5", "6"}
	tests := []struct {
		name string
		h    int
		want string
	}{
		{"fits", 10, "123456"},
		{"non-positive keeps all", 0, "123456"},
		{"keeps top and last two", 4, "1256"},
		{"two rows is the tail", 2, "56"},
		{"one row is the last", 1, "6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strings.Join(panel.ClipHeight(block, tt.h), ""); got != tt.want {
				t.Errorf("ClipHeight(h=%d) = %q, want %q", tt.h, got, tt.want)
			}
		})
	}
}

func TestOverlayKeepsTheFrameSize(t *testing.T) {
	th := newTheme()
	for cols := 1; cols <= 30; cols++ {
		for rows := 1; rows <= 12; rows++ {
			base := panel.Frame([]string{strings.Repeat("·", cols)}, cols, rows)
			box := panel.Panel(th, "Modal", "", "", 14, 6, []string{"body"})
			assertBlock(t, "Overlay", panel.Overlay(base, box, cols, rows), cols, rows)
		}
	}
}

func TestOverlayCentresTheBox(t *testing.T) {
	base := panel.Frame(nil, 10, 5)
	got := panel.Overlay(base, []string{"XX", "YY"}, 10, 5)
	// 10-2 = 8 -> left 4; 5-2 = 3 -> top 1.
	if got[1] != "    XX    " || got[2] != "    YY    " {
		t.Errorf("overlay not centred: %q", got)
	}
	if empty := panel.Overlay(base, nil, 10, 5); strings.Join(empty, "") != strings.Join(base, "") {
		t.Errorf("empty box must leave base unchanged")
	}
}

func TestTooSmallIsExact(t *testing.T) {
	th := newTheme()
	for cols := 0; cols <= 50; cols++ {
		for rows := 0; rows <= 12; rows++ {
			assertBlock(t, "TooSmall", panel.TooSmall(th, cols, rows, 40, 10), cols, rows)
		}
	}
	joined := strings.Join(panel.TooSmall(th, 20, 5, 40, 10), "\n")
	for _, want := range []string{"Terminal too small", "20x5", "40x10"} {
		if !strings.Contains(joined, want) {
			t.Errorf("notice missing %q:\n%s", want, joined)
		}
	}
}

func TestFitSegments(t *testing.T) {
	th := newTheme()
	segs := []panel.Segment{
		{Text: "App", Prio: 100},
		{Text: "host", Prio: 60},
		{Text: "linux", Prio: 20},
		{Text: "clock", Prio: 80},
	}
	full := panel.FitSegments(th, segs, 1000)
	for _, want := range []string{"App", "host", "linux", "clock"} {
		if !strings.Contains(full, want) {
			t.Errorf("roomy strip lost %q: %q", want, full)
		}
	}
	// Drop order: linux (20), then host (60), then clock (80); App never.
	// "App │ host │ clock" is 18 cells; "App │ clock" is 11.
	for _, tt := range []struct {
		budget     int
		kept, lost []string
	}{
		{18, []string{"App", "host", "clock"}, []string{"linux"}},
		{11, []string{"App", "clock"}, []string{"linux", "host"}},
		{0, []string{"App"}, []string{"linux", "host", "clock"}},
	} {
		got := panel.FitSegments(th, segs, tt.budget)
		for _, k := range tt.kept {
			if !strings.Contains(got, k) {
				t.Errorf("budget %d lost %q: %q", tt.budget, k, got)
			}
		}
		for _, l := range tt.lost {
			if strings.Contains(got, l) {
				t.Errorf("budget %d kept %q: %q", tt.budget, l, got)
			}
		}
	}
}

func TestHeaderIsExactlyWide(t *testing.T) {
	th := newTheme()
	segs := []panel.Segment{{Text: "App", Prio: 100}, {Text: "12:00:00", Prio: 80}}
	for cols := 4; cols <= 80; cols++ {
		if got := text.Width(panel.Header(th, segs, cols)); got != cols {
			t.Fatalf("Header(cols=%d) has width %d", cols, got)
		}
	}
}

func TestFitHintsNeverDropsTheHighestPriority(t *testing.T) {
	th := newTheme()
	hints := []panel.Hint{
		{Key: "↑↓", Label: "Move", Style: th.Key, Prio: 60},
		{Key: "Tab", Label: "Next Page", Style: th.Key, Prio: 70},
		{Key: "q", Label: "Quit", Style: th.Red, Prio: 1000},
	}
	if got := panel.FitHints(th, hints, 1000); !strings.Contains(got, "Move") || !strings.Contains(got, "Quit") {
		t.Errorf("roomy hints = %q", got)
	}
	for budget := 0; budget <= 60; budget++ {
		got := panel.FitHints(th, hints, budget)
		if budget >= text.Width(panel.FitHints(th, hints[2:], 1000)) && !strings.Contains(got, "Quit") {
			t.Errorf("budget %d dropped the quit hint: %q", budget, got)
		}
		if text.Width(got) > budget && strings.Contains(got, "Move") {
			t.Errorf("budget %d overflowed with all hints kept: %q", budget, got)
		}
	}
	if panel.FitHints(th, nil, 10) != "" {
		t.Error("no hints should render nothing")
	}
}

func TestRow(t *testing.T) {
	th := newTheme()
	got := panel.Row(th, "CPU", "12%")
	if text.Width(got) != panel.LabelWidth+1+text.Width("12%") {
		t.Errorf("Row width = %d, want label column + space + value", text.Width(got))
	}
	if !strings.HasSuffix(got, " 12%") || !strings.Contains(got, "CPU") {
		t.Errorf("Row = %q", got)
	}
}
