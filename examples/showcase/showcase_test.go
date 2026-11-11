package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
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

// The showcase is an integration test of the whole library: the real registry,
// filled by the real addons (imported by main.go), driven like a user would.

func opts(st dashygotui.Store) dashboard.Options { return dashboard.Options{Store: st} }

func newApp(t *testing.T, st dashygotui.Store) *dashtest.Driver {
	t.Helper()
	dashboard.Logs.Clear()
	return dashtest.New(t, dashboard.Config(opts(st)), 160, 44)
}

func newAppWith(t *testing.T, o dashboard.Options) *dashtest.Driver {
	t.Helper()
	dashboard.Logs.Clear()
	return dashtest.New(t, dashboard.Config(o), 160, 44)
}

// rows fails unless every "[n] Label" pair is on screen as a menu row.
// Whitespace between the number and the label is not fixed: the number column
// widens when another addon adds a longer number, and a test must not break
// just because someone added a file.
func rows(t *testing.T, d *dashtest.Driver, pairs ...string) {
	t.Helper()
	screen := d.Screen()
	for i := 0; i+1 < len(pairs); i += 2 {
		re := regexp.MustCompile(`\[` + regexp.QuoteMeta(pairs[i]) + `\]\s+` + regexp.QuoteMeta(pairs[i+1]))
		if !re.MatchString(screen) {
			t.Fatalf("no row [%s] %s on screen:\n%s", pairs[i], pairs[i+1], screen)
		}
	}
}

// open jumps to a numbered option and opens it.
func open(d *dashtest.Driver, number string) {
	d.Type(number)
	d.Press("enter")
}

func TestTheRegistryIsValid(t *testing.T) {
	if err := dashboard.Registry.Validate(); err != nil {
		t.Fatalf("registration errors:\n%v", err)
	}
	if _, err := dashygotui.New(dashboard.Config(opts(store.NewMemory()))); err != nil {
		t.Fatalf("New: %v", err)
	}
}

func TestEveryPageShowsWhatItDemonstrates(t *testing.T) {
	d := newApp(t, store.NewMemory())
	for i := 0; i < 8; i++ {
		d.Advance(time.Second) // let the graphs fill
	}
	pages := []struct {
		name string
		want []string
	}{
		{"monitor", []string{"System", "TRAFFIC", "Network", "Activity", "CPU Waveforms", "ONLINE", "LIVE"}},
		{"charts", []string{"Area & Sparkline", "Bars & Heat", "Trace & Mirror", "chart.Braille"}},
		{"actions", []string{"Quick Actions", "[4 actions]", "Settings", "[persisted]", "Danger Zone", "things to do"}},
		{"popups", []string{"Standard Pop-ups", "Custom Pop-ups", "from the modal package"}},
		{"views", []string{"Full-screen Views", "Navigation", "Hooks & Tasks"}},
		{"theme", []string{"Appearance", "Symbols", "Palette", "Solarized Dark", "ALL PALETTES"}},
	}
	for i, p := range pages {
		if i > 0 {
			d.Press("tab")
		}
		d.RequireContains(p.want...)
		t.Logf("page %d (%s) ok", i+1, p.name)
	}
	d.Press("tab") // wraps to the first page
	d.RequireContains("System")
}

func TestEveryOptionIsListedAndReachableByNumber(t *testing.T) {
	numbers := []struct{ n, label string }{
		{"1", "Run Backup"}, {"2", "Restart Service"}, {"3", "Export Report"},
		{"10", "Storage Limit"}, {"11", "Notifications"}, {"12", "API Token"}, {"13", "Region"},
		{"14", "Alert Volume"}, {"15", "Hostname"}, {"90", "Reset All Settings"},
		{"30", "Notice"}, {"31", "Notice (live content)"}, {"32", "Notice (scrolls)"}, {"33", "Confirm"},
		{"34", "Confirm (acknowledge)"}, {"35", "Type to confirm"}, {"36", "Choice"}, {"37", "Slider"},
		{"38", "Edit (with validation)"}, {"40", "Dice Roller"}, {"41", "Ping Host"}, {"42", "New Project"},
		{"50", "Diagnostics"}, {"51", "Processes"}, {"52", "Logs"}, {"53", "Release Notes"}, {"54", "Event Stream"},
		{"60", "Jump to Charts"}, {"61", "Jump to Theme"}, {"62", "Jump to Monitor"},
		{"70", "Simulate Alert"}, {"71", "Simulate Outage"}, {"72", "Run External Command"},
		{"73", "Write Log Entries"},
		{"20", "Color Theme"}, {"21", "Section Gap"}, {"22", "Refresh Rate"}, {"23", "Display Mode"},
		{"24", "Background"}, {"25", "Transparent Background"},
	}
	for _, o := range numbers {
		d := newApp(t, store.NewMemory())
		d.Type(o.n) // typing a number jumps to it, on whatever page it lives
		rows(t, d, o.n, o.label)
		found := false
		for _, l := range d.Lines() {
			if strings.Contains(l, "❯ ["+o.n+"]") {
				found = true
			}
		}
		if !found {
			t.Errorf("typing %s did not highlight [%s] %s", o.n, o.n, o.label)
		}
	}
}

