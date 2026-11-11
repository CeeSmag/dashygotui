package modal_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/text"
)

type state struct{ Polls int }

type ctx = dashygotui.Context[state]

// open builds a dashboard whose "o" shortcut opens mod, and presses it.
func open(t *testing.T, mod dashygotui.Modal[state]) *dashtest.Driver {
	t.Helper()
	d := dashtest.New(t, harness(mod), 120, 40)
	d.Press("o")
	return d
}

func harness(mod dashygotui.Modal[state]) dashygotui.Config[state] {
	r := dashygotui.NewRegistry[state]()
	r.Shortcut(dashygotui.Shortcut[state]{Key: "o", Run: func(c *ctx) tea.Cmd { c.OpenModal(mod); return nil }})
	return dashygotui.Config[state]{
		Title:    "T",
		Registry: r,
		Pages:    []dashygotui.Page[state]{{ID: "p", Title: "Page"}},
		Poll: func(context.Context) (state, error) {
			return state{}, nil
		},
	}
}

func TestEdit(t *testing.T) {
	t.Run("submit receives the trimmed value and the modal closes", func(t *testing.T) {
		var got string
		d := open(t, &modal.Edit[state]{
			Title: "Rename", Help: "Pick a new name.", Current: "old-name",
			Submit: func(c *ctx, v string) tea.Cmd { got = v; return nil },
		})
		d.RequireContains("Rename", "Pick a new name.", "Current: old-name", "Enter", "Esc")
		d.Type("  hello  ")
		d.RequireContains("hello")
		d.Press("enter")
		if got != "hello" {
			t.Errorf("Submit got %q, want the trimmed text", got)
		}
		d.RequireAbsent("Rename")
	})
	t.Run("initial value is prefilled", func(t *testing.T) {
		var got string
		d := open(t, &modal.Edit[state]{Title: "E", Initial: "prefilled",
			Submit: func(c *ctx, v string) tea.Cmd { got = v; return nil }})
		d.Press("enter")
		if got != "prefilled" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("esc cancels without submitting", func(t *testing.T) {
		called := false
		d := open(t, &modal.Edit[state]{Title: "E", Submit: func(*ctx, string) tea.Cmd { called = true; return nil }})
		d.Type("abc")
		d.Press("esc")
		d.RequireAbsent("Enter")
		if called {
			t.Error("Submit must not run on Esc")
		}
	})
	t.Run("empty closes with a note", func(t *testing.T) {
		called := false
		d := open(t, &modal.Edit[state]{Title: "E", Submit: func(*ctx, string) tea.Cmd { called = true; return nil }})
		d.Press("enter")
		d.RequireContains("Nothing was changed.")
		if called {
			t.Error("Submit must not run for an empty value")
		}
	})
	t.Run("allow empty passes it through", func(t *testing.T) {
		called := false
		d := open(t, &modal.Edit[state]{Title: "E", AllowEmpty: true, Submit: func(*ctx, string) tea.Cmd { called = true; return nil }})
		d.Press("enter")
		if !called {
			t.Error("Submit should run for an empty value")
		}
	})
	t.Run("validation error shows inline and clears on typing", func(t *testing.T) {
		d := open(t, &modal.Edit[state]{Title: "Port",
			Validate: func(v string) error {
				if v != "80" {
					return errors.New("only port 80\nis allowed")
				}
				return nil
			},
			Submit: func(*ctx, string) tea.Cmd { return nil }})
		d.Type("8080")
		d.Press("enter")
		d.RequireContains("✗ only port 80 is allowed", "Port")
		d.Type("x")
		d.RequireAbsent("✗")
	})
	t.Run("secret input is masked and current is hidden", func(t *testing.T) {
		d := open(t, &modal.Edit[state]{Title: "Password", Secret: true, Current: "hunter2", Submit: func(*ctx, string) tea.Cmd { return nil }})
		d.Type("s3cret")
		d.RequireContains("••••••")
		d.RequireAbsent("s3cret", "hunter2", "Current:")
	})
	t.Run("limit caps the input", func(t *testing.T) {
		var got string
		d := open(t, &modal.Edit[state]{Title: "E", Limit: 3, Submit: func(c *ctx, v string) tea.Cmd { got = v; return nil }})
		d.Type("abcdef")
		d.Press("enter")
		if got != "abc" {
			t.Errorf("got %q, want the input capped at 3", got)
		}
	})
}

func TestEditSave(t *testing.T) {
	t.Run("success closes, flashes and refreshes", func(t *testing.T) {
		var saved string
		polls := 0
		cfg := harness(&modal.Edit[state]{Title: "Save", Pending: "Saving it…",
			Save: func(_ context.Context, v string) (string, error) { saved = v; return "Saved " + v, nil }})
		cfg.Poll = func(context.Context) (state, error) { polls++; return state{Polls: polls}, nil }
		d := dashtest.New(t, cfg, 120, 40)
		before := polls
		d.Press("o")
		d.Type("val")
		d.Press("enter")
		if saved != "val" {
			t.Errorf("saved %q", saved)
		}
		d.RequireAbsent("Enter ")
		d.RequireContains("✓ Saved val")
		if polls != before+1 {
			t.Errorf("polls = %d, want one refresh after saving", polls-before)
		}
	})
	t.Run("failure stays open with the error", func(t *testing.T) {
		d := open(t, &modal.Edit[state]{Title: "Save",
			Save: func(context.Context, string) (string, error) { return "", errors.New("permission denied") }})
		d.Type("val")
		d.Press("enter")
		d.RequireContains("Save", "✗ permission denied")
		d.Type("x") // editing again clears it and allows another try
		d.RequireAbsent("permission denied")
	})
	t.Run("an unrelated task result is left for the other hooks", func(t *testing.T) {
		var other bool
		cfg := harness(&modal.Edit[state]{Title: "Save", Save: func(context.Context, string) (string, error) { return "ok", nil }})
		cfg.Registry.OnResult(func(c *ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
			if res.Tag == "other" {
				other = true
				return true, nil
			}
			return false, nil
		})
		cfg.Registry.OnTick(func(c *ctx, _ time.Time) tea.Cmd {
			return c.RunTask("other", "", func(context.Context) (string, error) { return "", nil })
		})
		d := dashtest.New(t, cfg, 120, 40)
		d.Press("o")
		d.Tick() // the hook starts an unrelated task while the editor is open
		if !other {
			t.Error("the editor must not swallow a result that is not its own")
		}
		d.RequireContains("Save") // and the editor is still open
	})
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		key        string
		accepted   bool
		cancelled  bool
		withCancel bool
	}{
		{"enter", true, false, false},
		{"y", true, false, false},
		{"esc", false, true, true},
		{"n", false, true, true},
		{"q", false, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var accepted, cancelled bool
			c := &modal.Confirm[state]{
				Title: "Delete?", Lines: []string{"This removes everything.", "", "It cannot be undone."}, Accept: "Delete",
				OnAccept: func(*ctx) tea.Cmd { accepted = true; return nil },
			}
			if tt.withCancel {
				c.OnCancel = func(*ctx) tea.Cmd { cancelled = true; return nil }
			}
			d := open(t, c)
			d.RequireContains("Delete?", "This removes everything.", "It cannot be undone.", "Enter Delete")
			d.Press(tt.key)
			d.RequireAbsent("Delete?")
			if accepted != tt.accepted || cancelled != tt.cancelled {
				t.Errorf("accepted=%v cancelled=%v", accepted, cancelled)
			}
		})
	}
	t.Run("labels", func(t *testing.T) {
		d := open(t, &modal.Confirm[state]{Title: "T", Lines: []string{"x"}})
		d.RequireContains("Enter Confirm", "Esc Cancel")
		d2 := open(t, &modal.Confirm[state]{Title: "T", Lines: []string{"x"}, OnCancel: func(*ctx) tea.Cmd { return nil }})
		d2.RequireContains("Esc Back")
		d3 := open(t, &modal.Confirm[state]{Title: "T", Lines: []string{"x"}, Accept: "Back"})
		d3.RequireContains("Enter/Esc Back")
	})
	t.Run("other keys do nothing", func(t *testing.T) {
		d := open(t, &modal.Confirm[state]{Title: "T", Lines: []string{"x"}})
		d.Press("a", "down")
		d.RequireContains("T")
	})
	t.Run("a command is returned", func(t *testing.T) {
		ran := false
		d := open(t, &modal.Confirm[state]{Title: "T", Lines: []string{"x"},
			OnAccept: func(*ctx) tea.Cmd { return func() tea.Msg { ran = true; return nil } }})
		d.Press("y")
		if !ran {
			t.Error("the accept command should have been executed")
		}
	})
}

