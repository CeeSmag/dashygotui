package chart_test

import (
	"math"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/text"
)

func plain(s string) string { return dashtest.Plain(s) }

func plainRows(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = plain(r)
	}
	return out
}

// awkward is input that should never break a renderer.
func awkward() [][]float64 {
	return [][]float64{
		nil,
		{},
		{0},
		{math.NaN(), math.NaN()},
		{-5, -1, 0},
		{math.Inf(1), math.Inf(-1)},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
		{1e300, 1e-300, 5, math.NaN(), 7},
	}
}

func TestWaveformsHaveExactDimensions(t *testing.T) {
	scales := []float64{-1, 0, 1, 10, 1e9}
	for name, th := range themes() {
		t.Run(name, func(t *testing.T) {
			for _, vals := range awkward() {
				for _, scale := range scales {
					for w := 0; w <= 24; w += 3 {
						if w > 0 {
							if got := text.Width(chart.Sparkline(th, vals, scale, w)); got != w {
								t.Fatalf("Sparkline w=%d is %d wide", w, got)
							}
							if got := text.Width(chart.HeatStrip(th, vals, scale, w)); got != w {
								t.Fatalf("HeatStrip w=%d is %d wide", w, got)
							}
						}
						for h := 0; h <= 7; h++ {
							for kind, rows := range map[string][]string{
								"BarGraph":  chart.BarGraph(th, vals, scale, w, h),
								"LineTrace": chart.LineTrace(th, vals, scale, w, h),
								"DualWave":  chart.DualWave(th, vals, vals, scale, w, h),
							} {
								if w <= 0 || h <= 0 {
									if rows != nil {
										t.Fatalf("%s %dx%d should be nil", kind, w, h)
									}
									continue
								}
								if len(rows) != h {
									t.Fatalf("%s %dx%d has %d rows", kind, w, h, len(rows))
								}
								for i, r := range rows {
									if got := text.Width(r); got != w {
										t.Fatalf("%s %dx%d row %d is %d wide", kind, w, h, i, got)
									}
								}
							}
						}
					}
				}
			}
		})
	}
	if chart.Sparkline(themes()["unicode"], []float64{1}, 1, 0) != "" || chart.HeatStrip(themes()["unicode"], []float64{1}, 1, -2) != "" {
		t.Error("a non-positive width draws nothing")
	}
}

