package stdaddons_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/panel"
	"github.com/CeeSmag/dashygotui/stdaddons"
	"github.com/CeeSmag/dashygotui/store"
)

type state struct{}

type ctx = dashygotui.Context[state]

// harness is a dashboard with all three standard addons; its header shows the live
// palette, spacing and interval so a test can watch them change.
func harness(st dashygotui.Store) dashygotui.Config[state] {
	r := dashygotui.NewRegistry[state]()
	stdaddons.Theme(r, "look", 20)
	stdaddons.Spacing(r, "look", 21)
	stdaddons.RefreshRate(r, "look", 22)
	return dashygotui.Config[state]{
		Title:    "T",
		Registry: r,
		Store:    st,
		Pages:    []dashygotui.Page[state]{{ID: "p", Title: "Page"}},
		Sections: []dashygotui.SectionSpec[state]{{ID: "look", Page: "p", Title: "Look"}},
		Poll:     func(context.Context) (state, error) { return state{}, nil },
		Header: func(c *ctx) []panel.Segment {
			sp := c.Spacing()
			return []panel.Segment{{Prio: 100, Text: fmt.Sprintf("pal=%s gaps=%d.%d.%d.%d every=%s",
				c.PaletteID(), sp.TerminalPage, sp.PageSection, sp.Section, sp.SectionOption, c.RefreshInterval())}}
		},
	}
}

func TestOptionsAppearWithBadges(t *testing.T) {
	d := dashtest.New(t, harness(store.NewMemory()), 140, 40)
	d.RequireContains("[20] Color Theme", "[Teal]", "[21] Section Gap", "[0·0·2·0]", "[22] Refresh Rate", "[1/s (1s)]")
}

func TestThemePicker(t *testing.T) {
	t.Run("lists the palettes", func(t *testing.T) {
		d := dashtest.New(t, harness(store.NewMemory()), 140, 40)
		d.Type("20")
		d.Press("enter")
		d.RequireContains("Color Theme", "Teal", "Catppuccin Mocha", "Dracula", "Nord", "Console (16 colours)", "Soothing pastel", "Enter Select")
	})
	t.Run("moving previews without saving, esc restores", func(t *testing.T) {
		st := store.NewMemory()
		d := dashtest.New(t, harness(st), 140, 40)
		d.Type("20")
		d.Press("enter", "down")
		d.RequireContains("pal=catppuccin")
		d.Press("down")
		d.RequireContains("pal=dracula")
		var id string
		if ok, _ := st.Get("dashygotui.palette", &id); ok {
			t.Errorf("a preview must not be saved, found %q", id)
		}
		d.Press("esc")
		d.RequireContains("pal=default")
		d.RequireAbsent("Enter Select")
	})
	t.Run("enter keeps and saves", func(t *testing.T) {
		st := store.NewMemory()
		d := dashtest.New(t, harness(st), 140, 40)
		d.Type("20")
		d.Press("enter", "down", "down", "down", "enter") // Nord is the fourth palette
		d.RequireContains("pal=nord", "Color theme: Nord", "[Nord]")
		var id string
		if ok, err := st.Get("dashygotui.palette", &id); err != nil || !ok || id != "nord" {
			t.Errorf("stored palette = %q ok=%v err=%v", id, ok, err)
		}
	})
	t.Run("the saved palette is highlighted next time", func(t *testing.T) {
		st := store.NewMemory()
		_ = st.Set("dashygotui.palette", "dracula")
		d := dashtest.New(t, harness(st), 140, 40)
		d.Type("20")
		d.Press("enter")
		for _, l := range d.Lines() {
			if strings.Contains(l, "Dracula") && strings.Contains(l, "[3]") && !strings.Contains(l, "❯") {
				t.Errorf("the saved palette should start highlighted: %q", l)
			}
		}
		d.RequireContains("pal=dracula")
	})
}