func TestTyped(t *testing.T) {
	t.Run("the right word confirms", func(t *testing.T) {
		confirmed := false
		d := open(t, &modal.Typed[state]{Title: "Reset", Lines: []string{"Everything goes."}, Word: "RESET", Prompt: "Type RESET to continue",
			OnConfirm: func(*ctx) tea.Cmd { confirmed = true; return nil }})
		d.RequireContains("Reset", "Everything goes.", "Type RESET to continue")
		d.Type("RESET")
		d.Press("enter")
		if !confirmed {
			t.Error("OnConfirm should run")
		}
		d.RequireAbsent("Type RESET")
	})
	t.Run("the wrong word is refused and cleared", func(t *testing.T) {
		confirmed := false
		d := open(t, &modal.Typed[state]{Title: "Reset", Word: "RESET", Prompt: "Type RESET",
			OnConfirm: func(*ctx) tea.Cmd { confirmed = true; return nil }})
		d.Type("reset")
		d.Press("enter")
		d.RequireContains("Type the word RESET exactly")
		if confirmed {
			t.Error("a wrong word must not confirm")
		}
		d.Type("x")
		d.RequireAbsent("Type the word RESET exactly")
	})
	t.Run("empty goes to OnEmpty when given", func(t *testing.T) {
		empty := false
		d := open(t, &modal.Typed[state]{Title: "T", Word: "X", Prompt: "p", OnConfirm: func(*ctx) tea.Cmd { return nil },
			OnEmpty: func(*ctx) tea.Cmd { empty = true; return nil }})
		d.Press("enter")
		if !empty {
			t.Error("OnEmpty should run")
		}
		d.RequireAbsent("Esc Cancel")
	})
	t.Run("empty without OnEmpty is an error", func(t *testing.T) {
		d := open(t, &modal.Typed[state]{Title: "T", Word: "X", Prompt: "p", OnConfirm: func(*ctx) tea.Cmd { return nil }})
		d.Press("enter")
		d.RequireContains("Type the word X exactly")
	})
	t.Run("esc cancels", func(t *testing.T) {
		d := open(t, &modal.Typed[state]{Title: "Title", Word: "X", Prompt: "p"})
		d.Press("esc")
		d.RequireAbsent("Title")
	})
	t.Run("without OnConfirm the right word is still refused", func(t *testing.T) {
		d := open(t, &modal.Typed[state]{Title: "T", Word: "X", Prompt: "p"})
		d.Type("X")
		d.Press("enter")
		d.RequireContains("Type the word X exactly")
	})
}

