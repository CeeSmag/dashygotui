package chart

import (
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// StreamOpts configures a [Stream] or a [DualStream]. Zero values select the
// defaults.
type StreamOpts struct {
	// Capacity is how many samples are kept. Default 240.
	Capacity int
	// Scaler tunes the adaptive scale; see [ScalerOpts].
	Scaler ScalerOpts
	// Format writes a scale for the "≤1.2M/s" tag. Default [text.RateShort].
	Format func(float64) string
	// Now supplies the time the scaler debounces with. Default [time.Now];
	// tests inject their own.
	Now func() time.Time
}

func (o StreamOpts) withDefaults() StreamOpts {
	if o.Capacity <= 0 {
		o.Capacity = 240
	}
	if o.Format == nil {
		o.Format = text.RateShort
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}

// scaleTag is "≤1.2M/s" ("<" on a TTY), the label that says what the top of a
// graph means.
func scaleTag(th *theme.Theme, format func(float64) string, scale float64) string {
	sym := "≤"
	if th.ASCII {
		sym = "<"
	}
	return sym + format(scale)
}

// push appends v to a bounded history, dropping the oldest.
func push(hist []float64, capacity int, v float64) []float64 {
	if len(hist) >= capacity {
		copy(hist, hist[1:])
		hist[len(hist)-1] = v
		return hist
	}
	return append(hist, v)
}

// peak is the largest real value among the newest n of vals.
func peak(vals []float64, n int) float64 {
	if len(vals) > n {
		vals = vals[len(vals)-n:]
	}
	p := 0.0
	for _, v := range vals {
		if !math.IsNaN(v) && v > p {
			p = v
		}
	}
	return p
}

// Stream is a self-contained live graph of one measurement: a bounded history,
// an adaptive scale, and renderers. Push a sample whenever one arrives and
// Render on every frame.
//
// The scale follows the samples that are on screen, not the whole history, so a
// crest still travelling across the graph keeps the ceiling up, and the ceiling
// only comes down after it has scrolled away. A Stream is safe for concurrent
// use.
type Stream struct {
	mu      sync.Mutex
	opts    StreamOpts
	hist    []float64
	visible int // samples on screen at the last render
	scaler  *WaveScaler
}

// NewStream returns an empty Stream.
func NewStream(o StreamOpts) *Stream {
	o = o.withDefaults()
	return &Stream{opts: o, visible: o.Capacity, scaler: NewWaveScaler(o.Scaler)}
}

// Push adds a sample, oldest first, and updates the scale.
func (s *Stream) Push(v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hist = push(s.hist, s.opts.Capacity, v)
	s.updateLocked()
}

func (s *Stream) updateLocked() float64 {
	return s.scaler.Update(peak(s.hist, s.visible), s.opts.Now())
}

// Latest returns the newest sample, or 0 when there is none.
func (s *Stream) Latest() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.hist) == 0 {
		return 0
	}
	return s.hist[len(s.hist)-1]
}

// Values returns a copy of the history, oldest first.
func (s *Stream) Values() []float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.hist)
}

// Scale returns the value that fills the full height of the graph now.
func (s *Stream) Scale() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateLocked()
}

// Reset forgets every sample and returns the scale to its floor.
func (s *Stream) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hist = s.hist[:0]
	s.visible = s.opts.Capacity
	s.scaler.Reset()
}

// Render draws the graph w cells wide and h rows tall, newest sample at the
// right edge. It returns nil when w or h is not positive.
func (s *Stream) Render(th *theme.Theme, w, h int) []string {
	if w <= 0 || h <= 0 {
		return nil
	}
	s.mu.Lock()
	s.visible = 2 * w
	scale := s.updateLocked()
	vals := slices.Clone(s.hist)
	s.mu.Unlock()
	return Braille(th, vals, scale, w, h)
}

// Sparkline draws a one-row graph in brackets, "[⡀⣀⠤⠒⠂]", cells wide inside
// them (cells+2 in all). It returns "" when cells is not positive.
func (s *Stream) Sparkline(th *theme.Theme, cells int) string {
	if cells <= 0 {
		return ""
	}
	rows := s.Render(th, cells, 1)
	return th.Border.Render("[") + rows[0] + th.Border.Render("]")
}

