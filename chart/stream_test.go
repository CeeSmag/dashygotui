package chart_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// clock is a fake clock that moves a second each time it is read, so the
// scaler's 40 ms debounce never skips a decision.
func clock() func() time.Time {
	t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time { t = t.Add(time.Second); return t }
}

func newTh(tty bool) *theme.Theme { return theme.New(theme.Default(), tty) }

func TestStreamKeepsACapacityOfSamples(t *testing.T) {
	s := chart.NewStream(chart.StreamOpts{Capacity: 5, Now: clock()})
	for i := 1; i <= 9; i++ {
		s.Push(float64(i))
	}
	got := s.Values()
	if len(got) != 5 || got[0] != 5 || got[4] != 9 {
		t.Errorf("values = %v, want 5..9", got)
	}
	if s.Latest() != 9 {
		t.Errorf("latest = %v", s.Latest())
	}
	s.Reset()
	if len(s.Values()) != 0 || s.Latest() != 0 {
		t.Error("Reset should empty the stream")
	}
}

func TestStreamScaleRisesWithAPeakAndHasHeadroom(t *testing.T) {
	s := chart.NewStream(chart.StreamOpts{Now: clock(), Scaler: chart.ScalerOpts{MinScale: 100}})
	if s.Scale() != 100 {
		t.Fatalf("an empty stream sits at its floor, got %v", s.Scale())
	}
	s.Push(10_000)
	if sc := s.Scale(); sc < 10_000*1.3-1 {
		t.Errorf("scale = %v, want at least the peak with headroom", sc)
	}
}

func TestStreamScaleHoldsWhileACrestIsStillVisible(t *testing.T) {
	s := chart.NewStream(chart.StreamOpts{Capacity: 100, Now: clock(), Scaler: chart.ScalerOpts{MinScale: 10}})
	s.Push(100_000)
	for range 30 {
		s.Push(1)
	}
	s.Render(newTh(false), 50, 3) // 100 samples on screen: the crest is visible
	high := s.Scale()
	for range 10 {
		s.Scale()
	}
	if s.Scale() < high {
		t.Errorf("the scale fell from %v to %v with the crest still on screen", high, s.Scale())
	}
	// Narrow the view until the crest has scrolled off: the scale may now decay.
	s.Render(newTh(false), 5, 3) // only the newest 10 samples are visible
	for range 40 {
		s.Scale()
	}
	if s.Scale() >= high {
		t.Errorf("the scale should decay once the crest is off screen, still %v", s.Scale())
	}
}

func TestStreamRenderHasExactDimensions(t *testing.T) {
	s := chart.NewStream(chart.StreamOpts{Now: clock()})
	for i := range 50 {
		s.Push(float64(i * 100))
	}
	for _, tty := range []bool{false, true} {
		for _, sz := range [][2]int{{1, 1}, {10, 1}, {30, 4}, {80, 8}} {
			rows := s.Render(newTh(tty), sz[0], sz[1])
			if len(rows) != sz[1] {
				t.Fatalf("%v: %d rows", sz, len(rows))
			}
			for _, r := range rows {
				if w := text.Width(r); w != sz[0] {
					t.Fatalf("%v: row width %d", sz, w)
				}
			}
		}
	}
	if s.Render(newTh(false), 0, 3) != nil || s.Render(newTh(false), 3, 0) != nil {
		t.Error("a zero size renders nothing")
	}
}

func TestStreamSparklineIsBracketed(t *testing.T) {
	s := chart.NewStream(chart.StreamOpts{Now: clock()})
	for i := range 20 {
		s.Push(float64(i + 1))
	}
	got := dashtest.Plain(s.Sparkline(newTh(false), 8))
	if !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") || text.Width(got) != 10 {
		t.Errorf("sparkline = %q, want 8 cells in brackets", got)
	}
	if s.Sparkline(newTh(false), 0) != "" {
		t.Error("zero cells gives nothing")
	}
}