func TestNotice(t *testing.T) {
	long := make([]string, 60)
	for i := range long {
		long[i] = fmt.Sprintf("line number %02d", i)
	}
	t.Run("static text with a badge", func(t *testing.T) {
		d := open(t, &modal.Notice[state]{Title: "Report", Badge: "[OK]", Lines: []string{"All good.", "", "Nothing to do."}})
		d.RequireContains("Report", "[OK]", "All good.", "Nothing to do.", "to return")
	})
	t.Run("long text scrolls", func(t *testing.T) {
		d := open(t, &modal.Notice[state]{Title: "Log", Lines: long})
		d.RequireContains("line number 00", "scroll", "more")
		d.RequireAbsent("line number 59")
		d.Press("end")
		d.RequireContains("line number 59")
		d.RequireAbsent("line number 00")
		d.Press("home")
		d.RequireContains("line number 00")
		d.Press("pgdown")
		d.RequireAbsent("line number 00")
	})
	t.Run("content is computed per frame", func(t *testing.T) {
		calls := 0
		d := open(t, &modal.Notice[state]{Content: func(c *ctx, inner int) (string, string, []string) {
			calls++
			return "Dynamic", "[live]", []string{fmt.Sprintf("inner=%v", inner > 10)}
		}})
		d.RequireContains("Dynamic", "[live]", "inner=true")
		if calls == 0 {
			t.Error("Content was never called")
		}
	})
	t.Run("lines wrap to the width", func(t *testing.T) {
		d := open(t, &modal.Notice[state]{Title: "Wrap", Width: 40, Lines: []string{strings.Repeat("word ", 30)}})
		for _, line := range d.Lines() {
			if text.Width(line) > 120 {
				t.Fatalf("line wider than the screen")
			}
		}
		d.RequireContains("word word")
	})
	for _, key := range []string{"enter", "esc", "q", "b", "space"} {
		t.Run("closes on "+key, func(t *testing.T) {
			d := open(t, &modal.Notice[state]{Title: "Gone", Lines: []string{"x"}})
			d.Press(key)
			d.RequireAbsent("Gone")
		})
	}
}