func TestSpacingMenu(t *testing.T) {
	openMenu := func(t *testing.T, st dashygotui.Store) *dashtest.Driver {
		d := dashtest.New(t, harness(st), 140, 40)
		d.Type("21")
		d.Press("enter")
		return d
	}
	t.Run("lists the four gaps and a reset", func(t *testing.T) {
		d := openMenu(t, store.NewMemory())
		d.RequireContains("Terminal ↔ Page", "Page ↔ Section", "Section ↔ Section", "Section ↔ Option", "Reset to defaults")
	})
	t.Run("a slider adjusts live and enter saves", func(t *testing.T) {
		st := store.NewMemory()
		d := openMenu(t, st)
		d.Press("down", "down", "enter") // Section ↔ Section, currently 2
		d.RequireContains("Section ↔ Section", "2 cells", "0 (min)", "10 (max)")
		d.Press("right", "right")
		d.RequireContains("gaps=0.0.4.0", "4 cells")
		var sp dashygotui.Spacing
		if ok, _ := st.Get("dashygotui.spacing", &sp); ok {
			t.Errorf("a preview must not be saved: %+v", sp)
		}
		d.Press("enter")
		d.RequireContains("gaps=0.0.4.0", "Choose a gap to adjust.") // back at the menu
		if ok, err := st.Get("dashygotui.spacing", &sp); err != nil || !ok || sp.Section != 4 {
			t.Errorf("stored spacing = %+v ok=%v err=%v", sp, ok, err)
		}
	})
	t.Run("esc on the slider restores and returns to the menu", func(t *testing.T) {
		st := store.NewMemory()
		d := openMenu(t, st)
		d.Press("enter") // Terminal ↔ Page, currently 0
		d.Press("right", "right", "right")
		d.RequireContains("gaps=3.0.2.0")
		d.Press("esc")
		d.RequireContains("gaps=0.0.2.0", "Choose a gap to adjust.")
		var sp dashygotui.Spacing
		if ok, _ := st.Get("dashygotui.spacing", &sp); ok {
			t.Errorf("a cancelled change must not be saved: %+v", sp)
		}
	})
	t.Run("each gap keeps the others", func(t *testing.T) {
		d := openMenu(t, store.NewMemory())
		d.Press("enter", "right", "enter") // terminal gap -> 1
		d.Press("down", "enter", "right", "right", "enter")
		d.RequireContains("gaps=1.2.2.0")
	})
	t.Run("reset restores the defaults", func(t *testing.T) {
		st := store.NewMemory()
		_ = st.Set("dashygotui.spacing", dashygotui.Spacing{TerminalPage: 3, PageSection: 2, Section: 7, SectionOption: 1})
		d := openMenu(t, st)
		d.RequireContains("gaps=3.2.7.1")
		d.Press("end", "enter")
		d.RequireContains("gaps=0.0.2.0", "Spacing reset to defaults")
		var sp dashygotui.Spacing
		_, _ = st.Get("dashygotui.spacing", &sp)
		if sp != dashygotui.DefaultSpacing() {
			t.Errorf("stored = %+v, want the defaults", sp)
		}
	})
}

func TestRefreshRatePicker(t *testing.T) {
	st := store.NewMemory()
	d := dashtest.New(t, harness(st), 140, 40)
	d.Type("22")
	d.Press("enter")
	d.RequireContains("Refresh Rate", "10/s (100ms)", "0.2/s (5s)", "balanced (default)")
	d.Press("down", "enter") // 2s
	d.RequireContains("every=2s", "Refresh rate: 0.5/s (2s)", "LIVE 2s")
	var ms int
	if ok, err := st.Get("dashygotui.interval_ms", &ms); err != nil || !ok || ms != 2000 {
		t.Errorf("stored interval = %d ok=%v err=%v", ms, ok, err)
	}
	if got := d.Screen(); !strings.Contains(got, "[0.5/s (2s)]") {
		t.Errorf("the badge should show the new rate")
	}
}

func TestRefreshRatePickerStartsOnTheNearestRate(t *testing.T) {
	st := store.NewMemory()
	_ = st.Set("dashygotui.interval_ms", 480) // between 250ms and 500ms, nearest 500ms
	d := dashtest.New(t, harness(st), 140, 40)
	d.Type("22")
	d.Press("enter")
	for _, l := range d.Lines() {
		if strings.Contains(l, "2/s (500ms)") && strings.Contains(l, "[3]") && !strings.Contains(l, "❯") {
			t.Errorf("the nearest rate should be highlighted: %q", l)
		}
	}
}

func TestRefreshIntervalsAreFastestFirstAndIndependent(t *testing.T) {
	a := dashygotui.RefreshIntervals()
	if len(a) < 2 {
		t.Fatalf("intervals = %v", a)
	}
	for i := 1; i < len(a); i++ {
		if a[i] <= a[i-1] {
			t.Errorf("intervals must be ascending: %v", a)
		}
	}
	a[0] = time.Hour
	if dashygotui.RefreshIntervals()[0] == time.Hour {
		t.Error("RefreshIntervals must return a fresh slice")
	}
}

func TestRegisteringTwiceIsReported(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	stdaddons.Theme(r, "look", 20)
	stdaddons.Theme(r, "look", 30)
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), `duplicate option key "stdaddons.theme"`) {
		t.Errorf("err = %v", err)
	}
}