func TestEveryOptionDoesSomethingVisible(t *testing.T) {
	tests := []struct {
		number string
		want   []string
	}{
		{"1", []string{"Backup finished."}},
		{"2", []string{"Restart the service?", "Enter Restart"}},
		{"3", []string{"Report", "Report generated at", "Top processes"}},
		{"10", []string{"Storage Limit", "Current: 50 GB"}},
		{"11", []string{"Notifications off."}},
		{"12", []string{"API Token", "hidden as you type"}},
		{"13", []string{"Region", "us-east", "eu-central"}},
		{"14", []string{"Alert Volume", "60 percent"}},
		{"15", []string{"Hostname", "Current: example.local"}},
		{"90", []string{"Reset all settings?", "Type RESET"}},
		{"30", []string{"About", "read-only card"}},
		{"31", []string{"Live Status", "Uptime"}},
		{"32", []string{"Changelog", "60 entries", "more"}},
		{"33", []string{"Archive old logs?"}},
		{"34", []string{"Heads up", "Enter/Esc Back"}},
		{"35", []string{"Delete the archive?", "Type DELETE"}},
		{"36", []string{"Alert level", "[1] Low", "[3] Urgent"}},
		{"37", []string{"Retention", "14 days"}},
		{"38", []string{"Contact", "Where alerts are sent"}},
		{"40", []string{"Dice", "Space roll", "no rolls yet"}},
		{"41", []string{"Ping", "press Enter to ping"}},
		{"42", []string{"Step 1 of 3", "Create a new project?"}},
		{"50", []string{"Diagnostics", "HEALTHY", "Run Self-Test", "DIAG"}},
		{"51", []string{"Processes", "api-gateway", "PID", "PROCS"}},
		{"52", []string{"Logs", "LIVE", "LOGS"}},
		{"70", []string{"Alert #1"}}, // each dashboard counts its own alerts
		{"73", []string{"Wrote 16 log entries"}},
		{"20", []string{"Color Theme", "Solarized Dark", "Console (16 colours)"}},
		{"21", []string{"Section Gap", "Terminal"}},
		{"22", []string{"Refresh Rate", "balanced (default)"}},
		{"23", []string{"Display Mode", "Linux console"}},
		{"24", []string{"Background", "Terminal", "Paint", "Off (transparent)"}},
		{"25", []string{"Transparent background on", "[ON"}},
	}
	for _, tt := range tests {
		t.Run("option_"+tt.number, func(t *testing.T) {
			d := newApp(t, store.NewMemory())
			open(d, tt.number)
			d.RequireContains(tt.want...)
		})
	}
}

func TestExternalCommandIsHandedToTheTerminal(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "72") // the driver does not run the process; it must simply not break
	d.RequireContains("Run External Command")
	if d.Quit() {
		t.Error("running a command must not quit the dashboard")
	}
}

func TestStorageLimitValidatesAndPersists(t *testing.T) {
	st := store.NewMemory()
	d := newApp(t, st)
	d.Press("s") // the hotkey works from the Monitor page
	d.RequireContains("Storage Limit")
	d.Type("abc")
	d.Press("enter")
	d.RequireContains(`"abc" is not a number from 1 to 1000`)
	d.Press("backspace", "backspace", "backspace")
	d.Type("75")
	d.Press("enter")
	d.RequireContains("Storage limit set to 75 GB.")
	var gb int
	if ok, _ := st.Get("storage.limit_gb", &gb); !ok || gb != 75 {
		t.Errorf("stored limit = %d", gb)
	}
	d.Press("tab", "tab")
	d.RequireContains("[75 GB]", "Storage Limit (s)")
}

func TestSecretTokenIsNeverShown(t *testing.T) {
	st := store.NewMemory()
	d := newApp(t, st)
	open(d, "12")
	d.Type("s3cret-token")
	d.RequireContains("••••")
	d.RequireAbsent("s3cret-token")
	d.Press("enter")
	d.RequireContains("Token accepted.")
	d.RequireAbsent("s3cret-token")
	var n int
	if ok, _ := st.Get("token.length", &n); !ok || n != 12 {
		t.Errorf("only the length should be stored: %d", n)
	}
	d.RequireContains("[12 characters]") // opening the option jumped to its page
}

func TestSettingsSurviveARestart(t *testing.T) {
	st := store.NewMemory()
	d := newApp(t, st)
	open(d, "13")           // Region
	d.Press("down", "down") // eu-west -> eu-central -> ap-south? start is eu-west (index 2)
	d.RequireContains("Highlighted ")
	d.Press("enter")
	open(d, "14") // Volume
	d.Press("right", "right", "right", "enter")
	open(d, "11") // Notifications off
	open(d, "15") // Hostname
	d.Press("backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace")
	d.Type("db1")
	d.Press("enter")

	again := newApp(t, st)
	again.Press("tab", "tab")
	again.RequireContains("[ap-south]", "[63%]", "[OFF ✗]", "[db1]")
}

func TestHostnameEmptyMeansDefault(t *testing.T) {
	st := store.NewMemory()
	_ = st.Set("hostname", "custom")
	d := newApp(t, st)
	open(d, "15")
	d.Press("backspace", "backspace", "backspace", "backspace", "backspace", "backspace") // "custom"
	d.Press("enter")
	d.RequireContains("Hostname set to example.local.")
	open(d, "15")
	d.Press("backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace",
		"backspace", "backspace", "backspace", "backspace", "backspace", "backspace") // "example.local"
	d.Type("a b")
	d.Press("enter")
	d.RequireContains("cannot contain spaces")
}

