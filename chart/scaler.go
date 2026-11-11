package chart

import (
	"math"
	"sync"
	"time"
)

// ScalerOpts configures a [WaveScaler]. Zero values select the defaults.
type ScalerOpts struct {
	MinScale  float64 // floor of the scale, in value units (default 1)
	MaxScale  float64 // optional ceiling; 0 means unbounded
	Headroom  float64 // multiplier above the peak on upscale (default 1.30)
	DownRatio float64 // downscale threshold as a fraction of the scale (default 0.10)
	DownLimit int     // consecutive low samples before downscaling (default 5)
}

// WaveScaler adapts the vertical scale of a live graph with hysteresis, in the
// manner of btop:
//
//  1. Headroom: a peak is scaled by 1.3x on upscale so crests do not hit the
//     border.
//  2. Wide deadband: values between 10% and 100% of the current scale do not
//     rescale, which stops vertical jitter during normal fluctuation.
//  3. Debounced decay: only DownLimit consecutive low samples shrink the scale,
//     to three times the average of the last five.
//
// A WaveScaler is safe for concurrent use.
type WaveScaler struct {
	mu           sync.Mutex
	currentScale float64
	minScale     float64
	maxScale     float64
	headroom     float64
	downRatio    float64
	downLimit    int
	downCount    int
	last5        [5]float64
	sampleCount  int
	lastTick     time.Time
}

// NewWaveScaler returns a WaveScaler starting at its floor.
func NewWaveScaler(o ScalerOpts) *WaveScaler {
	if o.MinScale <= 0 {
		o.MinScale = 1
	}
	if o.Headroom <= 1 {
		o.Headroom = 1.30
	}
	if o.DownRatio <= 0 || o.DownRatio >= 1 {
		o.DownRatio = 0.10
	}
	if o.DownLimit <= 0 {
		o.DownLimit = 5
	}
	return &WaveScaler{
		minScale:     o.MinScale,
		maxScale:     o.MaxScale,
		headroom:     o.Headroom,
		downRatio:    o.DownRatio,
		downLimit:    o.DownLimit,
		currentScale: o.MinScale,
	}
}

// clamp keeps a scale between the floor and the optional ceiling.
func (s *WaveScaler) clamp(v float64) float64 {
	v = max(v, s.minScale)
	if s.maxScale > 0 {
		v = min(v, s.maxScale)
	}
	return v
}

// Update feeds the observed peak and returns the scale to draw with. now may
// be the zero time to skip debouncing.
func (s *WaveScaler) Update(peak float64, now time.Time) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if math.IsNaN(peak) || peak < 0 {
		peak = 0
	}
	if s.currentScale <= 0 {
		s.currentScale = s.minScale
	}

	s.last5[s.sampleCount%5] = peak
	s.sampleCount++
	n := min(s.sampleCount, 5)
	sum := 0.0
	for i := range n {
		sum += s.last5[i]
	}
	avg5 := sum / float64(n)

	// Fast upscale: a burst that exceeds the current scale (with headroom).
	if target := s.clamp(roundScale(peak * s.headroom)); peak*s.headroom > s.currentScale && target > s.currentScale {
		s.currentScale = target
		s.downCount = 0
		if !now.IsZero() {
			s.lastTick = now
		}
		return s.currentScale
	}

	// Debounce downscale checks to at most one per 40 ms.
	if !now.IsZero() && !s.lastTick.IsZero() && now.Sub(s.lastTick) < 40*time.Millisecond {
		return s.currentScale
	}
	if !now.IsZero() {
		s.lastTick = now
	}

	if s.currentScale > s.minScale && peak < s.currentScale*s.downRatio {
		s.downCount++
	} else if s.downCount > 0 {
		s.downCount-- // inside the deadband: decay toward stable
	}

	if s.downCount >= s.downLimit {
		if decay := s.clamp(roundScale(avg5 * 3)); decay < s.currentScale {
			s.currentScale = decay
		}
		s.downCount = 0
	}
	s.currentScale = s.clamp(s.currentScale)
	return s.currentScale
}

// UpdateSync updates the scaler from two concurrent streams (download and
// upload, say). Only the dominant stream drives the scale, so both directions
// share one ceiling.
func (s *WaveScaler) UpdateSync(a, b float64, now time.Time) float64 {
	return s.Update(math.Max(a, b), now)
}

// Scale returns the current scale without changing any state.
func (s *WaveScaler) Scale() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return max(s.currentScale, s.minScale)
}

// Reset returns the scaler to its floor and clears its history.
func (s *WaveScaler) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentScale = s.minScale
	s.downCount = 0
	s.sampleCount = 0
	s.last5 = [5]float64{}
	s.lastTick = time.Time{}
}