func TestScaleTagNamesTheCeilingAndAnyNote(t *testing.T) {
	s := chart.NewStream(chart.StreamOpts{Now: clock(), Scaler: chart.ScalerOpts{MinScale: 5000}})
	if got := dashtest.Plain(s.ScaleTag(newTh(false), "")); got != "≤5K/s" {
		t.Errorf("tag = %q", got)
	}
	if got := dashtest.Plain(s.ScaleTag(newTh(true), "mesh")); got != "<5K/s (mesh)" {
		t.Errorf("TTY tag = %q (ASCII less-than, with the note)", got)
	}
	custom := chart.NewStream(chart.StreamOpts{Now: clock(), Format: func(v float64) string { return "X" }})
	if got := dashtest.Plain(custom.ScaleTag(newTh(false), "")); got != "≤X" {
		t.Errorf("custom format tag = %q", got)
	}
}

func TestDualStreamSharesOneScaleAndReportsLatest(t *testing.T) {
	d := chart.NewDualStream("RX", "TX", chart.StreamOpts{Now: clock(), Scaler: chart.ScalerOpts{MinScale: 10}})
	d.Push(1_000, 50_000)
	d.Push(2_000, 10)
	top, bottom := d.Latest()
	if top != 2_000 || bottom != 10 {
		t.Errorf("latest = %v, %v", top, bottom)
	}
	if d.Scale() < 50_000 {
		t.Errorf("the shared scale must cover the larger stream, got %v", d.Scale())
	}
	d.Reset()
	if a, b := d.Latest(); a != 0 || b != 0 {
		t.Error("Reset should empty both streams")
	}
}

func TestDualStreamLayoutByHeight(t *testing.T) {
	d := chart.NewDualStream("Read", "Write", chart.StreamOpts{Now: clock()})
	for i := range 80 {
		d.Push(float64(i*1000), float64((80-i)*1000))
	}
	th := newTh(false)
	cases := []struct {
		h       int
		divider bool
	}{{1, false}, {2, false}, {3, true}, {6, true}, {9, true}}
	for _, c := range cases {
		rows := plainRows(d.Render(th, 60, c.h))
		if len(rows) != c.h {
			t.Fatalf("h=%d: %d rows", c.h, len(rows))
		}
		joined := strings.Join(rows, "\n")
		has := strings.Contains(joined, "Read") && strings.Contains(joined, "Write") && strings.Contains(joined, "≤")
		if has != c.divider {
			t.Errorf("h=%d: labelled centre line present = %v, want %v:\n%s", c.h, has, c.divider, joined)
		}
		for _, r := range rows {
			if w := text.Width(r); w != 60 {
				t.Fatalf("h=%d: row width %d", c.h, w)
			}
		}
	}
}

func TestDualStreamCentreLineShortensToFit(t *testing.T) {
	d := chart.NewDualStream("Read", "Write", chart.StreamOpts{Now: clock()})
	d.Push(1, 1)
	th := newTh(false)
	for _, w := range []int{1, 5, 12, 16, 22, 40, 100} {
		rows := plainRows(d.Render(th, w, 5))
		centre := rows[2]
		if text.Width(centre) != w {
			t.Fatalf("w=%d: centre line is %d wide: %q", w, text.Width(centre), centre)
		}
		switch {
		case w >= 40 && !strings.Contains(centre, "≤"):
			t.Errorf("w=%d should show the scale: %q", w, centre)
		case w < 14 && strings.Contains(centre, "Read"):
			t.Errorf("w=%d is too narrow for labels: %q", w, centre)
		}
	}
}

func TestDualStreamOnATTYUsesPlainGlyphs(t *testing.T) {
	d := chart.NewDualStream("Read", "Write", chart.StreamOpts{Now: clock()})
	for i := range 60 {
		d.Push(float64(i*10), float64(i*5))
	}
	for _, row := range plainRows(d.Render(newTh(true), 50, 7)) {
		for _, r := range row {
			if r > 0x7e && !strings.ContainsRune("─│░▒▓█", r) {
				t.Fatalf("TTY mode drew %q in %q", r, row)
			}
		}
	}
}

func TestStreamsAreSafeForConcurrentUse(t *testing.T) {
	s := chart.NewStream(chart.StreamOpts{})
	d := chart.NewDualStream("a", "b", chart.StreamOpts{})
	th := newTh(false)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := range 200 {
				s.Push(float64(i))
				d.Push(float64(i), float64(i))
			}
		}()
		go func() {
			defer wg.Done()
			for range 100 {
				s.Render(th, 20, 3)
				d.Render(th, 20, 5)
				s.Scale()
				d.Scale()
			}
		}()
	}
	wg.Wait()
}
