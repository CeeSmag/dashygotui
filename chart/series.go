package chart

import (
	"math"
	"time"
)

// DefaultHold is how long [Resample] keeps drawing the last known value when
// no sample arrives: one missed collector tick, no more.
const DefaultHold = 3 * time.Second

// Sample is one timestamped measurement.
type Sample struct {
	T time.Time
	V float64
}

// Series keeps timestamped samples for a bounded retention period. It is not
// safe for concurrent use; dashboards touch it only from the update loop.
type Series struct {
	keep    time.Duration
	samples []Sample
}

// NewSeries returns a Series that retains samples for keep.
func NewSeries(keep time.Duration) *Series { return &Series{keep: keep} }

// Add appends a sample taken at t and drops those older than the retention
// period. Samples are expected in chronological order.
func (s *Series) Add(t time.Time, v float64) {
	s.samples = append(s.samples, Sample{T: t, V: v})
	cut := 0
	for cut < len(s.samples) && t.Sub(s.samples[cut].T) > s.keep {
		cut++
	}
	if cut > 0 {
		s.samples = append(s.samples[:0], s.samples[cut:]...)
	}
}

// Samples returns a copy of the stored samples, oldest first.
func (s *Series) Samples() []Sample { return append([]Sample(nil), s.samples...) }

// Len reports how many samples are stored.
func (s *Series) Len() int { return len(s.samples) }

// Resample maps samples (oldest first) onto n evenly spaced buckets covering
// (end-window, end]. Several samples in one bucket collapse to their maximum,
// so a one-second burst survives being squeezed into a narrow graph. A bucket
// no sample fell into holds the last value for up to hold and is NaN after
// that. Use [DefaultHold] when unsure.
func Resample(samples []Sample, end time.Time, window, hold time.Duration, n int) []float64 {
	out := make([]float64, max(n, 0))
	if n <= 0 {
		return out
	}
	start := end.Add(-window)
	step := window / time.Duration(n)
	next, last := 0, -1
	for i := range out {
		bucketEnd := start.Add(step * time.Duration(i+1))
		peak := math.NaN()
		for next < len(samples) && !samples[next].T.After(bucketEnd) {
			if math.IsNaN(peak) || samples[next].V > peak {
				peak = samples[next].V
			}
			last = next
			next++
		}
		switch {
		case !math.IsNaN(peak):
			out[i] = peak
		case last < 0 || bucketEnd.Sub(samples[last].T) > hold:
			out[i] = math.NaN()
		default:
			out[i] = samples[last].V
		}
	}
	return out
}

// SampleHistory returns exactly count values: the newest count entries of
// hist, right-aligned and padded on the left with 0 when hist is shorter.
func SampleHistory(hist []float64, count int) []float64 {
	out := make([]float64, max(count, 0))
	if count <= 0 {
		return out
	}
	if n := len(hist); n >= count {
		copy(out, hist[n-count:])
	} else {
		copy(out[count-n:], hist)
	}
	return out
}

// niceCeil rounds up to the next 1, 2 or 5 times a power of ten, so a graph
// scale changes in recognisable steps instead of following every peak.
func niceCeil(v float64) float64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	pow := math.Pow(10, math.Floor(math.Log10(v)))
	switch m := v / pow; {
	case m <= 1:
		return pow
	case m <= 2:
		return 2 * pow
	case m <= 5:
		return 5 * pow
	default:
		return 10 * pow
	}
}

// roundScale rounds v up to two significant figures (20.8 -> 21, 3250 -> 3300),
// which keeps headroom tight without coarse 1-2-5 decade jumps.
func roundScale(v float64) float64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	unit := math.Pow(10, math.Floor(math.Log10(v))-1)
	if unit <= 0 {
		return v
	}
	return math.Ceil(v/unit) * unit
}

// GraphScale is the top of a traffic graph: the peak of vals (ignoring NaN)
// rounded up to a nice number, never below floor.
func GraphScale(vals []float64, floor float64) float64 {
	peak := 0.0
	for _, v := range vals {
		if !math.IsNaN(v) && v > peak {
			peak = v
		}
	}
	if scale := niceCeil(peak); scale > floor {
		return scale
	}
	return floor
}

// GraphRows is how many rows a graph of width w gets: roughly one row per 12
// columns, clamped to 2..10 and to the rows actually spare. It returns 0 when
// fewer than two rows are spare, meaning the caller should fall back to an
// inline sparkline.
func GraphRows(w, spare int) int {
	if spare < 2 {
		return 0
	}
	return min(min(max(w/12, 2), 10), spare)
}