func TestConfirmAndTypedCallbacks(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "33")
	d.Press("esc")
	d.RequireContains("Left as it was")
	open(d, "33")
	d.Press("y")
	d.RequireContains("Archived")

	open(d, "35") // typed, with OnEmpty
	d.Press("enter")
	d.RequireContains("Nothing deleted")
	open(d, "35")
	d.Type("DELETE")
	d.Press("enter")
	d.RequireContains("Archive deleted")

	open(d, "34")
	d.Press("esc")
	d.RequireAbsent("Heads up")

	open(d, "2") // restart: cancel then accept
	d.Press("n")
	d.RequireContains("Restart cancelled")
	open(d, "2")
	d.Press("y")
	d.RequireContains("Service restarted.")
}

func TestChoiceSliderAndEditPopups(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "36")
	d.Press("3", "enter")
	d.RequireContains("Level: Urgent")

	open(d, "37")
	d.Press("end", "enter")
	d.RequireContains("Keeping 30 days")

	open(d, "38")
	d.Type("nobody")
	d.Press("enter")
	d.RequireContains(`"nobody" does not look like an email address`)
	d.Type("@example.com")
	d.Press("enter")
	d.RequireContains("Alerts go to nobody@example.com")
}

func TestScrollableNoticeScrolls(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "32")
	d.RequireContains("v1.01")
	d.RequireAbsent("v1.60")
	d.Press("end")
	d.RequireContains("v1.60")
	d.RequireAbsent("v1.01")
}

func TestLiveNoticeUpdatesWhileOpen(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "31")
	d.RequireContains("12:00:00")
	d.Advance(30 * time.Second)
	d.RequireContains("12:00:30")
}

func TestDiceModalKeepsItsOwnState(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "40")
	d.Press("space", "space", "space")
	d.RequireAbsent("no rolls yet")
	d.RequireContains("History")
	d.Press("r")
	d.RequireContains("no rolls yet")
	d.Press("esc")
	d.RequireAbsent("Space roll")
	open(d, "40") // a fresh modal starts empty
	d.RequireContains("no rolls yet")
}

func TestPingModalReceivesItsOwnTaskResult(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "41")
	d.Press("enter")
	d.RequireContains("reply in")
	d.RequireContains("Status   ✓ reply in") // the result arrived inside the pop-up, still open
	if strings.Contains(d.Lines()[len(d.Lines())-1], "reply in") {
		t.Error("the footer should not repeat a result the pop-up consumed")
	}
	d.Press("esc")
	d.RequireAbsent("press Enter to ping")
}

func TestModalChainFlowsThroughThreeSteps(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "42")
	d.RequireContains("Step 1 of 3")
	d.Press("enter")
	d.RequireContains("Step 2 of 3")
	d.Type("Apollo")
	d.Press("enter")
	d.RequireContains("Step 3 of 3", `Project "Apollo" was created.`)
}

func TestExportReportIsHandledByAnOnResultHook(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "3")
	d.RequireContains("Report", "[exported]", "CPU", "Memory", "Top processes")
	if strings.Contains(d.Lines()[len(d.Lines())-1], "Building the report") {
		t.Error("the hook handled the result, so the working message must be cleared")
	}
	d.Press("esc")
	d.RequireContains("Export Report")
}

func TestDiagnosticsViewHasItsOwnHotkeyAndOptions(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("d") // the option hotkey opens the view from the Monitor page
	d.RequireContains("Diagnostics", "HEALTHY", "Uptime", "Requests", "DIAG")
	rows(t, d, "1", "Run Self-Test", "2", "Open Logs", "3", "Back to the Page")
	d.Press("t") // a hotkey that exists only inside this view
	d.RequireContains("Self-test passed")
	d.Advance(10 * time.Second)
	d.Type("2")
	d.Press("enter") // a view opens another view
	d.RequireContains("LOGS", "self-test passed")
	d.Press("esc", "esc")
	d.RequireContains("System") // back on the page we started from
	d.Press("t")                // the view's hotkey is not active here
	d.RequireAbsent("Self-test passed: 12")
}

func TestProcessesViewSelectsSortsAndKills(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("p")
	d.RequireContains("Processes", "api-gateway", "12 running", "sorted by cpu")
	first := rowOf(d, "❯")
	d.Press("down")
	if rowOf(d, "❯") == first {
		t.Error("down should move the selection")
	}
	d.Press("m")
	d.RequireContains("sorted by mem")
	d.RequireContains("postgres") // the largest by memory
	d.Press("home")
	if !strings.Contains(rowOf(d, "❯"), "postgres") {
		t.Errorf("home should select the top row, which is postgres by memory:\n%s", d.Screen())
	}
	d.Press("enter")
	d.RequireContains("postgres", "Memory")
	d.Press("esc")
	d.Press("k") // kill
	d.RequireContains("Killed postgres.", "11 running")
	d.RequireAbsent("postgres  ")
	d.Press("c")
	d.RequireContains("sorted by cpu")
	d.Press("esc")
	d.RequireContains("System")
}

func rowOf(d *dashtest.Driver, s string) string {
	for _, l := range d.Lines() {
		if strings.Contains(l, s) {
			return l
		}
	}
	return ""
}

