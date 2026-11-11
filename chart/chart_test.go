package chart_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

func themes() map[string]*theme.Theme {
	return map[string]*theme.Theme{
		"unicode": theme.New(theme.Default(), false),
		"ascii":   theme.New(theme.Console(), true),
	}
}

func TestMeterWidthAndFill(t *testing.T) {
	for name, th := range themes() {
		t.Run(name, func(t *testing.T) {
			for w := 1; w <= 40; w++ {
				for _, pct := range []float64{math.NaN(), -10, 0, 33.3, 50, 99.9, 100, 250} {
					got := chart.Meter(th, pct, w)
					if text.Width(got) != w+2 {
						t.Fatalf("Meter(%v, %d) width = %d, want %d", pct, w, text.Width(got), w+2)
					}
				}
			}
			if got := chart.Meter(th, 50, 0); got != "[]" {
				t.Errorf("Meter(w=0) = %q, want []", got)
			}
		})
	}
}

func TestMeterFillCount(t *testing.T) {
	th := theme.New(theme.Default(), false)
	// Filled and empty cells use the same glyph but different colours; count by
	// distinct colour sequences is brittle, so check the extremes differ.
	empty := chart.Meter(th, 0, 10)
	full := chart.Meter(th, 100, 10)
	if empty == full {
		t.Error("an empty meter and a full meter render identically")
	}
	if chart.Meter(th, math.NaN(), 10) != empty {
		t.Error("NaN must render as an empty meter")
	}
	if chart.Meter(th, 1e9, 10) != full {
		t.Error("over-range must clamp to a full meter")
	}
}

func TestMeterWidthHelper(t *testing.T) {
	tests := []struct{ inner, want int }{{0, 4}, {20, 4}, {32, 4}, {40, 12}, {88, 60}, {500, 60}}
	for _, tt := range tests {
		if got := chart.MeterWidth(tt.inner); got != tt.want {
			t.Errorf("MeterWidth(%d) = %d, want %d", tt.inner, got, tt.want)
		}
	}
}

func TestSliderFitsItsWidth(t *testing.T) {
	th := theme.New(theme.Default(), false)
	for maxW := 1; maxW <= 80; maxW++ {
		got := chart.Slider(th, 3, 0, 6, maxW)
		if text.Width(got) > maxW && maxW >= 3 {
			t.Errorf("Slider(maxW=%d) is %d wide: %q", maxW, text.Width(got), got)
		}
	}
	t.Run("degenerate range is a lone knob", func(t *testing.T) {
		for _, bounds := range [][2]int{{5, 5}, {6, 2}} {
			if got := chart.Slider(th, 5, bounds[0], bounds[1], 30); text.Width(got) != 3 {
				t.Errorf("Slider(%v) = %q, want a 3-cell knob", bounds, got)
			}
		}
	})
	t.Run("roomy slider shows the labels", func(t *testing.T) {
		got := chart.Slider(th, 2, 0, 6, 80)
		if !strings.Contains(got, "0 (min)") || !strings.Contains(got, "6 (max)") {
			t.Errorf("Slider = %q, want min and max labels", got)
		}
	})
}

func TestChoiceRows(t *testing.T) {
	for name, th := range themes() {
		t.Run(name, func(t *testing.T) {
			for inner := 5; inner <= 60; inner += 5 {
				rows := chart.ChoiceRows(th, 3, "Storage", "[ON]", "Limit how much space is used", inner, true)
				if len(rows) != 2 {
					t.Fatalf("got %d rows, want 2", len(rows))
				}
				for _, r := range rows {
					if text.Width(r) > inner {
						t.Errorf("inner=%d row is %d wide: %q", inner, text.Width(r), r)
					}
				}
			}
			sel := strings.Join(chart.ChoiceRows(th, 1, "Label", "", "detail", 40, true), "")
			if !strings.Contains(sel, th.Pointer()) {
				t.Errorf("selected row lacks the pointer %q", th.Pointer())
			}
			plain := strings.Join(chart.ChoiceRows(th, 1, "Label", "", "detail", 40, false), "")
			if strings.Contains(plain, th.Pointer()) {
				t.Error("unselected row shows the pointer")
			}
		})
	}
}