func TestSparklineRampAndAlignment(t *testing.T) {
	th := themes()["unicode"]
	if got := plain(chart.Sparkline(th, []float64{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 9)); got != " ▁▂▃▄▅▆▇█" {
		t.Errorf("ramp = %q", got)
	}
	if got := plain(chart.Sparkline(th, []float64{8, 8}, 8, 5)); got != "   ██" {
		t.Errorf("short history should be right-aligned, got %q", got)
	}
	if got := plain(chart.Sparkline(th, []float64{1, 2, 3, 4, 5, 6, 7, 8}, 8, 3)); got != "▆▇█" {
		t.Errorf("only the newest w values show, got %q", got)
	}
	if got := plain(chart.Sparkline(th, []float64{0.0001}, 8, 1)); got != "▁" {
		t.Errorf("a trickle must stay visible, got %q", got)
	}
	if got := plain(chart.Sparkline(th, []float64{1e9}, 8, 1)); got != "█" {
		t.Errorf("values above scale clamp to full, got %q", got)
	}
	if got := plain(chart.Sparkline(th, []float64{5, math.NaN(), -2, 0}, 8, 4)); got != "▅   " {
		t.Errorf("gaps = %q", got)
	}
	if got := strings.TrimSpace(plain(chart.Sparkline(th, []float64{5}, 0, 4))); got != "" {
		t.Errorf("a non-positive scale draws nothing, got %q", got)
	}
}

func TestSparklineInTTYMode(t *testing.T) {
	th := themes()["ascii"]
	got := plain(chart.Sparkline(th, []float64{0, 1, 2, 3, 4, 5, 6, 7, 8}, 8, 9))
	if got != " .:-=+*#%" {
		t.Errorf("TTY ramp = %q", got)
	}
}

func TestBarGraphHeights(t *testing.T) {
	th := themes()["unicode"]
	rows := plainRows(chart.BarGraph(th, []float64{0, 5, 10, 20}, 10, 4, 2))
	// Columns: gap, half (one full block row), full, over-scale clamped.
	if rows[0] != "  ██" || rows[1] != " ███" {
		t.Errorf("rows = %q", rows)
	}
	// A value between whole rows uses an eighth block on the top row.
	rows = plainRows(chart.BarGraph(th, []float64{6}, 8, 1, 2)) // 6/8 of 16 eighths = 12 -> 1 full + 4/8
	if rows[0] != "▄" || rows[1] != "█" {
		t.Errorf("partial bar = %q", rows)
	}
	// A trickle shows at least the lowest block on the bottom row.
	rows = plainRows(chart.BarGraph(th, []float64{0.00001}, 10, 1, 3))
	if rows[0] != " " || rows[1] != " " || rows[2] != "▁" {
		t.Errorf("trickle = %q", rows)
	}
	// Right-aligned with the newest at the right: two gaps, then 1/3, 2/3 and full.
	if got := plainRows(chart.BarGraph(th, []float64{1, 2, 3}, 3, 5, 1)); got[0] != "  ▃▅█" {
		t.Errorf("alignment = %q", got[0])
	}
}

func TestBarGraphInTTYMode(t *testing.T) {
	rows := plainRows(chart.BarGraph(themes()["ascii"], []float64{10}, 10, 1, 2))
	if rows[0] != "%" || rows[1] != "%" {
		t.Errorf("TTY full bar = %q", rows)
	}
}

func TestLineTraceHorizontalAndSpike(t *testing.T) {
	th := themes()["unicode"]
	// A flat line at half height lands on one braille row.
	flat := make([]float64, 16)
	for i := range flat {
		flat[i] = 5
	}
	rows := plainRows(chart.LineTrace(th, flat, 10, 8, 4))
	lit := 0
	for _, r := range rows {
		if strings.TrimSpace(r) != "" {
			lit++
			if strings.Count(r, " ") != 0 {
				t.Errorf("a flat trace should fill its whole row: %q", r)
			}
		}
	}
	if lit != 1 {
		t.Errorf("a flat line should occupy exactly one text row, got %d:\n%s", lit, strings.Join(rows, "\n"))
	}

	// A spike from the floor to the ceiling is drawn as a vertical stroke.
	spike := make([]float64, 16)
	spike[8] = 10
	rows = plainRows(chart.LineTrace(th, spike, 10, 8, 4))
	touched := 0
	for _, r := range rows {
		if strings.TrimSpace(r) != "" {
			touched++
		}
	}
	if touched != 4 {
		t.Errorf("a full-height spike should touch all 4 rows, got %d:\n%s", touched, strings.Join(rows, "\n"))
	}
}

func TestLineTraceGapBreaksTheLine(t *testing.T) {
	th := themes()["unicode"]
	vals := []float64{5, 5, math.NaN(), math.NaN(), 5, 5}
	row := plainRows(chart.LineTrace(th, vals, 10, 3, 1))[0]
	if !strings.Contains(row, " ") && strings.TrimSpace(row) == "" {
		t.Fatalf("row = %q", row)
	}
	if got := plainRows(chart.LineTrace(th, []float64{math.NaN()}, 10, 3, 2)); strings.TrimSpace(strings.Join(got, "")) != "" {
		t.Errorf("an all-gap trace is blank: %q", got)
	}
}

func TestLineTraceInTTYMode(t *testing.T) {
	rows := plainRows(chart.LineTrace(themes()["ascii"], []float64{0, 5, 10}, 10, 3, 5))
	stars := strings.Count(strings.Join(rows, ""), "*")
	if stars != 3 {
		t.Errorf("TTY trace should have one '*' per value, got %d:\n%s", stars, strings.Join(rows, "\n"))
	}
	for _, r := range strings.Join(rows, "") {
		if r > 0x7e {
			t.Errorf("TTY trace drew %q", r)
		}
	}
	if rows[0][2] != '*' || rows[4][0] != '*' {
		t.Errorf("highest value at the top right, lowest at the bottom left:\n%s", strings.Join(rows, "\n"))
	}
}

func TestDualWaveMirrorsAroundTheAxis(t *testing.T) {
	th := themes()["unicode"]
	up := []float64{10, 10, 10, 10}
	down := []float64{0, 0, 0, 0}
	rows := plainRows(chart.DualWave(th, up, down, 10, 2, 5)) // axis on row 2
	if strings.TrimSpace(rows[0]) == "" || strings.TrimSpace(rows[1]) == "" {
		t.Errorf("the up stream should fill the rows above the axis:\n%s", strings.Join(rows, "\n"))
	}
	if strings.Trim(rows[2], "─") != "" {
		t.Errorf("row 2 should be the axis, got %q", rows[2])
	}
	if strings.TrimSpace(rows[3]) != "" || strings.TrimSpace(rows[4]) != "" {
		t.Errorf("an idle down stream leaves the lower half empty:\n%s", strings.Join(rows, "\n"))
	}

	rows = plainRows(chart.DualWave(th, down, up, 10, 2, 5))
	if strings.TrimSpace(rows[0]) != "" || strings.TrimSpace(rows[1]) != "" {
		t.Errorf("an idle up stream leaves the upper half empty:\n%s", strings.Join(rows, "\n"))
	}
	if strings.TrimSpace(rows[3]) == "" || strings.TrimSpace(rows[4]) == "" {
		t.Errorf("the down stream should fill the rows below the axis:\n%s", strings.Join(rows, "\n"))
	}
}

func TestDualWaveSharesOneScale(t *testing.T) {
	th := themes()["unicode"]
	// With the same scale, a stream at 1/4 reaches only the bottom of its half.
	rows := plainRows(chart.DualWave(th, []float64{2.5, 2.5}, []float64{10, 10}, 10, 1, 5))
	if strings.TrimSpace(rows[0]) != "" {
		t.Errorf("a quarter-scale up stream must not reach the top row:\n%s", strings.Join(rows, "\n"))
	}
	if strings.TrimSpace(rows[3]) == "" || strings.TrimSpace(rows[4]) == "" {
		t.Errorf("a full-scale down stream fills its half:\n%s", strings.Join(rows, "\n"))
	}
}

func TestDualWaveSmallHeights(t *testing.T) {
	th := themes()["unicode"]
	one := plainRows(chart.DualWave(th, []float64{5}, []float64{5}, 10, 4, 1))
	if len(one) != 1 || strings.Trim(one[0], "─") != "" {
		t.Errorf("h=1 draws only the axis, got %q", one)
	}
	two := plainRows(chart.DualWave(th, []float64{10, 10}, []float64{10, 10}, 10, 2, 2))
	if len(two) != 2 || strings.TrimSpace(two[0]) == "" || strings.Trim(two[1], "─") != "" {
		t.Errorf("h=2 has one row above the axis and none below, got %q", two)
	}
}

func TestDualWaveInTTYMode(t *testing.T) {
	rows := plainRows(chart.DualWave(themes()["ascii"], []float64{10, 10}, []float64{10, 10}, 10, 2, 5))
	for _, r := range strings.Join(rows, "") {
		if r > 0x7e && !strings.ContainsRune("░▒▓█", r) {
			t.Errorf("TTY dual wave drew %q", r)
		}
	}
	if strings.Trim(rows[2], "-") != "" {
		t.Errorf("the TTY axis is plain dashes, got %q", rows[2])
	}
}

func TestHeatStrip(t *testing.T) {
	th := themes()["unicode"]
	cold, hot := chart.HeatStrip(th, []float64{1}, 10, 1), chart.HeatStrip(th, []float64{10}, 10, 1)
	if plain(cold) != "█" || plain(hot) != "█" {
		t.Errorf("cells are solid blocks: %q %q", plain(cold), plain(hot))
	}
	if cold == hot {
		t.Error("cold and hot cells must differ in colour")
	}
	gap := chart.HeatStrip(th, []float64{math.NaN(), 0}, 10, 2)
	if plain(gap) != "■■" {
		t.Errorf("gaps draw as empty-meter cells, got %q", plain(gap))
	}
	tty := plain(chart.HeatStrip(themes()["ascii"], []float64{1, 4, 8, 10}, 10, 4))
	if tty != ".=#%" {
		t.Errorf("TTY strip = %q, want weight rising with the value", tty)
	}
	for _, r := range tty {
		if r > 0x7e {
			t.Errorf("TTY heat strip drew %q", r)
		}
	}
}