func TestBackspaceGoesBackBecauseTheKeyIsRebound(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("d")
	d.RequireContains("DIAG")
	d.Press("backspace")
	d.RequireAbsent("DIAG")
	d.RequireContains("System")
}

func TestNavigationRowsAreReorderedByOrder(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("tab", "tab", "tab", "tab")
	lines := d.Lines()
	at := map[string]int{}
	for i, l := range lines {
		for _, name := range []string{"Jump to Charts", "Jump to Theme", "Jump to Monitor"} {
			if strings.Contains(l, name) {
				at[name] = i
			}
		}
	}
	if at["Jump to Monitor"] >= at["Jump to Theme"] || at["Jump to Theme"] >= at["Jump to Charts"] {
		t.Errorf("Order should reverse the listing, got rows %v:\n%s", at, d.Screen())
	}
	d.Type("60") // but typing the number still finds the right row
	d.Press("enter")
	d.RequireContains("Now on Charts", "Area & Sparkline")
	open(d, "61")
	d.RequireContains("Appearance")
}

func TestAlertMessageIsHandledByAnOnMsgHook(t *testing.T) {
	d := newApp(t, store.NewMemory())
	// The addon counts alerts in a closure that lives as long as the process, so
	// the numbers depend on which tests ran first: match the pattern, not the count.
	open(d, "70")
	d.RequireContains("disk usage above 80%")
	if !regexp.MustCompile(`Alert #\d+: disk usage`).MatchString(d.Screen()) {
		t.Errorf("alert message missing:\n%s", d.Screen())
	}
	d.Advance(10 * time.Second) // an odd tick count changes the alert's level
	open(d, "70")
	d.RequireContains("replica lag exceeds 30 seconds")
	d.Press("g")
	d.RequireContains("replica lag", "ERROR", "disk usage above 80%", "WARN")
}

func TestOutageMakesTheDashboardStaleAndRecovers(t *testing.T) {
	st := store.NewMemory()
	d := newApp(t, st)
	d.RequireContains("LIVE", "ONLINE")
	open(d, "71")
	d.RequireContains("Outage started")
	d.Advance(2 * time.Second)
	d.RequireContains("STALE") // the header
	d.RequireAbsent("ONLINE")
	d.Press("tab", "tab")                           // back to the Monitor page, wrapping past Theme
	d.RequireContains("System", "STALE", "no data") // the tab badge and widget badge too
	d.RequireContains("CPU")                        // the last good data is still drawn

	open(d, "71") // end the outage
	d.Advance(2 * time.Second)
	d.Press("tab", "tab")
	d.RequireContains("ONLINE", "LIVE")
	d.RequireAbsent("STALE")
}

func TestLogViewShowsStdlibAndSlogEntries(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "73")
	d.Advance(10 * time.Second)
	d.Press("g")
	d.RequireContains("LIVE", "slog error", "connection reset by peer", "slog warning", "standard logger, error")
	d.RequireContains("ERROR", "WARN", "INFO")
	d.Press("f", "f", "f") // Error only
	d.RequireContains("slog error", "standard logger, error")
	d.RequireAbsent("standard logger, info")
	raw := d.Raw()
	if strings.Contains(raw, "\x1b[2J") {
		t.Error("escape sequences in log text must never reach the terminal")
	}
}

func TestGlobalShortcuts(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.RequireContains("[Ctrl+Q] Stop & Exit", "[Ctrl+R] Refresh", "[?] Help")
	d.RequireAbsent("[ctrl+q]")

	d.Press("ctrl+r")
	d.RequireContains("Refreshing...")

	open(d, "73")
	d.Press("g") // inside the log view
	d.RequireContains("standard logger")
	d.Press("ctrl+l") // clear, from inside a view
	d.RequireContains("no log entries yet")
	d.Press("esc")

	d.Press("ctrl+q")
	if d.Quit() {
		t.Fatal("ctrl+q should ask first")
	}
	d.RequireContains("Stop and exit?")
	d.Press("esc")
	open(d, "10")
	d.Press("ctrl+q") // over another pop-up
	d.RequireContains("Stop and exit?")
	d.Press("enter")
	if !d.Quit() {
		t.Fatal("confirming should stop")
	}
	if got := d.Exit(); got.Reason != "stop" || got.Message != "Service stopped." {
		t.Errorf("exit = %+v", got)
	}
}

func TestCtrlCAsksThroughOnForceQuit(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("ctrl+c")
	if d.Quit() {
		t.Fatal("ctrl+c should ask first")
	}
	d.RequireContains("Quit the dashboard?")
	d.Press("enter")
	if !d.Quit() || d.Exit().Message != "Goodbye." || d.Exit().Reason != "" {
		t.Errorf("quit=%v exit=%+v", d.Quit(), d.Exit())
	}
}

func TestHelpShortcutWorksOnPagesOnly(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("?")
	d.RequireContains("Keys", "Ctrl+Q", "stop the service")
	d.Press("esc")
	d.Press("d") // into a view
	d.Press("?")
	d.RequireAbsent("stop the service")
}

func TestResetEndsWithAReason(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "90")
	d.Type("RESET")
	d.Press("enter")
	if !d.Quit() {
		t.Fatal("RESET should quit")
	}
	if got := d.Exit(); got.Reason != "reset" {
		t.Errorf("exit = %+v", got)
	}
}

