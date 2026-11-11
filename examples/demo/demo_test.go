package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/store"
	"github.com/CeeSmag/dashygotui/text"
)

// TestMain diverts the standard logger, as main does, so the addons' log lines
// land in the Logs view instead of the test output.
func TestMain(m *testing.M) {
	restore := dashboard.Logs.Divert(log.Default())
	code := m.Run()
	restore()
	os.Exit(code)
}

// The demo is the integration test: the real registry, filled by the real
// addons (imported by main.go), driven like a user would.

func newDemo(t *testing.T, st dashygotui.Store) *dashtest.Driver {
	t.Helper()
	return dashtest.New(t, dashboard.Config(st), 140, 40)
}

// requireRows fails unless every "[n] Label" pair is on screen as a menu row.
// Whitespace between the number and the label is not fixed: the number column
// widens when another addon adds a longer number, and a test must not break
// just because someone added a file.
func requireRows(t *testing.T, d *dashtest.Driver, pairs ...string) {
	t.Helper()
	screen := d.Screen()
	for i := 0; i+1 < len(pairs); i += 2 {
		re := regexp.MustCompile(`\[` + regexp.QuoteMeta(pairs[i]) + `\]\s+` + regexp.QuoteMeta(pairs[i+1]))
		if !re.MatchString(screen) {
			t.Fatalf("no row [%s] %s on screen:\n%s", pairs[i], pairs[i+1], screen)
		}
	}
}

func TestTheRegistryIsValid(t *testing.T) {
	if err := dashboard.Registry.Validate(); err != nil {
		t.Fatalf("registration errors:\n%v", err)
	}
	if _, err := dashygotui.New(dashboard.Config(store.NewMemory())); err != nil {
		t.Fatalf("New: %v", err)
	}
}

func TestMonitorPageShowsTheWidgets(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.RequireContains("Demo v0.1", "example.local", "Monitor", "Control", "Theme",
		"System", "CPU", "Memory", "Disk", "Uptime", "Network", "▼", "▲", "Activity", "● LIVE")
}

func TestControlPageShowsEveryOption(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Press("tab")
	d.RequireContains("Quick Actions", "Settings", "[never]", "[50 GB]", "[ON ✓]")
	requireRows(t, d,
		"1", "Run Backup", "2", "Restart Service", "3", "Diagnostics", "4", "Logs",
		"10", "Storage Limit", "11", "Notifications", "90", "Reset All Settings")
}

func TestThemePageHasTheBuiltins(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Press("tab", "tab")
	d.RequireContains("Appearance")
	requireRows(t, d, "20", "Color Theme", "21", "Section Gap", "22", "Refresh Rate", "23", "Display Mode")
}

func TestNetworkGraphFillsAsTimePasses(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	for range 6 {
		d.Advance(time.Second)
	}
	braille := false
	for _, r := range d.Screen() {
		if r >= 0x2801 && r <= 0x28FF {
			braille = true
		}
	}
	if !braille {
		t.Errorf("the network widget drew no graph:\n%s", d.Screen())
	}
	d.RequireContains("requests") // the activity log grows too
}

func TestBackupRunsAndRemembersWhen(t *testing.T) {
	st := store.NewMemory()
	d := newDemo(t, st)
	d.Type("1")
	d.Press("enter")
	d.RequireContains("✓ Backup finished.")
	var last time.Time
	if ok, err := st.Get("backup.last", &last); err != nil || !ok || last.IsZero() {
		t.Errorf("backup time not stored: ok=%v err=%v", ok, err)
	}
	d.RequireAbsent("[never]")
}

func TestRestartAsksFirst(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Type("2")
	d.Press("enter")
	d.RequireContains("Restart the service?", "Enter Restart")
	d.Press("esc")
	d.RequireAbsent("Restart the service?")
	d.RequireAbsent("Service restarted.")

	d.Press("enter", "y")
	d.RequireContains("✓ Service restarted.")
}

func TestStorageLimitValidatesAndPersists(t *testing.T) {
	st := store.NewMemory()
	d := newDemo(t, st)
	d.Type("10")
	d.Press("enter")
	d.RequireContains("Storage Limit", "Current: 50 GB")

	d.Type("abc")
	d.Press("enter")
	d.RequireContains(`✗ "abc" is not a number from 1 to 1000`)

	d.Press("backspace", "backspace", "backspace")
	d.Type("2000")
	d.Press("enter")
	d.RequireContains("✗ ", "not a number from 1 to 1000")

	d.Press("backspace", "backspace", "backspace", "backspace")
	d.Type("75")
	d.Press("enter")
	d.RequireContains("✓ Storage limit set to 75 GB.", "[75 GB]")
	var gb int
	if ok, _ := st.Get("storage.limit_gb", &gb); !ok || gb != 75 {
		t.Errorf("stored limit = %d", gb)
	}
}