func TestChoiceRowsShowsBadgeOnlyWhenItFits(t *testing.T) {
	th := theme.New(theme.Default(), false)
	if got := chart.ChoiceRows(th, 1, "L", "[BADGE]", "d", 40, false)[0]; !strings.Contains(got, "[BADGE]") {
		t.Errorf("badge should fit: %q", got)
	}
	if got := chart.ChoiceRows(th, 1, "Label", "[BADGE]", "d", 10, false)[0]; strings.Contains(got, "BADGE") {
		t.Errorf("badge should be dropped when it cannot fit: %q", got)
	}
}

func TestCells(t *testing.T) {
	tests := []struct {
		name    string
		ascii   bool
		l, r    int
		up, dwn rune
	}{
		{"empty", false, 0, 0, ' ', ' '},
		{"empty ascii", true, 0, 0, ' ', ' '},
		{"one dot left", false, 1, 0, 0x2800 + 0x40, 0x2800 + 0x01},
		{"one dot right", false, 0, 1, 0x2800 + 0x80, 0x2800 + 0x08},
		{"full", false, 4, 4, 0x28FF, 0x28FF},
		{"clamped above", false, 9, 9, 0x28FF, 0x28FF},
		{"clamped below", false, -3, 0, ' ', ' '},
		{"ascii full", true, 4, 4, '█', '█'},
		{"ascii light", true, 1, 0, '░', '░'},
		{"ascii medium", true, 2, 2, '▒', '▒'},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chart.UpCell(tt.ascii, tt.l, tt.r); got != tt.up {
				t.Errorf("UpCell = %U, want %U", got, tt.up)
			}
			if got := chart.DownCell(tt.ascii, tt.l, tt.r); got != tt.dwn {
				t.Errorf("DownCell = %U, want %U", got, tt.dwn)
			}
		})
	}
}