func TestAppearanceAddonsAndPersistence(t *testing.T) {
	st := store.NewMemory()
	d := newApp(t, st)
	d.Press("m") // theme picker, by hotkey
	d.RequireContains("Solarized Dark")
	d.Press("end", "enter") // Solarized, this app's own palette, is listed last
	d.RequireContains("Color theme: Solarized Dark")
	d.Type("21")
	d.Press("enter", "down", "down", "enter", "right", "right", "enter") // Section gap 4; the menu reopens
	d.Press("esc")
	d.Type("23")
	d.Press("enter", "end", "enter") // TTY

	again := newApp(t, st)
	again.Press("tab", "tab", "tab", "tab", "tab")
	again.RequireContains("[TTY]", "┌", "[0.0.4.0]", "[Solarized Dark]")
}

func TestPaletteBackgroundReachesTheTerminal(t *testing.T) {
	d := newAppWith(t, dashboard.Options{Store: store.NewMemory(), Palette: "solarized"})
	v := d.Model().View()
	if v.BackgroundColor == nil || v.ForegroundColor == nil {
		t.Fatal("Solarized sets a background and foreground")
	}
	r, g, b, _ := v.BackgroundColor.RGBA()
	if r>>8 != 0x00 || g>>8 != 0x2B || b>>8 != 0x36 {
		t.Errorf("background = #%02X%02X%02X", r>>8, g>>8, b>>8)
	}
	d.Press("tab", "tab", "tab", "tab", "tab")
	d.RequireContains("#002B36", "Solarized Dark")

	off := newAppWith(t, dashboard.Options{Store: store.NewMemory(), Palette: "solarized", Background: dashygotui.BackgroundOff})
	if off.Model().View().BackgroundColor != nil {
		t.Error("BackgroundOff must leave the terminal's colours alone")
	}
	paint := newAppWith(t, dashboard.Options{Store: store.NewMemory(), Palette: "solarized", Background: dashygotui.BackgroundPaint})
	if !strings.Contains(paint.Raw(), "\x1b[48;2;0;43;54m") {
		t.Error("BackgroundPaint should write the background into the cells")
	}
}

func TestPageTitlesShortenOnNarrowTerminals(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.RequireContains("Monitor LIVE", "Charts", "Views & Hooks") // full titles, and the active tab's badge
	d.Resize(90, 40)
	d.RequireContains("Mon LIVE", "Chrt", "View")
	d.RequireAbsent("Views & Hooks", "Monitor")
	d.Resize(60, 40)
	d.RequireContains("M LIVE")
	d.RequireAbsent("Chrt", "Mon ")
}

