package dashygotui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/modal"
)

func TestOptionHotkeyRunsFromAnyPage(t *testing.T) {
	f := newFixture(t)
	f.reg.Option(option{Key: "dir", Section: "actions", Number: 5, Label: "Open Storage Dir", Hotkey: "e",
		Run: func(*ctx) tea.Cmd { f.log = append(f.log, "dir"); return nil }})
	d := f.driver(120, 40)

	d.Press("e") // on the Monitor page, where the option is not even shown
	if got := strings.Join(f.log, ","); got != "dir" {
		t.Fatalf("log = %q, want the hotkey to run the option", got)
	}
	d.Press("tab", "tab", "e") // and from another page
	if got := strings.Join(f.log, ","); got != "dir,dir" {
		t.Errorf("log = %q", got)
	}
	d.Press("shift+tab")
	d.RequireContains("Open Storage Dir (e)") // the row shows its key
}

func TestHotkeyDoesNotOpenWhileAModalIsUp(t *testing.T) {
	f := newFixture(t)
	mod := &stubModal{}
	f.reg.Option(option{Key: "ask", Section: "actions", Number: 3, Label: "Ask", Run: func(c *ctx) tea.Cmd { c.OpenModal(mod); return nil }})
	f.reg.Option(option{Key: "dir", Section: "actions", Number: 5, Label: "Dir", Hotkey: "e",
		Run: func(*ctx) tea.Cmd { f.log = append(f.log, "dir"); return nil }})
	d := f.driver(120, 40)
	d.Type("3")
	d.Press("enter", "e")
	if len(f.log) != 0 {
		t.Errorf("the hotkey fired through a modal: %v", f.log)
	}
	if got := strings.Join(mod.keys, ","); got != "e" {
		t.Errorf("the modal should have received the key, got %q", got)
	}
}

func TestHotkeysAreValidated(t *testing.T) {
	run := func(*ctx) tea.Cmd { return nil }
	tests := []struct {
		name     string
		register func(r *reg)
		want     string
	}{
		{"duplicate hotkey", func(r *reg) {
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "e", Run: run})
			r.Option(option{Key: "b", Section: "actions", Number: 2, Label: "B", Hotkey: "e", Run: run})
		}, `option "b" reuses hotkey "e"`},
		{"hotkey then shortcut", func(r *reg) {
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "e", Run: run})
			r.Shortcut(dashygotui.Shortcut[state]{Key: "e", Run: run})
		}, `shortcut "e" is already an option hotkey`},
		{"shortcut then hotkey", func(r *reg) {
			r.Shortcut(dashygotui.Shortcut[state]{Key: "e", Run: run})
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "e", Run: run})
		}, `hotkey "e" is already a shortcut`},
		{"navigation key", func(r *reg) {
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "tab", Run: run})
		}, `hotkey "tab" is reserved`},
		{"vim key", func(r *reg) {
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "k", Run: run})
		}, `hotkey "k" is reserved`},
		{"quit key", func(r *reg) {
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "q", Run: run})
		}, `hotkey "q" is reserved`},
		{"digit", func(r *reg) {
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "5", Run: run})
		}, `hotkey "5" is reserved`},
		{"force-quit key", func(r *reg) {
			r.Option(option{Key: "a", Section: "actions", Number: 1, Label: "A", Hotkey: "ctrl+q", Run: run})
		}, `hotkey "ctrl+q" is reserved`},
		{"pages shortcut on a force key", func(r *reg) {
			r.Shortcut(dashygotui.Shortcut[state]{Key: "ctrl+q", Run: run})
		}, `shortcut "ctrl+q" is reserved`},
		{"everywhere shortcut on a navigation key", func(r *reg) {
			r.Shortcut(dashygotui.Shortcut[state]{Key: "esc", Scope: dashygotui.ScopeEverywhere, Run: run})
		}, `shortcut "esc" is reserved`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := dashygotui.NewRegistry[state]()
			tt.register(r)
			if got := errText(t, baseConfig(r)); !strings.Contains(got, tt.want) {
				t.Errorf("error = %q, want it to mention %q", got, tt.want)
			}
		})
	}
}

func TestEverywhereShortcutMayTakeAForceKey(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Shortcut(dashygotui.Shortcut[state]{Key: "ctrl+q", Scope: dashygotui.ScopeEverywhere, Run: noop})
	if msg := errText(t, baseConfig(r)); msg != "" {
		t.Fatalf("a global shortcut on ctrl+q should be allowed: %s", msg)
	}
}

func TestViewLocalHotkeys(t *testing.T) {
	f := newFixture(t)
	f.reg.View(dashygotui.View[state]{ID: "v", Title: "V",
		Options: func(*ctx) []option {
			return []option{{Number: 1, Label: "Do It", Hotkey: "d", Run: func(*ctx) tea.Cmd { f.log = append(f.log, "view"); return nil }}}
		}})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "v", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("v"); return nil }})
	d := f.driver(120, 40)

	d.Press("d") // not in the view yet
	if len(f.log) != 0 {
		t.Fatalf("a view's hotkey must not work on the pages: %v", f.log)
	}
	d.Press("v")
	d.RequireContains("Do It (d)")
	d.Press("d")
	if got := strings.Join(f.log, ","); got != "view" {
		t.Errorf("log = %q", got)
	}
}