func TestNotificationsToggleAndPersistAcrossRuns(t *testing.T) {
	st := store.NewMemory()
	d := newDemo(t, st)
	d.Type("11")
	d.Press("enter")
	d.RequireContains("Notifications off.", "[OFF ✗]")

	again := newDemo(t, st) // a fresh run on the same store
	again.Type("11")
	again.RequireContains("[OFF ✗]")
	again.Press("enter")
	again.RequireContains("Notifications on.", "[ON ✓]")
}

func TestDiagnosticsViewAndSelfTest(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Type("3")
	d.Press("enter")
	d.RequireContains("Diagnostics", "● HEALTHY", "Uptime", "Requests", "DIAG", "[Esc/b] Back")
	requireRows(t, d, "1", "Run Self-Test")
	d.Press("enter")
	d.RequireContains("✓ Self-test passed: 12 of 12 checks.")
	d.Press("esc")
	d.RequireContains("Quick Actions")
}

func TestResetEndsWithAReasonForMainToAct(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Type("90")
	d.Press("enter")
	d.RequireContains("Reset all settings?", "Type RESET to continue")

	d.Type("nope")
	d.Press("enter")
	d.RequireContains("Type the word RESET exactly")
	if d.Quit() {
		t.Fatal("the wrong word must not quit")
	}

	d.Type("RESET")
	d.Press("enter")
	if !d.Quit() {
		t.Fatal("RESET should quit")
	}
	if got := d.Exit(); got.Reason != "reset" || got.Message != "All settings were reset." {
		t.Errorf("exit = %+v", got)
	}
}

func TestThemeAndSpacingSurviveARestart(t *testing.T) {
	st := store.NewMemory()
	d := newDemo(t, st)
	d.Type("20")
	d.Press("enter", "down", "down", "down", "enter") // Nord, the fourth palette
	d.Type("21")
	d.Press("enter", "down", "down", "enter", "right", "right", "enter") // Section gap 4

	again := newDemo(t, st)
	again.Press("tab", "tab")
	again.RequireContains("[Nord]", "[0·0·4·0]")
}

func TestEveryPageFitsEverySize(t *testing.T) {
	sizes := [][2]int{{1, 1}, {20, 6}, {39, 9}, {40, 10}, {60, 14}, {79, 24}, {80, 24}, {100, 30}, {140, 40}, {220, 60}}
	for _, sz := range sizes {
		for page := range 3 {
			t.Run(fmt.Sprintf("page%d_%dx%d", page+1, sz[0], sz[1]), func(t *testing.T) {
				d := newDemo(t, store.NewMemory())
				for range page {
					d.Press("tab")
				}
				d.Advance(time.Second)
				d.Resize(sz[0], sz[1])
				lines := strings.Split(d.Raw(), "\n")
				if len(lines) != sz[1] {
					t.Fatalf("%d rows, want %d", len(lines), sz[1])
				}
				for i, l := range lines {
					if w := text.Width(l); w != sz[0] {
						t.Fatalf("row %d is %d wide, want %d", i, w, sz[0])
					}
				}
			})
		}
	}
}

func TestOptionNumbersAreAllReachable(t *testing.T) {
	// Every number typed on the keyboard must land on its own row.
	for _, n := range []string{"1", "2", "3", "4", "10", "11", "20", "21", "22", "23", "90"} {
		d := newDemo(t, store.NewMemory())
		d.Type(n)
		found := false
		for _, l := range d.Lines() {
			if strings.Contains(l, "❯ ["+n+"]") {
				found = true
			}
		}
		if !found {
			t.Errorf("typing %s did not highlight [%s]:\n%s", n, n, d.Screen())
		}
	}
}

func TestOptionHotkeysWorkFromAnyPage(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Press("s") // on the Monitor page, where Storage Limit is not even shown
	d.RequireContains("Storage Limit", "Current: 50 GB")
	d.Press("esc", "tab")
	d.RequireContains("Storage Limit (s)", "Logs (g)") // the rows show their keys
}

func TestCtrlQAsksBeforeStoppingFromAnywhere(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.RequireContains("[ctrl+q] Stop", "[ctrl+r] Refresh")
	d.Press("ctrl+q")
	if d.Quit() {
		t.Fatal("ctrl+q should ask first")
	}
	d.RequireContains("Stop and exit?")
	d.Press("esc")
	d.RequireAbsent("Stop and exit?")

	d.Type("3")
	d.Press("enter", "ctrl+q") // from inside a view
	d.RequireContains("Stop and exit?")
	d.Press("esc", "esc")
	d.Type("10")
	d.Press("enter", "ctrl+q") // and over another pop-up
	d.RequireContains("Stop and exit?")
	d.Press("enter")
	if !d.Quit() {
		t.Fatal("confirming should stop")
	}
	if got := d.Exit(); got.Reason != "stop" || got.Message != "Service stopped." {
		t.Errorf("exit = %+v", got)
	}
}