func TestClampDots(t *testing.T) {
	for in, want := range map[int]int{-5: 0, 0: 0, 3: 3, 4: 4, 99: 4} {
		if got := chart.ClampDots(in); got != want {
			t.Errorf("ClampDots(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestTotalDots(t *testing.T) {
	tests := []struct {
		name         string
		v, scale     float64
		available, w int
	}{
		{"zero value", 0, 10, 8, 0},
		{"negative", -1, 10, 8, 0},
		{"NaN", math.NaN(), 10, 8, 0},
		{"bad scale", 5, 0, 8, 0},
		{"no room", 5, 10, 0, 0},
		{"trickle lights one dot", 0.0001, 10, 8, 1},
		{"half", 5, 10, 8, 4},
		{"full", 10, 10, 8, 8},
		{"over scale clamps", 99, 10, 8, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chart.TotalDots(tt.v, tt.scale, tt.available); got != tt.w {
				t.Errorf("TotalDots = %d, want %d", got, tt.w)
			}
		})
	}
}

func TestBrailleDimensions(t *testing.T) {
	vals := []float64{0, 1, 5, 10, math.NaN(), 3, 7, 100, 0, 2}
	for name, th := range themes() {
		t.Run(name, func(t *testing.T) {
			for w := 1; w <= 12; w++ {
				for h := 1; h <= 5; h++ {
					got := chart.Braille(th, vals, 10, w, h)
					if len(got) != h {
						t.Fatalf("Braille(w=%d,h=%d) has %d rows", w, h, len(got))
					}
					for _, line := range got {
						if text.Width(line) != w {
							t.Fatalf("Braille(w=%d,h=%d) row width %d", w, h, text.Width(line))
						}
					}
				}
			}
			if chart.Braille(th, vals, 10, 0, 3) != nil || chart.Braille(th, vals, 10, 3, 0) != nil {
				t.Error("non-positive size must give nil")
			}
		})
	}
}

func TestBrailleRightAlignsAndShowsRecentData(t *testing.T) {
	th := theme.New(theme.Default(), false)
	// A single non-zero sample at the very end lights the right edge only.
	got := chart.Braille(th, []float64{10}, 10, 4, 1)[0]
	plain := stripANSI(got)
	if strings.TrimRight(plain, " ") == "" {
		t.Fatalf("graph is empty: %q", plain)
	}
	if !strings.HasPrefix(plain, "   ") {
		t.Errorf("one sample should sit at the right edge, got %q", plain)
	}
}

func stripANSI(s string) string {
	var sb strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			in = true
		case in && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			in = false
		case !in:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func TestSeries(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := chart.NewSeries(10 * time.Second)
	for i := range 30 {
		s.Add(t0.Add(time.Duration(i)*time.Second), float64(i))
	}
	got := s.Samples()
	if len(got) != 11 { // the last 10 s inclusive of both ends
		t.Fatalf("retained %d samples, want 11", len(got))
	}
	if got[0].V != 19 || got[len(got)-1].V != 29 {
		t.Errorf("range = %v..%v, want 19..29", got[0].V, got[len(got)-1].V)
	}
	if s.Len() != 11 {
		t.Errorf("Len = %d", s.Len())
	}
	got[0].V = -1 // mutating the copy must not affect the series
	if s.Samples()[0].V == -1 {
		t.Error("Samples must return a copy")
	}
}

func TestResample(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := t0.Add(10 * time.Second)
	window := 10 * time.Second

	t.Run("empty window is NaN", func(t *testing.T) {
		for _, v := range chart.Resample(nil, end, window, chart.DefaultHold, 5) {
			if !math.IsNaN(v) {
				t.Errorf("got %v, want NaN", v)
			}
		}
	})
	t.Run("non-positive n", func(t *testing.T) {
		if got := chart.Resample(nil, end, window, chart.DefaultHold, 0); len(got) != 0 {
			t.Errorf("got %v", got)
		}
	})
	t.Run("bucket keeps its maximum", func(t *testing.T) {
		samples := []chart.Sample{{T: t0.Add(time.Second), V: 3}, {T: t0.Add(1500 * time.Millisecond), V: 9}}
		if got := chart.Resample(samples, end, window, chart.DefaultHold, 5)[0]; got != 9 {
			t.Errorf("first bucket = %v, want 9", got)
		}
	})
	t.Run("holds then goes NaN", func(t *testing.T) {
		samples := []chart.Sample{{T: t0.Add(time.Second), V: 4}}
		got := chart.Resample(samples, end, window, 2*time.Second, 10) // 1 s buckets
		if got[0] != 4 {
			t.Errorf("bucket 0 = %v, want 4", got[0])
		}
		if got[2] != 4 {
			t.Errorf("bucket 2 should still hold the value, got %v", got[2])
		}
		if !math.IsNaN(got[8]) {
			t.Errorf("bucket 8 should have gone NaN, got %v", got[8])
		}
	})
}

func TestSampleHistory(t *testing.T) {
	tests := []struct {
		name  string
		hist  []float64
		count int
		want  []float64
	}{
		{"newest kept", []float64{1, 2, 3, 4}, 2, []float64{3, 4}},
		{"padded left", []float64{1, 2}, 4, []float64{0, 0, 1, 2}},
		{"exact", []float64{1, 2}, 2, []float64{1, 2}},
		{"empty", nil, 3, []float64{0, 0, 0}},
		{"zero count", []float64{1}, 0, []float64{}},
		{"negative count", []float64{1}, -2, []float64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chart.SampleHistory(tt.hist, tt.count)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestGraphScale(t *testing.T) {
	tests := []struct {
		name  string
		vals  []float64
		floor float64
		want  float64
	}{
		{"empty uses floor", nil, 100, 100},
		{"below floor", []float64{5, 10}, 100, 100},
		{"rounds up to 1-2-5", []float64{130, 40}, 100, 200},
		{"rounds to 5", []float64{300}, 100, 500},
		{"rounds to 10", []float64{700}, 100, 1000},
		{"ignores NaN", []float64{math.NaN(), 120}, 100, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chart.GraphScale(tt.vals, tt.floor); got != tt.want {
				t.Errorf("GraphScale = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGraphRows(t *testing.T) {
	tests := []struct{ w, spare, want int }{
		{100, 1, 0}, {100, 0, 0}, {10, 8, 2}, {60, 8, 5}, {200, 20, 10}, {200, 3, 3},
	}
	for _, tt := range tests {
		if got := chart.GraphRows(tt.w, tt.spare); got != tt.want {
			t.Errorf("GraphRows(%d,%d) = %d, want %d", tt.w, tt.spare, got, tt.want)
		}
	}
}

func TestWaveScaler(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tick := func(i int) time.Time { return t0.Add(time.Duration(i) * time.Second) }

	t.Run("starts at its floor", func(t *testing.T) {
		if got := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100}).Scale(); got != 100 {
			t.Errorf("Scale = %v, want 100", got)
		}
	})
	t.Run("defaults the floor", func(t *testing.T) {
		if got := chart.NewWaveScaler(chart.ScalerOpts{}).Scale(); got != 1 {
			t.Errorf("Scale = %v, want 1", got)
		}
	})
	t.Run("upscales at once with headroom", func(t *testing.T) {
		s := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100})
		got := s.Update(1000, tick(0))
		if got < 1300 || got > 1400 {
			t.Errorf("scale = %v, want about 1.3x the peak", got)
		}
	})
	t.Run("deadband does not rescale", func(t *testing.T) {
		s := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100})
		high := s.Update(1000, tick(0))
		for i := 1; i <= 20; i++ {
			if got := s.Update(high*0.5, tick(i)); got != high {
				t.Fatalf("tick %d: scale moved to %v inside the deadband", i, got)
			}
		}
	})
	t.Run("decays after sustained quiet", func(t *testing.T) {
		s := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100})
		high := s.Update(100000, tick(0))
		var got float64
		for i := 1; i <= 12; i++ {
			got = s.Update(10, tick(i))
		}
		if got >= high {
			t.Errorf("scale %v never decayed from %v", got, high)
		}
		if got < 100 {
			t.Errorf("scale %v fell below the floor", got)
		}
	})
	t.Run("respects the ceiling", func(t *testing.T) {
		s := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100, MaxScale: 500})
		if got := s.Update(1e9, tick(0)); got != 500 {
			t.Errorf("scale = %v, want the 500 ceiling", got)
		}
	})
	t.Run("sanitises bad input", func(t *testing.T) {
		s := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100})
		for _, v := range []float64{math.NaN(), -5, math.Inf(-1)} {
			if got := s.Update(v, tick(0)); got != 100 {
				t.Errorf("Update(%v) = %v, want the floor", v, got)
			}
		}
	})
	t.Run("sync follows the dominant stream", func(t *testing.T) {
		a := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100})
		b := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100})
		if a.UpdateSync(500, 9000, tick(0)) != b.Update(9000, tick(0)) {
			t.Error("UpdateSync must equal Update on the larger stream")
		}
	})
	t.Run("reset returns to the floor", func(t *testing.T) {
		s := chart.NewWaveScaler(chart.ScalerOpts{MinScale: 100})
		s.Update(1e6, tick(0))
		s.Reset()
		if got := s.Scale(); got != 100 {
			t.Errorf("Scale after Reset = %v, want 100", got)
		}
	})
}
