package modal

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/text"
)

// Slider lets the user pick an integer between Min and Max. Left and down
// decrease, right and up increase, Home and End jump to the ends, Enter keeps
// the value and Esc restores the original.
type Slider[S any] struct {
	// Title is shown in the pop-up's top border.
	Title string
	// Help is explanatory text shown above the slider.
	Help string
	// Unit is appended to the value, e.g. "cells".
	Unit string
	// Min, Max and Value are the range and the starting value.
	Min, Max, Value int
	// OnChange is called on every change, so the effect shows live. On Esc it
	// is called once more with the original value. Optional.
	OnChange func(ctx *dashygotui.Context[S], value int)
	// OnDone is called, with the modal already closed, when Enter keeps the
	// value. Optional.
	OnDone func(ctx *dashygotui.Context[S], value int) tea.Cmd
	// OnCancel is called, with the modal already closed, after Esc restored the
	// original value. Optional.
	OnCancel func(ctx *dashygotui.Context[S]) tea.Cmd

	original int
	ready    bool
}

func (s *Slider[S]) ensure() {
	if s.ready {
		return
	}
	s.ready = true
	if s.Max < s.Min {
		s.Max = s.Min
	}
	s.Value = min(max(s.Value, s.Min), s.Max)
	s.original = s.Value
}

func (s *Slider[S]) set(ctx *dashygotui.Context[S], v int) {
	v = min(max(v, s.Min), s.Max)
	if v == s.Value {
		return
	}
	s.Value = v
	if s.OnChange != nil {
		s.OnChange(ctx, v)
	}
}

// HandleKey implements dashygotui.Modal.
func (s *Slider[S]) HandleKey(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) tea.Cmd {
	s.ensure()
	switch strings.ToLower(msg.String()) {
	case "left", "down", "-", "h":
		s.set(ctx, s.Value-1)
	case "right", "up", "+", "=", "l":
		s.set(ctx, s.Value+1)
	case "home":
		s.set(ctx, s.Min)
	case "end":
		s.set(ctx, s.Max)
	case "enter":
		v, done := s.Value, s.OnDone
		ctx.CloseModal()
		if done != nil {
			return done(ctx, v)
		}
	case "esc", "q":
		s.set(ctx, s.original)
		cancel := s.OnCancel
		ctx.CloseModal()
		if cancel != nil {
			return cancel(ctx)
		}
	}
	return nil
}

// Render implements dashygotui.Modal.
func (s *Slider[S]) Render(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
	s.ensure()
	th := ctx.Theme()
	w := width(size.Cols, 66)
	inner := w - 4

	var body []string
	if s.Help != "" {
		body = append(body, text.Wrap(s.Help, inner)...)
		body = append(body, "")
	}
	value := fmt.Sprintf("%d", s.Value)
	if s.Unit != "" {
		value += " " + s.Unit
	}
	body = append(body,
		chart.Slider(th, s.Value, s.Min, s.Max, inner),
		"",
		th.White.Render(th.Trunc(value, inner)),
		"",
		keys(th, th.G.LeftRight, "Adjust", "Enter", "Keep", "Esc", "Restore"),
	)
	return popup(th, s.Title, "", w, size.Rows, body)
}