func TestCtrlCStillQuitsAtOnce(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Press("ctrl+c")
	if !d.Quit() {
		t.Error("ctrl+c is the default force-quit key")
	}
}

func TestCtrlRRefreshes(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Press("ctrl+r")
	d.RequireContains("Refreshing...")
}

func TestLogsViewShowsWhatTheAddonsLogged(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	d.Type("1")
	d.Press("enter")            // run the backup, which logs a line
	d.Advance(10 * time.Second) // let the result message clear from the footer
	d.Press("g")
	d.RequireContains("Logs", "LIVE", "backup finished", "INFO", "[Esc/b] Back")
	d.Press("esc")

	d.Type("2")
	d.Press("enter", "y") // a restart logs a warning
	d.Press("g")
	d.RequireContains("WARN", "service restarted by the operator")
	lines := d.Lines()
	warn, backup := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "service restarted") {
			warn = i
		}
		if strings.Contains(l, "backup finished") {
			backup = i
		}
	}
	if warn < 0 || backup < 0 || warn > backup {
		t.Errorf("the newer warning should be above the older backup line (rows %d, %d)", warn, backup)
	}
}

func TestWaveformWidgetDrawsAllStyles(t *testing.T) {
	d := newDemo(t, store.NewMemory())
	for range 8 {
		d.Advance(time.Second)
	}
	d.RequireContains("CPU Waveforms", "Spark", "Heat", "Bars", "Trace")
	var sawBlock, sawBraille bool
	for _, r := range d.Screen() {
		if r >= 0x2581 && r <= 0x2588 {
			sawBlock = true
		}
		if r >= 0x2801 && r <= 0x28FF {
			sawBraille = true
		}
	}
	if !sawBlock || !sawBraille {
		t.Errorf("expected block characters and braille dots: block=%v braille=%v\n%s", sawBlock, sawBraille, d.Screen())
	}
}

// ttySafe reports whether r is allowed on the Linux console.
func ttySafe(r rune) bool {
	return (r >= 0x20 && r <= 0x7e) || strings.ContainsRune("─│┌┐└┘├┤░▒▓█\n", r)
}

func TestWholeDemoIsConsoleSafeInTTYMode(t *testing.T) {
	cfg := func() dashygotui.Config[dashboard.State] {
		c := dashboard.Config(store.NewMemory())
		c.Glyphs = dashygotui.GlyphsTTY
		return c
	}
	steps := map[string]func(d *dashtest.Driver){
		"monitor":      func(*dashtest.Driver) {},
		"control":      func(d *dashtest.Driver) { d.Press("tab") },
		"theme":        func(d *dashtest.Driver) { d.Press("tab", "tab") },
		"diagnostics":  func(d *dashtest.Driver) { d.Type("3"); d.Press("enter") },
		"logs":         func(d *dashtest.Driver) { d.Press("g") },
		"restart":      func(d *dashtest.Driver) { d.Type("2"); d.Press("enter") },
		"storage":      func(d *dashtest.Driver) { d.Press("s", "x"); d.Press("enter") },
		"reset":        func(d *dashtest.Driver) { d.Type("90"); d.Press("enter") },
		"stop":         func(d *dashtest.Driver) { d.Press("ctrl+q") },
		"theme picker": func(d *dashtest.Driver) { d.Type("20"); d.Press("enter") },
		"spacing menu": func(d *dashtest.Driver) { d.Type("21"); d.Press("enter") },
		"gap slider":   func(d *dashtest.Driver) { d.Type("21"); d.Press("enter", "enter") },
		"display":      func(d *dashtest.Driver) { d.Type("23"); d.Press("enter") },
		"refresh":      func(d *dashtest.Driver) { d.Type("22"); d.Press("enter") },
		"flash":        func(d *dashtest.Driver) { d.Type("11"); d.Press("enter") },
	}
	for name, step := range steps {
		t.Run(name, func(t *testing.T) {
			d := dashtest.New(t, cfg(), 140, 40)
			for range 8 {
				d.Advance(time.Second) // let the graphs fill
			}
			step(d)
			seen := map[rune]bool{}
			for _, r := range d.Screen() {
				if !ttySafe(r) && !seen[r] {
					seen[r] = true
					t.Errorf("TTY mode drew %q (U+%04X)", r, r)
				}
			}
			if t.Failed() {
				t.Logf("\n%s", d.Screen())
			}
		})
	}
}

func TestDemoSwitchesToTTYAndBack(t *testing.T) {
	st := store.NewMemory()
	d := newDemo(t, st)
	d.Press("tab", "tab")
	d.RequireContains("╭", "[Unicode, auto]")
	d.Type("23")
	d.Press("enter", "end", "enter")
	d.RequireContains("┌", "[TTY]")
	d.RequireAbsent("╭", "❯")
	again := newDemo(t, st) // remembered
	again.Press("tab", "tab")
	again.RequireContains("[TTY]")
}