func TestChoice(t *testing.T) {
	items := []modal.Item{
		{Label: "Alpha", Detail: "the first"},
		{Label: "Beta", Badge: "[B]", Detail: "the second"},
		{Label: "Gamma", Detail: "the third"},
	}
	t.Run("renders entries with details", func(t *testing.T) {
		d := open(t, &modal.Choice[state]{Title: "Pick", Help: "Choose one.", Items: items})
		d.RequireContains("Pick", "Choose one.", "[1] Alpha", "the first", "[2] Beta [B]", "[3] Gamma", "Enter Select")
		if !strings.Contains(rowOf(d, "Alpha"), "❯") {
			t.Error("the first entry should be highlighted")
		}
	})
	t.Run("cursor starts where it is told", func(t *testing.T) {
		d := open(t, &modal.Choice[state]{Title: "Pick", Items: items, Cursor: 2})
		if !strings.Contains(rowOf(d, "Gamma"), "❯") {
			t.Error("Cursor should select the third entry")
		}
	})
	t.Run("moves wrap and report", func(t *testing.T) {
		var moves []int
		d := open(t, &modal.Choice[state]{Title: "Pick", Items: items, OnMove: func(_ *ctx, i int) { moves = append(moves, i) }})
		d.Press("down", "down", "down", "up", "end", "home", "k", "j")
		want := []int{1, 2, 0, 2, 2, 0, 2, 0}
		if fmt.Sprint(moves) != fmt.Sprint(want) {
			t.Errorf("moves = %v, want %v", moves, want)
		}
	})
	t.Run("digits jump", func(t *testing.T) {
		d := open(t, &modal.Choice[state]{Title: "Pick", Items: items})
		d.Press("3")
		if !strings.Contains(rowOf(d, "Gamma"), "❯") {
			t.Error("3 should jump to the third entry")
		}
		d.Press("9", "0") // out of range: ignored
		if !strings.Contains(rowOf(d, "Gamma"), "❯") {
			t.Error("an out-of-range digit must be ignored")
		}
	})
	t.Run("enter selects", func(t *testing.T) {
		got := -1
		d := open(t, &modal.Choice[state]{Title: "Pick", Items: items, OnSelect: func(_ *ctx, i int) tea.Cmd { got = i; return nil }})
		d.Press("down", "enter")
		if got != 1 {
			t.Errorf("selected %d, want 1", got)
		}
		d.RequireAbsent("Pick")
	})
	t.Run("esc cancels", func(t *testing.T) {
		cancelled := false
		d := open(t, &modal.Choice[state]{Title: "Pick", Items: items, OnCancel: func(*ctx) tea.Cmd { cancelled = true; return nil }})
		d.Press("esc")
		if !cancelled {
			t.Error("OnCancel should run")
		}
		d.RequireAbsent("Pick")
	})
	t.Run("an empty list closes harmlessly", func(t *testing.T) {
		d := open(t, &modal.Choice[state]{Title: "Pick"})
		d.Press("down", "enter")
		d.RequireAbsent("Pick")
	})
	t.Run("a long list is windowed around the cursor", func(t *testing.T) {
		many := make([]modal.Item, 30)
		for i := range many {
			many[i] = modal.Item{Label: fmt.Sprintf("Entry%02d", i+1), Detail: "d"}
		}
		cfg := harness(&modal.Choice[state]{Title: "Many", Items: many})
		d := dashtest.New(t, cfg, 120, 20)
		d.Press("o")
		d.RequireContains("Entry01", "[30 choices]")
		d.RequireAbsent("Entry30")
		d.Press("end")
		d.RequireContains("Entry30")
		d.RequireAbsent("Entry01")
	})
}

func rowOf(d *dashtest.Driver, s string) string {
	for _, l := range d.Lines() {
		if strings.Contains(l, s) {
			return l
		}
	}
	return ""
}

