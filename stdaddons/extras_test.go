package stdaddons_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/stdaddons"
	"github.com/CeeSmag/dashygotui/store"
)

// extras is a dashboard with the display picker, the log view, and a hotkeyed
// theme picker.
func extras(buf *logs.Buffer, st dashygotui.Store, term string) dashygotui.Config[state] {
	r := dashygotui.NewRegistry[state]()
	stdaddons.Theme(r, "look", 20, stdaddons.Setup{Hotkey: "m"})
	stdaddons.Display(r, "look", 23)
	stdaddons.Logs(r, buf, "look", 24, stdaddons.Setup{Hotkey: "g"})
	return dashygotui.Config[state]{
		Title:    "T",
		Registry: r,
		Store:    st,
		Getenv:   func(k string) string { return map[string]string{"TERM": term}[k] },
		Pages:    []dashygotui.Page[state]{{ID: "p", Title: "Page"}},
		Sections: []dashygotui.SectionSpec[state]{{ID: "look", Page: "p", Title: "Look"}},
		Poll:     func(context.Context) (state, error) { return state{}, nil },
	}
}

func TestSetupGivesAHotkey(t *testing.T) {
	d := dashtest.New(t, extras(logs.NewBuffer(10), store.NewMemory(), "xterm"), 140, 40)
	d.RequireContains("Color Theme (m)", "Logs (g)")
	d.Press("m")
	d.RequireContains("Color Theme", "Move to preview a palette")
}

func TestDisplayModePicker(t *testing.T) {
	st := store.NewMemory()
	d := dashtest.New(t, extras(logs.NewBuffer(10), st, "xterm"), 140, 40)
	d.RequireContains("[Unicode, auto]", "╭")
	d.Type("23")
	d.Press("enter")
	d.RequireContains("Display Mode", "Auto", "Unicode", "TTY", "Linux console")

	d.Press("down", "down") // preview TTY live
	d.RequireContains("┌")
	d.RequireAbsent("╭", "❯")
	var saved string
	if ok, _ := st.Get("dashygotui.glyphs", &saved); ok {
		t.Errorf("a preview must not be saved: %q", saved)
	}

	d.Press("esc") // cancelling restores the original look
	d.RequireContains("╭", "[Unicode, auto]")
	d.RequireAbsent("┌─┐")

	d.Press("enter", "end", "enter") // and choosing TTY keeps it
	d.RequireContains("┌", "[TTY]", "Display mode: tty")
	if ok, err := st.Get("dashygotui.glyphs", &saved); err != nil || !ok || saved != "tty" {
		t.Errorf("stored mode = %q ok=%v err=%v", saved, ok, err)
	}
}

func TestDisplayModeAutoOnTheConsole(t *testing.T) {
	d := dashtest.New(t, extras(logs.NewBuffer(10), store.NewMemory(), "linux"), 140, 40)
	d.RequireContains("[TTY, auto]", "┌")
	d.RequireAbsent("╭")
}

func TestDisplayModePickerStartsOnTheCurrentMode(t *testing.T) {
	st := store.NewMemory()
	_ = st.Set("dashygotui.glyphs", "unicode")
	d := dashtest.New(t, extras(logs.NewBuffer(10), st, "linux"), 140, 40)
	d.RequireContains("[Unicode]") // the saved choice beats detection
	d.Type("23")
	d.Press("enter")
	for _, l := range d.Lines() {
		if strings.Contains(l, "Unicode") && strings.Contains(l, "[2]") && !strings.Contains(l, "❯") {
			t.Errorf("the saved mode should be highlighted: %q", l)
		}
	}
}

func seeded() *logs.Buffer {
	b := logs.NewBuffer(200)
	for i := 1; i <= 40; i++ {
		b.Add(logs.Info, fmt.Sprintf("entry %03d", i))
	}
	b.Add(logs.Error, "disk failure")
	b.Add(logs.Error, "network down")
	return b
}

func TestLogsOptionBadge(t *testing.T) {
	d := dashtest.New(t, extras(seeded(), store.NewMemory(), "xterm"), 140, 40)
	d.RequireContains("[42 entries]", "2 errors")
	d2 := dashtest.New(t, extras(logs.NewBuffer(5), store.NewMemory(), "xterm"), 140, 40)
	d2.RequireContains("[0 entries]")
	d2.RequireAbsent("errors")
}