// ScaleTag is the muted label "≤1.2M/s" for the current scale ("<" on a TTY),
// followed by " (note)" when note is not empty.
func (s *Stream) ScaleTag(th *theme.Theme, note string) string {
	tag := scaleTag(th, s.opts.Format, s.Scale())
	if note != "" {
		tag += " (" + note + ")"
	}
	return th.Muted.Render(tag)
}

// DualStream is a live graph of two related measurements, such as download and
// upload or read and write, drawn mirrored around a labelled centre line. They
// share one scale so their heights compare. A DualStream is safe for
// concurrent use.
type DualStream struct {
	mu          sync.Mutex
	opts        StreamOpts
	topLabel    string
	bottomLabel string
	top, bottom []float64
	visible     int
	scaler      *WaveScaler
}

// NewDualStream returns an empty DualStream. topLabel names the stream that
// grows upward ("Read", "RX") and bottomLabel the one that grows downward
// ("Write", "TX"); they appear on the centre line.
func NewDualStream(topLabel, bottomLabel string, o StreamOpts) *DualStream {
	o = o.withDefaults()
	return &DualStream{
		opts: o, topLabel: topLabel, bottomLabel: bottomLabel,
		visible: o.Capacity, scaler: NewWaveScaler(o.Scaler),
	}
}

// Push adds one sample of each stream.
func (d *DualStream) Push(top, bottom float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.top = push(d.top, d.opts.Capacity, top)
	d.bottom = push(d.bottom, d.opts.Capacity, bottom)
	d.updateLocked()
}

func (d *DualStream) updateLocked() float64 {
	return d.scaler.UpdateSync(peak(d.top, d.visible), peak(d.bottom, d.visible), d.opts.Now())
}

// Latest returns the newest sample of each stream, or 0 when there is none.
func (d *DualStream) Latest() (top, bottom float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if n := len(d.top); n > 0 {
		top = d.top[n-1]
	}
	if n := len(d.bottom); n > 0 {
		bottom = d.bottom[n-1]
	}
	return top, bottom
}

// Scale returns the value that fills the full height of either half now.
func (d *DualStream) Scale() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.updateLocked()
}

// Reset forgets every sample and returns the scale to its floor.
func (d *DualStream) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.top, d.bottom = d.top[:0], d.bottom[:0]
	d.visible = d.opts.Capacity
	d.scaler.Reset()
}

// Render draws the mirrored graph w cells wide and h rows tall.
//
//   - h of 3 or more: the top stream, a centre line carrying both labels and the
//     scale, then the bottom stream.
//   - h of 2: the two streams with no centre line.
//   - h of 1: the top stream only.
//
// It returns nil when w or h is not positive.
func (d *DualStream) Render(th *theme.Theme, w, h int) []string {
	if w <= 0 || h <= 0 {
		return nil
	}
	d.mu.Lock()
	d.visible = 2 * w
	scale := d.updateLocked()
	top, bottom := slices.Clone(d.top), slices.Clone(d.bottom)
	d.mu.Unlock()

	switch h {
	case 1:
		return Braille(th, top, scale, w, 1)
	case 2:
		rows := DualWave(th, top, bottom, scale, w, 3) // up, axis, down
		return []string{rows[0], rows[2]}
	}
	rows := DualWave(th, top, bottom, scale, w, h)
	rows[h/2] = d.divider(th, w, scale)
	return rows
}

// divider is the centre line: "▲ Read ──── [≤1.2M/s] ──── ▼ Write", shortened
// to what fits in w cells.
func (d *DualStream) divider(th *theme.Theme, w int, scale float64) string {
	topTag := th.G.Up + " " + d.topLabel
	botTag := th.G.Down + " " + d.bottomLabel
	centre := "[" + scaleTag(th, d.opts.Format, scale) + "]"
	rule := func(n int) string { return th.Border.Render(strings.Repeat(th.G.SliderRail, max(n, 0))) }
	top, bot := th.Green.Render(topTag), th.Cyan.Render(botTag)

	tw, bw, cw := text.Width(topTag), text.Width(botTag), text.Width(centre)
	switch {
	case w >= tw+bw+cw+6:
		rest := w - tw - bw - cw - 4
		return top + " " + rule(rest/2) + " " + th.Muted.Render(centre) + " " + rule(rest-rest/2) + " " + bot
	case w >= tw+bw+4:
		return top + " " + rule(w-tw-bw-2) + " " + bot
	}
	return rule(w)
}