func TestTTYModeIsConsoleSafeEverywhere(t *testing.T) {
	cfg := func() dashboard.Options {
		return dashboard.Options{Store: store.NewMemory(), Glyphs: dashygotui.GlyphsTTY}
	}
	safe := func(r rune) bool {
		return (r >= 0x20 && r <= 0x7e) || strings.ContainsRune("─│┌┐└┘├┤░▒▓█\n", r)
	}
	steps := map[string]func(d *dashtest.Driver){
		"monitor": func(*dashtest.Driver) {},
		"charts":  func(d *dashtest.Driver) { d.Press("tab") },
		"actions": func(d *dashtest.Driver) { d.Press("tab", "tab") },
		"popups":  func(d *dashtest.Driver) { d.Press("tab", "tab", "tab") },
		"views":   func(d *dashtest.Driver) { d.Press("tab", "tab", "tab", "tab") },
		"theme":   func(d *dashtest.Driver) { d.Press("tab", "tab", "tab", "tab", "tab") },
		"help":    func(d *dashtest.Driver) { d.Press("?") },
		"stop":    func(d *dashtest.Driver) { d.Press("ctrl+q") },
		"quit":    func(d *dashtest.Driver) { d.Press("ctrl+c") },
	}
	for _, n := range []string{"2", "3", "10", "12", "13", "14", "15", "30", "31", "32", "33", "34", "35", "36", "37", "38", "40", "41", "42", "20", "21", "22", "23", "73", "70"} {
		steps["option "+n] = func(d *dashtest.Driver) { open(d, n) }
	}
	steps["slider"] = func(d *dashtest.Driver) { open(d, "21"); d.Press("enter", "enter") }
	steps["diagnostics"] = func(d *dashtest.Driver) { d.Press("d") }
	steps["processes"] = func(d *dashtest.Driver) { d.Press("p") }
	steps["logs"] = func(d *dashtest.Driver) { open(d, "73"); d.Press("g") }
	steps["notes"] = func(d *dashtest.Driver) { d.Press("n", "pgdown") }
	steps["log tabs"] = func(d *dashtest.Driver) { open(d, "73"); d.Press("g", "c", "c") }
	steps["process details"] = func(d *dashtest.Driver) { d.Press("p", "enter") }
	for name, step := range steps {
		t.Run(name, func(t *testing.T) {
			dashboard.Logs.Clear()
			d := dashtest.New(t, dashboard.Config(cfg()), 160, 44)
			for i := 0; i < 8; i++ {
				d.Advance(time.Second)
			}
			step(d)
			seen := map[rune]bool{}
			for _, r := range d.Screen() {
				if !safe(r) && !seen[r] {
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

func TestEverythingFitsEverySize(t *testing.T) {
	sizes := [][2]int{{1, 1}, {20, 6}, {39, 9}, {40, 10}, {60, 14}, {79, 24}, {100, 30}, {160, 44}, {240, 70}}
	scenarios := map[string]func(d *dashtest.Driver){
		"p1":          func(*dashtest.Driver) {},
		"p2":          func(d *dashtest.Driver) { d.Press("tab") },
		"p3":          func(d *dashtest.Driver) { d.Press("tab", "tab") },
		"p4":          func(d *dashtest.Driver) { d.Press("tab", "tab", "tab") },
		"p5":          func(d *dashtest.Driver) { d.Press("tab", "tab", "tab", "tab") },
		"p6":          func(d *dashtest.Driver) { d.Press("tab", "tab", "tab", "tab", "tab") },
		"diagnostics": func(d *dashtest.Driver) { d.Press("d") },
		"processes":   func(d *dashtest.Driver) { d.Press("p") },
		"notes":       func(d *dashtest.Driver) { d.Press("n", "pgdown", "end") },
		"logs":        func(d *dashtest.Driver) { open(d, "73"); d.Press("g") },
		"log tabs":    func(d *dashtest.Driver) { open(d, "73"); d.Press("g", "c", "c", "C") },
		"modal":       func(d *dashtest.Driver) { open(d, "36") },
		"dice":        func(d *dashtest.Driver) { open(d, "40") },
		"ping":        func(d *dashtest.Driver) { open(d, "41") },
		"notice":      func(d *dashtest.Driver) { open(d, "32") },
	}
	for name, setup := range scenarios {
		for _, sz := range sizes {
			t.Run(fmt.Sprintf("%s_%dx%d", name, sz[0], sz[1]), func(t *testing.T) {
				d := newApp(t, store.NewMemory())
				for i := 0; i < 3; i++ {
					d.Advance(time.Second)
				}
				setup(d)
				d.Resize(sz[0], sz[1])
				got := strings.Split(d.Raw(), "\n")
				if len(got) != sz[1] {
					t.Fatalf("%d rows, want %d", len(got), sz[1])
				}
				for i, l := range got {
					if w := text.Width(l); w != sz[0] {
						t.Fatalf("row %d is %d wide, want %d", i, w, sz[0])
					}
				}
				d.Press("down", "up", "pgdown", "home", "end") // keys at this size must not panic
			})
		}
	}
}

func TestTheBackgroundCanBeTurnedOffAndOnByTheUser(t *testing.T) {
	st := store.NewMemory()
	d := newApp(t, st)
	view := func() (bg bool) { return d.Model().View().BackgroundColor != nil }
	if !view() {
		t.Fatal("the default palette has a background, so the app has one out of the box")
	}
	open(d, "24")
	d.Press("end", "enter") // Off
	if view() {
		t.Error("Off must leave the terminal's colours alone")
	}
	d.RequireContains("[off]")

	again := newApp(t, st) // remembered
	if again.Model().View().BackgroundColor != nil {
		t.Error("the saved choice should apply on the next run")
	}
	open(again, "24")
	again.Press("home", "enter") // back to Terminal
	if again.Model().View().BackgroundColor == nil {
		t.Error("choosing Terminal should turn the background back on")
	}
}

func TestEveryPaletteHasABackgroundExceptTheConsole(t *testing.T) {
	for _, id := range []string{"default", "catppuccin", "dracula", "nord", "solarized"} {
		d := newAppWith(t, dashboard.Options{Store: store.NewMemory(), Palette: id})
		if d.Model().View().BackgroundColor == nil {
			t.Errorf("%s sets no background", id)
		}
	}
	d := newAppWith(t, dashboard.Options{Store: store.NewMemory(), Palette: "console"})
	if d.Model().View().BackgroundColor != nil {
		t.Error("the 16-colour console palette cannot take a background")
	}
}

func TestTransparentBackgroundHotkeyAndFlag(t *testing.T) {
	d := newApp(t, store.NewMemory())
	if d.Model().View().BackgroundColor == nil {
		t.Fatal("themed by default")
	}
	d.Press("t") // the option's hotkey, from the Monitor page
	d.RequireContains("Transparent background on")
	if d.Model().View().BackgroundColor != nil {
		t.Error("transparent: the terminal's own background must show through")
	}
	d.Press("t")
	if d.Model().View().BackgroundColor == nil {
		t.Error("and back")
	}

	// A developer starts transparent with Config.Background; a user can still switch it.
	start := newAppWith(t, dashboard.Options{Store: store.NewMemory(), Background: dashygotui.BackgroundTransparent})
	if start.Model().View().BackgroundColor != nil {
		t.Error("starting transparent")
	}
	start.Press("t")
	if start.Model().View().BackgroundColor == nil {
		t.Error("the user can turn the themed background on")
	}
}

// openLogs opens the Logs view and puts it on the All tab at the debug level,
// which the assertions below expect. (A new dashboard starts at the default level.)
func openLogs(d *dashtest.Driver) {
	d.Press("g")
	for i := 0; i < 12 && !d.Contains("[All]"); i++ {
		d.Press("c")
	}
	for i := 0; i < 5 && !d.Contains("f filter: debug"); i++ {
		d.Press("f")
	}
	d.Press("home")
}

func TestLogsHaveCategoryTabsSwitchedByC(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "73") // writes entries in several categories
	openLogs(d)
	d.RequireContains("[All]", "net", "disk", "auth", "c category")
	d.RequireContains("write failed", "failed login", "listening") // All shows everything

	d.Press("c")
	d.RequireContains("[net]", "listening", "slow upstream")
	d.RequireAbsent("write failed", "failed login", "[All]")
	d.Press("c")
	d.RequireContains("[disk]", "write failed", "volume mounted")
	d.RequireAbsent("listening")
	d.Press("c")
	d.RequireContains("[auth]", "failed login")
	d.RequireAbsent("write failed")
	for i := 0; i < 12 && !d.Contains("[All]"); i++ {
		d.Press("c") // the remaining tabs, then round to All
	}
	d.RequireContains("[All]", "listening", "write failed")
	d.Press("C") // the other direction wraps to the last tab
	d.RequireAbsent("[All]")
}

func TestReleaseNotesScrollAndCountVisits(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("n")
	d.RequireContains("Release Notes", "Version 0.40.0", "Scroll", "NOTES")
	d.Press("end")
	d.RequireContains("Version 0.1.0")
	d.RequireAbsent("Version 0.40.0")
	d.Press("esc")
	d.RequireContains("System") // back on the page the hotkey was pressed on

	// Entering and leaving the view ran its hooks, which log to the "ui" tab.
	openLogs(d)
	for i := 0; i < 12 && !d.Contains("[ui]"); i++ {
		d.Press("c")
	}
	d.RequireContains("[ui]", "release notes shown", "release notes hidden")
}

func TestMonitorFallsBackToASummaryOnASmallTerminal(t *testing.T) {
	d := newApp(t, store.NewMemory())
	for i := 0; i < 3; i++ {
		d.Advance(time.Second)
	}
	d.RequireContains("TRAFFIC", "CPU Waveforms")
	d.RequireAbsent("compact view")
	d.Resize(100, 14)
	d.RequireContains("compact view", "CPU", "Memory", "Disk", "Traffic")
	d.RequireAbsent("TRAFFIC", "CPU Waveforms")
	d.Resize(160, 44)
	d.RequireContains("TRAFFIC")
}

func TestPushedEventsReachTheDashboard(t *testing.T) {
	var events atomic.Int64
	o := opts(store.NewMemory())
	o.Events = &events
	d := newAppWith(t, o)
	d.Press("d")
	d.RequireContains("0 events")
	events.Store(7) // the feed counted something...
	d.Notify()      // ...and told the dashboard, with no tick needed
	d.RequireContains("7 events")
}

// newestEvent is the highest "#nnnn" event number on screen. The stream's
// events live in the addon's closure for the whole process, so a test cannot
// know the absolute numbers.
func newestEvent(t *testing.T, d *dashtest.Driver) int {
	t.Helper()
	best := -1
	for _, m := range regexp.MustCompile(`#(\d{4})`).FindAllStringSubmatch(d.Screen(), -1) {
		n, _ := strconv.Atoi(m[1])
		best = max(best, n)
	}
	if best < 0 {
		t.Fatalf("no events on screen:\n%s", d.Screen())
	}
	return best
}

func TestEventStreamIsATailView(t *testing.T) {
	d := newApp(t, store.NewMemory())
	for i := 0; i < 60; i++ {
		d.Advance(time.Second)
	}
	d.Press("e")
	d.RequireContains("Event Stream", "STREAM")
	live := newestEvent(t, d) // starts at the newest event
	d.RequireAbsent("Live")

	d.Press("up", "up", "up")
	d.RequireContains("Live")
	back := newestEvent(t, d)
	if back != live-3 {
		t.Fatalf("three rows back should end at #%04d, got #%04d", live-3, back)
	}

	for i := 0; i < 10; i++ {
		d.Advance(time.Second) // ten more events arrive
	}
	if got := newestEvent(t, d); got != back {
		t.Errorf("a view scrolled back must hold its place: was #%04d, now #%04d", back, got)
	}
	d.Press("end")
	if got := newestEvent(t, d); got != live+10 {
		t.Errorf("End should return to the live bottom #%04d, got #%04d", live+10, got)
	}
	d.RequireAbsent("Live")
}

func TestStreamsWidgetShowsTheScale(t *testing.T) {
	d := newApp(t, store.NewMemory())
	for i := 0; i < 20; i++ {
		d.Advance(time.Second)
	}
	d.Press("tab")
	d.RequireContains("Streams", "Load", "Read", "Write", "≤")
}

func TestShortcutHintsAreWrittenAndColoured(t *testing.T) {
	d := newApp(t, store.NewMemory())
	raw := d.Raw()
	colour := func(word string) string {
		i := strings.Index(raw, word)
		if i < 0 {
			t.Fatalf("%q is not on screen", word)
		}
		j := strings.LastIndex(raw[:i], "\x1b[")
		return raw[j:i]
	}
	stop, refresh := colour("[Ctrl+Q]"), colour("[Ctrl+R]")
	if stop == refresh {
		t.Errorf("Stop (danger) and Refresh (good) must differ, both are %q", stop)
	}
	if colour("Stop & Exit") != stop {
		t.Error("the danger tone colours the whole hint, label included")
	}
}

func TestOpenWebHotkeyIsWrittenAsCtrlO(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("tab", "tab")
	d.RequireContains("Open Web UI (Ctrl+O)")
	d.RequireAbsent("(ctrl+o)")
	d.Press("ctrl+o")
	d.RequireContains("Would open http://localhost:8080")
}

func TestStorageSaveChainsAReload(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "10")
	d.Type("60")
	d.Press("enter")
	d.RequireContains("Storage limit set to 60 GB. Service reloaded.") // OnSaved's task ran after the save
	d.RequireContains("[60 GB]")
}

func TestDiagnosticsViewChrome(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("d")
	d.RequireContains("Diagnostics", "Mode: ON", "Actions", "Enter runs the highlighted one", "[t] Self-test", "[Esc/b] Back")
	d.RequireAbsent("Options")
}

func TestRegionChoiceUsesTheNewChoiceFields(t *testing.T) {
	d := newApp(t, store.NewMemory())
	open(d, "13")
	d.RequireContains("now eu-west", "Changing the region moves the service", "[current]", "Enter Save", "Esc Keep current")
	d.RequireAbsent("Enter Select")
}

func TestEventStreamIsFramedInAPanel(t *testing.T) {
	d := newApp(t, store.NewMemory())
	for i := 0; i < 60; i++ {
		d.Advance(time.Second)
	}
	d.Press("e")
	d.RequireContains("Events", "newest at the bottom")
	if !strings.Contains(d.Screen(), "of 60") && !strings.Contains(d.Screen(), "rows ") {
		t.Errorf("the scroll position should be in the panel's border:\n%s", d.Screen())
	}
}

func TestChartsPageFallsBackByRule(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("tab") // charts
	d.RequireAbsent("compact view")
	d.Resize(70, 30) // narrow but tall
	d.RequireAbsent("compact view")
	d.Resize(160, 18) // short but wide
	d.RequireAbsent("compact view")
	d.Resize(70, 20) // narrow and short: the rule's first branch
	d.RequireContains("compact view: the graphs need a larger window")
	d.Resize(160, 12) // very short: the rule's second branch
	d.RequireContains("compact view")
}

func TestStartViewFlagOpensTheStack(t *testing.T) {
	o := opts(store.NewMemory())
	o.StartView = []string{"diagnostics", "stdaddons.logs"}
	d := newAppWith(t, o)
	d.RequireAbsent("Quick Actions")
	d.RequireContains("LOGS")
	d.Press("esc")
	d.RequireContains("Mode: ON") // the diagnostics view
	d.Press("esc")
	d.RequireContains("Monitor", "System")
}

func TestStdaddonsSettingsAreOneLineEach(t *testing.T) {
	st := store.NewMemory()
	d := newApp(t, st)
	d.Press("tab", "tab")
	d.RequireContains("Log Retention", "[30 days]", "Log Level", "[info]", "Display Name", "[not set]", "Notifications")

	open(d, "16")
	d.Type("90")
	d.Press("enter")
	d.RequireContains("Log Retention set to 90 days.", "[90 days]")

	open(d, "17")
	d.Press("down", "enter")
	d.RequireContains("Log Level set to warn.", "[warn]")

	open(d, "18")
	d.Type("Ada")
	d.Press("enter")
	d.RequireContains("Display Name saved.", "[Ada]")

	var days int
	if ok, _ := st.Get("logs.retention_days", &days); !ok || days != 90 {
		t.Errorf("stored retention = %d", days)
	}
}

func TestViewEntriesNeedNoSeparateOption(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.RequireContains("[d] Diagnostics") // Entry.HotkeyHint advertises the hotkey in the footer
	d.Press("tab", "tab", "tab", "tab")
	d.RequireContains("Diagnostics (d)", "Processes (p)", "Release Notes (n)")
	d.Press("d")
	d.RequireContains("Mode: ON", "DIAGNOSTICS")
}

func TestDiagnosticsSessionLogsWhileOpen(t *testing.T) {
	d := newApp(t, store.NewMemory())
	d.Press("d")
	d.Press("esc")
	openLogs(d) // the viewer keeps its tab and filter between tests
	d.RequireContains("diagnostics opened", "diagnostics closed")
}

// The showcase passes the library's own audit: every page and view at every
// size, exact frames, and only console-safe characters in TTY mode.
func TestShowcasePassesTheAudit(t *testing.T) {
	dashtest.Audit(t, dashboard.Config(opts(store.NewMemory())))
}

// The counts the examples README quotes must match the application, or the
// README is lying to whoever reads it first.
func TestExamplesReadmeCountsAreTrue(t *testing.T) {
	b, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	line := ""
	for _, l := range strings.Split(string(b), "\n") {
		if strings.Contains(l, "](showcase/)") {
			line = l
		}
	}
	d, err := dashygotui.New(dashboard.Config(opts(store.NewMemory())))
	if err != nil {
		t.Fatal(err)
	}
	out := d.Outline()
	files, _ := filepath.Glob("dashboard/addons/*.go")
	for _, want := range []string{
		fmt.Sprintf("%d options", len(out.Options)),
		fmt.Sprintf("%d views", len(out.Views)),
		fmt.Sprintf("%d one-file addons", len(files)),
	} {
		if !strings.Contains(line, want) {
			t.Errorf("examples/README.md says %q, but the showcase has %q", line, want)
		}
	}
}