func TestLogsViewShowsNewestFirstAndScrolls(t *testing.T) {
	buf := seeded()
	d := dashtest.New(t, extras(buf, store.NewMemory(), "xterm"), 140, 20)
	d.Press("g") // the hotkey opens the view
	d.RequireContains("Logs", "LIVE", "LOGS", "network down", "disk failure", "entry 040", "f filter: debug", "[Esc/b] Back")
	lines := d.Lines()
	var order []int
	for i, l := range lines {
		for _, want := range []string{"network down", "disk failure", "entry 040"} {
			if strings.Contains(l, want) {
				order = append(order, i)
			}
		}
	}
	if len(order) != 3 || order[0] >= order[1] || order[1] >= order[2] {
		t.Errorf("entries are not newest first: rows %v\n%s", order, d.Screen())
	}

	d.Press("down", "down", "down")
	d.RequireContains("3 newer")
	d.RequireAbsent("LIVE", "network down")
	d.Press("home")
	d.RequireContains("LIVE", "network down")
	d.Press("end")
	d.RequireContains("entry 001")
	d.RequireAbsent("network down")
}

func TestLogsViewHoldsItsPlaceWhileEntriesArrive(t *testing.T) {
	buf := seeded()
	d := dashtest.New(t, extras(buf, store.NewMemory(), "xterm"), 140, 20)
	// 42 entries: two errors on top, then entry 040 downwards. Five steps down
	// leaves entry 037 at the top, with five entries above it.
	d.Press("g", "down", "down", "down", "down", "down")
	d.RequireContains("entry 037", "5 newer")

	for i := 0; i < 12; i++ {
		buf.Add(logs.Warn, fmt.Sprintf("fresh %02d", i))
	}
	d.Tick() // time passes and the dashboard redraws
	d.RequireContains("entry 037", "17 newer")
	d.RequireAbsent("fresh 11")

	d.Press("home")
	d.RequireContains("fresh 11", "LIVE")
}

func TestLogsViewFilterAndBack(t *testing.T) {
	d := dashtest.New(t, extras(seeded(), store.NewMemory(), "xterm"), 140, 20)
	d.Press("g", "f", "f", "f") // Debug -> Info -> Warn -> Error
	d.RequireContains("f filter: error", "network down", "disk failure")
	d.RequireAbsent("entry 040")
	d.Press("esc")
	d.RequireContains("Color Theme") // back on the page
}

func TestLogsViewCapturesTheStandardLogger(t *testing.T) {
	buf := logs.NewBuffer(50)
	var terminal bytes.Buffer
	logger := log.New(&terminal, "", log.LstdFlags)
	restore := buf.Divert(logger)
	defer restore()

	d := dashtest.New(t, extras(buf, store.NewMemory(), "xterm"), 140, 20)
	logger.Printf("[ERROR] cannot reach %s", "db.internal")
	logger.Print("plain note")
	d.Press("g")
	d.RequireContains("ERROR", "cannot reach db.internal", "plain note", "INFO")
	if terminal.Len() != 0 {
		t.Errorf("the logger wrote to the terminal: %q", terminal.String())
	}
}

func TestLogsViewInTTYMode(t *testing.T) {
	d := dashtest.New(t, extras(seeded(), store.NewMemory(), "linux"), 140, 20)
	d.Press("g")
	d.RequireContains("* LIVE", "Up/Dn scroll")
	for _, r := range d.Screen() {
		if r > 0x7e && !strings.ContainsRune("─│┌┐└┘├┤░▒▓█\n", r) {
			t.Errorf("TTY log view drew %q", r)
		}
	}
}

func TestLogsViewFitsEverySize(t *testing.T) {
	for _, sz := range [][2]int{{40, 10}, {60, 12}, {80, 24}, {140, 40}, {30, 8}, {1, 1}} {
		d := dashtest.New(t, extras(seeded(), store.NewMemory(), "xterm"), 140, 40)
		d.Press("g")
		d.Resize(sz[0], sz[1])
		d.Press("down", "pgdown", "end", "home", "f")
		lines := strings.Split(d.Raw(), "\n")
		if len(lines) != sz[1] {
			t.Fatalf("%dx%d: %d rows", sz[0], sz[1], len(lines))
		}
	}
}