func TestSlider(t *testing.T) {
	t.Run("adjusts within bounds and reports live", func(t *testing.T) {
		var changes []int
		d := open(t, &modal.Slider[state]{Title: "Gap", Help: "How far apart.", Unit: "cells", Min: 0, Max: 4, Value: 2,
			OnChange: func(_ *ctx, v int) { changes = append(changes, v) }})
		d.RequireContains("Gap", "How far apart.", "2 cells", "0 (min)", "4 (max)", "Enter Keep")
		d.Press("right", "right", "right", "left", "left", "left", "left", "left", "home", "end", "down", "up", "-", "+")
		want := []int{3, 4, 3, 2, 1, 0, 4, 3, 4, 3, 4}
		if fmt.Sprint(changes) != fmt.Sprint(want) {
			t.Errorf("changes = %v, want %v", changes, want)
		}
	})
	t.Run("enter keeps the value", func(t *testing.T) {
		got := -1
		d := open(t, &modal.Slider[state]{Title: "Gap", Min: 0, Max: 4, Value: 1,
			OnDone: func(_ *ctx, v int) tea.Cmd { got = v; return nil }})
		d.Press("right", "enter")
		if got != 2 {
			t.Errorf("OnDone got %d, want 2", got)
		}
		d.RequireAbsent("Gap")
	})
	t.Run("esc restores the original", func(t *testing.T) {
		var changes []int
		cancelled := false
		d := open(t, &modal.Slider[state]{Title: "Gap", Min: 0, Max: 4, Value: 1,
			OnChange: func(_ *ctx, v int) { changes = append(changes, v) },
			OnCancel: func(*ctx) tea.Cmd { cancelled = true; return nil }})
		d.Press("right", "right", "esc")
		if fmt.Sprint(changes) != "[2 3 1]" {
			t.Errorf("changes = %v, want the original restored last", changes)
		}
		if !cancelled {
			t.Error("OnCancel should run")
		}
	})
	t.Run("esc without changes does not call OnChange", func(t *testing.T) {
		called := false
		d := open(t, &modal.Slider[state]{Title: "Gap", Min: 0, Max: 4, Value: 1, OnChange: func(*ctx, int) { called = true }})
		d.Press("esc")
		if called {
			t.Error("nothing changed, so nothing to restore")
		}
	})
	t.Run("a degenerate range is repaired", func(t *testing.T) {
		d := open(t, &modal.Slider[state]{Title: "Odd", Min: 5, Max: 2, Value: 99})
		d.RequireContains("Odd", "5")
		d.Press("right", "left", "enter")
	})
}

func TestEveryModalRendersAtEverySize(t *testing.T) {
	long := make([]string, 40)
	for i := range long {
		long[i] = strings.Repeat("text ", 12)
	}
	mods := map[string]func() dashygotui.Modal[state]{
		"edit": func() dashygotui.Modal[state] {
			return &modal.Edit[state]{Title: "Edit", Help: strings.Repeat("help ", 30), Current: strings.Repeat("c", 80)}
		},
		"secret": func() dashygotui.Modal[state] { return &modal.Edit[state]{Title: "Secret", Secret: true} },
		"confirm": func() dashygotui.Modal[state] {
			return &modal.Confirm[state]{Title: "Confirm", Lines: long[:6], OnCancel: func(*ctx) tea.Cmd { return nil }}
		},
		"typed": func() dashygotui.Modal[state] {
			return &modal.Typed[state]{Title: "Typed", Lines: long[:6], Word: "WORD", Prompt: strings.Repeat("p", 90)}
		},
		"notice": func() dashygotui.Modal[state] {
			return &modal.Notice[state]{Title: "Notice", Badge: "[b]", Lines: long}
		},
		"choice": func() dashygotui.Modal[state] {
			items := make([]modal.Item, 12)
			for i := range items {
				items[i] = modal.Item{Label: "Entry", Badge: "[x]", Detail: strings.Repeat("d", 100)}
			}
			return &modal.Choice[state]{Title: "Choice", Help: strings.Repeat("h ", 40), Items: items}
		},
		"slider": func() dashygotui.Modal[state] {
			return &modal.Slider[state]{Title: "Slider", Help: strings.Repeat("h ", 40), Unit: "cells", Min: 0, Max: 10, Value: 3}
		},
	}
	sizes := [][2]int{{1, 1}, {5, 3}, {30, 8}, {39, 9}, {40, 10}, {41, 11}, {60, 12}, {80, 24}, {120, 40}, {200, 60}}
	for name, mk := range mods {
		for _, sz := range sizes {
			t.Run(fmt.Sprintf("%s_%dx%d", name, sz[0], sz[1]), func(t *testing.T) {
				d := dashtest.New(t, harness(mk()), 120, 40)
				d.Press("o")
				d.Resize(sz[0], sz[1])
				lines := strings.Split(d.Raw(), "\n")
				if len(lines) != sz[1] {
					t.Fatalf("%d rows, want %d", len(lines), sz[1])
				}
				for i, line := range lines {
					if w := text.Width(line); w != sz[0] {
						t.Fatalf("row %d is %d cells wide, want %d", i, w, sz[0])
					}
				}
				d.Press("down", "x", "backspace", "esc") // no panics while handling keys at this size
			})
		}
	}
}
