// Package chart provides the small data-display primitives of a dashboard:
// meters, sliders, choice rows, Braille graphs, time series and the adaptive
// scaling that keeps a live graph readable.
//
// Functions that colour their output take a [theme.Theme]. Nothing here keeps
// package-level state.
package chart

import (
	"fmt"
	"math"
	"strings"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// Meter draws a bracketed meter w cells wide inside the brackets, so the
// result is w+2 cells. pct is a percentage from 0 to 100; NaN counts as 0.
// Filled cells take their colour from their own position along the bar, which
// is what makes a meter redden as it fills.
func Meter(th *theme.Theme, pct float64, w int) string {
	if w <= 0 {
		return "[]"
	}
	if math.IsNaN(pct) {
		pct = 0
	}
	frac := math.Max(0, math.Min(1, pct/100))
	filled := int(frac*float64(w) + 0.5)
	var sb strings.Builder
	sb.WriteString(th.Border.Render("["))
	for i := range w {
		if i < filled {
			sb.WriteString(th.Gradient(float64(i) / float64(w)).Render(th.G.MeterFill))
			continue
		}
		sb.WriteString(th.Track.Render(th.G.MeterEmpty))
	}
	sb.WriteString(th.Border.Render("]"))
	return sb.String()
}

// MeterWidth is how wide a [Meter] may be in a panel of the given inner width,
// leaving room for a label and a reading beside it.
func MeterWidth(inner int) int {
	return min(max(inner-28, 4), 60)
}

// Slider draws a discrete slider from minVal to maxVal with the knob at val,
// within maxWidth cells. It degrades from "min [track] max" to the bare track
// to a truncated track as the space shrinks.
func Slider(th *theme.Theme, val, minVal, maxVal, maxWidth int) string {
	steps := maxVal - minVal
	if steps <= 0 {
		return th.Border.Render("[") + th.Key.Render(th.G.SliderKnob) + th.Border.Render("]")
	}
	minLabel := fmt.Sprintf("%d (min) ", minVal)
	maxLabel := fmt.Sprintf(" %d (max)", maxVal)
	overhead := text.Width(minLabel) + text.Width(maxLabel) + 3 // brackets + knob
	segW := 3
	for segW > 1 && overhead+steps*segW > maxWidth {
		segW--
	}

	var track strings.Builder
	track.WriteString(th.Border.Render("["))
	for i := minVal; i <= maxVal; i++ {
		switch {
		case i == val:
			track.WriteString(th.Key.Render(th.G.SliderKnob))
		case i < val:
			track.WriteString(th.Green.Render(th.G.SliderFill))
		default:
			track.WriteString(th.Track.Render(th.G.SliderRail))
		}
		if i < maxVal && segW > 1 {
			fill := strings.Repeat(th.G.SliderRail, segW-1)
			if i < val {
				track.WriteString(th.Green.Render(fill))
			} else {
				track.WriteString(th.Track.Render(fill))
			}
		}
	}
	track.WriteString(th.Border.Render("]"))

	full := th.Muted.Render(minLabel) + track.String() + th.Muted.Render(maxLabel)
	if text.Width(full) <= maxWidth {
		return full
	}
	if text.Width(track.String()) <= maxWidth {
		return track.String()
	}
	return th.Trunc(track.String(), maxWidth)
}

// ChoiceRows renders one numbered choice of a pop-up as two rows: the choice
// line ("❯ [n] Label badge") and an indented, muted detail line, both fitted
// to inner cells.
func ChoiceRows(th *theme.Theme, num int, label, badge, detail string, inner int, selected bool) []string {
	mark := "  "
	if selected {
		mark = th.Key.Render(th.Pointer()) + " "
		if !strings.Contains(label, "\x1b") {
			label = th.White.Render(label)
		}
	}
	line := mark + th.Key.Render(fmt.Sprintf("[%d]", num)) + " " + label
	if badge != "" && text.Width(line)+1+text.Width(badge) <= inner {
		line += " " + badge
	}
	return []string{th.Trunc(line, inner), th.Muted.Render(th.Trunc("      "+detail, inner))}
}