func TestShortcutScopes(t *testing.T) {
	f := newFixture(t)
	mod := &stubModal{}
	f.reg.View(dashygotui.View[state]{ID: "v", Title: "V", Body: func(*ctx, dashygotui.Size) []string { return []string{"body"} }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "ctrl+r", Label: "Refresh", Scope: dashygotui.ScopeEverywhere,
		Run: func(*ctx) tea.Cmd { f.log = append(f.log, "global"); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "x", Label: "Local",
		Run: func(*ctx) tea.Cmd { f.log = append(f.log, "pages"); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "w", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("v"); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "m", Run: func(c *ctx) tea.Cmd { c.OpenModal(mod); return nil }})

	d := f.driver(160, 40)
	d.Press("ctrl+r", "x")
	if got := strings.Join(f.log, ","); got != "global,pages" {
		t.Fatalf("on a page: %q", got)
	}

	f.log = nil
	d.Press("w") // in a view
	d.RequireContains("Refresh")
	d.RequireAbsent("Local")
	d.Press("ctrl+r", "x")
	if got := strings.Join(f.log, ","); got != "global" {
		t.Errorf("in a view: %q, want only the global shortcut", got)
	}

	f.log = nil
	d.Press("esc", "m") // over a modal
	d.Press("ctrl+r", "x")
	if got := strings.Join(f.log, ","); got != "global" {
		t.Errorf("over a modal: %q, want only the global shortcut", got)
	}
	if got := strings.Join(mod.keys, ","); got != "x" {
		t.Errorf("the modal should have received the page shortcut key as plain input, got %q", got)
	}
}

func TestGlobalShortcutReplacesTheDefaultQuit(t *testing.T) {
	f := newFixture(t)
	mod := &stubModal{}
	stops := 0
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "ctrl+q", Label: "Stop", Scope: dashygotui.ScopeEverywhere,
		Run: func(c *ctx) tea.Cmd {
			stops++
			if stops == 1 {
				c.OpenModal(mod)
				return nil
			}
			return c.Quit("stop", "shutting down")
		}})
	d := f.driver(160, 40)
	d.RequireContains("[ctrl+q] Stop")

	d.Press("ctrl+q")
	if d.Quit() {
		t.Fatal("the app's own handler should replace the default quit")
	}
	d.RequireContains("Really do it?") // its handler opened a modal
	d.Press("ctrl+q")                  // pressed again, even over the modal
	if !d.Quit() {
		t.Fatal("the handler should have quit on the second press")
	}
	if got := d.Exit(); got.Reason != "stop" || got.Message != "shutting down" {
		t.Errorf("exit = %+v", got)
	}
}

func TestOnForceQuitReplacesTheDefault(t *testing.T) {
	f := newFixture(t)
	asked := 0
	f.cfg.OnForceQuit = func(c *ctx) tea.Cmd {
		asked++
		c.OpenModal(&modal.Confirm[state]{Title: "Stop and exit?", Lines: []string{"The service stops."}, Accept: "Stop",
			OnAccept: func(c *ctx) tea.Cmd { return c.Quit("stop", "") }})
		return nil
	}
	d := f.driver(120, 40)
	d.Press("ctrl+q")
	if d.Quit() || asked != 1 {
		t.Fatalf("quit=%v asked=%d, want the hook to run instead of quitting", d.Quit(), asked)
	}
	d.RequireContains("Stop and exit?")
	d.Press("ctrl+c") // the other force key reaches the hook too, even over its own modal
	if asked != 2 || d.Quit() {
		t.Fatalf("asked=%d quit=%v", asked, d.Quit())
	}
	d.Press("esc", "q") // an ordinary quit is unaffected
	if !d.Quit() || d.Exit().Reason != "" {
		t.Errorf("q should still quit normally: quit=%v exit=%+v", d.Quit(), d.Exit())
	}
}

func TestForceKeysQuitByDefault(t *testing.T) {
	for _, k := range []string{"ctrl+q", "ctrl+c"} {
		d := newFixture(t).driver(120, 40)
		d.Press(k)
		if !d.Quit() {
			t.Errorf("%s should quit by default", k)
		}
	}
}

func TestExitReasonSurvivesAShortcutQuit(t *testing.T) {
	f := newFixture(t)
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "ctrl+k", Scope: dashygotui.ScopeEverywhere,
		Run: func(c *ctx) tea.Cmd { return c.Quit("kill", "bye") }})
	d := dashtest.New(t, f.cfg, 120, 40)
	d.Press("ctrl+k")
	if got := d.Exit(); got.Reason != "kill" || got.Message != "bye" {
		t.Errorf("exit = %+v", got)
	}
}
