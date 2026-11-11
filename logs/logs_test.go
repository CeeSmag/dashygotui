package logs_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

func texts(es []logs.Entry) string {
	var out []string
	for _, e := range es {
		out = append(out, e.Text)
	}
	return strings.Join(out, "|")
}

func TestSnapshotIsNewestFirst(t *testing.T) {
	b := logs.NewBuffer(10)
	for _, s := range []string{"one", "two", "three"} {
		b.Add(logs.Info, s)
	}
	if got := texts(b.Snapshot()); got != "three|two|one" {
		t.Errorf("snapshot = %q, want newest first", got)
	}
	es := b.Snapshot()
	if es[0].Seq != 3 || es[2].Seq != 1 {
		t.Errorf("sequence numbers = %d..%d", es[2].Seq, es[0].Seq)
	}
	es[0].Text = "mutated"
	if texts(b.Snapshot())[:5] == "mutat" {
		t.Error("Snapshot must return a copy")
	}
}

func TestBufferEvictsTheOldest(t *testing.T) {
	b := logs.NewBuffer(3)
	for i := 1; i <= 7; i++ {
		b.Add(logs.Info, fmt.Sprint(i))
	}
	if got := texts(b.Snapshot()); got != "7|6|5" {
		t.Errorf("snapshot = %q, want only the newest 3", got)
	}
	if b.Len() != 3 {
		t.Errorf("Len = %d", b.Len())
	}
	if es := b.Snapshot(); es[2].Seq != 5 {
		t.Errorf("oldest Seq = %d, want sequence numbers to keep counting", es[2].Seq)
	}
}

func TestDefaultCapacity(t *testing.T) {
	for _, c := range []int{0, -4} {
		b := logs.NewBuffer(c)
		for range logs.DefaultCapacity + 50 {
			b.Add(logs.Info, "x")
		}
		if b.Len() != logs.DefaultCapacity {
			t.Errorf("capacity %d: Len = %d, want %d", c, b.Len(), logs.DefaultCapacity)
		}
	}
}

func TestCountAndClear(t *testing.T) {
	b := logs.NewBuffer(10)
	b.Add(logs.Debug, "d")
	b.Add(logs.Info, "i")
	b.Add(logs.Warn, "w")
	b.Add(logs.Error, "e1")
	b.Add(logs.Error, "e2")
	for level, want := range map[logs.Level]int{logs.Debug: 5, logs.Info: 4, logs.Warn: 3, logs.Error: 2} {
		if got := b.Count(level); got != want {
			t.Errorf("Count(%v) = %d, want %d", level, got, want)
		}
	}
	b.Clear()
	if b.Len() != 0 || len(b.Snapshot()) != 0 {
		t.Error("Clear should empty the buffer")
	}
	b.Add(logs.Info, "after")
	if es := b.Snapshot(); es[0].Seq != 6 {
		t.Errorf("Seq after Clear = %d, want numbering to continue at 6", es[0].Seq)
	}
}

func TestLevelStrings(t *testing.T) {
	for l, want := range map[logs.Level]string{logs.Debug: "DEBUG", logs.Info: "INFO", logs.Warn: "WARN", logs.Error: "ERROR"} {
		if l.String() != want {
			t.Errorf("%d.String() = %q", l, l.String())
		}
	}
}

func TestWriteParsesLevelsAndTimestamps(t *testing.T) {
	tests := []struct {
		line  string
		level logs.Level
		text  string
	}{
		{"plain message", logs.Info, "plain message"},
		{"2026/11/11 10:15:02 server started", logs.Info, "server started"},
		{"2026/11/11 10:15:02.123456 [ERROR] disk full", logs.Error, "disk full"},
		{"10:15:02 WARN: cache cold", logs.Warn, "cache cold"},
		{"[debug] verbose detail", logs.Debug, "verbose detail"},
		{"ERROR something broke", logs.Error, "something broke"},
		{"warning: low space", logs.Warn, "low space"},
		{"fatal: cannot continue", logs.Error, "cannot continue"},
		{"info: all good", logs.Info, "all good"},
		{"information is power", logs.Info, "information is power"}, // not a level
		{"an ERROR in the middle", logs.Info, "an ERROR in the middle"},
		{"myapp [ERROR] failed to connect", logs.Error, "myapp failed to connect"},
		{"myapp: [warn] slow", logs.Warn, "myapp: slow"},
		{`time=2026-11-11T10:15:02.000Z level=WARN msg="slow query" ms=912`, logs.Warn, `msg="slow query" ms=912`},
		{`time=2026-11-11T10:15:02.000Z level=ERROR+2 msg=boom`, logs.Error, `msg=boom`},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			b := logs.NewBuffer(5)
			n, err := b.Write([]byte(tt.line + "\n"))
			if err != nil || n != len(tt.line)+1 {
				t.Fatalf("Write = %d, %v", n, err)
			}
			es := b.Snapshot()
			if len(es) != 1 || es[0].Level != tt.level || es[0].Text != tt.text {
				t.Errorf("got %+v, want level %v text %q", es, tt.level, tt.text)
			}
		})
	}
}

func TestWriteSplitsLinesAndSkipsBlanks(t *testing.T) {
	b := logs.NewBuffer(10)
	_, _ = b.Write([]byte("first\r\n\n  \nsecond\nthird"))
	if got := texts(b.Snapshot()); got != "third|second|first" {
		t.Errorf("entries = %q", got)
	}
}

func TestStandardLoggerCapture(t *testing.T) {
	b := logs.NewBuffer(10)
	var original bytes.Buffer
	l := log.New(&original, "pfx ", log.LstdFlags)

	restore := b.Divert(l)
	l.Printf("[WARN] disk at %d%%", 91)
	if original.Len() != 0 {
		t.Errorf("a diverted logger must not write to its old output: %q", original.String())
	}
	es := b.Snapshot()
	if len(es) != 1 || es[0].Level != logs.Warn || es[0].Text != "pfx disk at 91%" {
		t.Fatalf("captured %+v", es)
	}

	restore()
	l.Print("back to normal")
	if !strings.Contains(original.String(), "back to normal") || !strings.HasPrefix(original.String(), "pfx 20") {
		t.Errorf("restore should bring back the output and the flags: %q", original.String())
	}
	if b.Len() != 1 {
		t.Error("nothing more should be captured after restore")
	}
	b.Divert(nil)() // a nil logger is ignored
}

func TestSlogCapture(t *testing.T) {
	b := logs.NewBuffer(10)
	logger := slog.New(b.Handler(&slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Debug("noisy", "k", 1)
	logger.Error("kaboom", "code", 500)
	es := b.Snapshot()
	if len(es) != 2 || es[0].Level != logs.Error || es[1].Level != logs.Debug {
		t.Fatalf("captured %+v", es)
	}
	if !strings.Contains(es[0].Text, "kaboom") || !strings.Contains(es[0].Text, "code=500") {
		t.Errorf("text = %q", es[0].Text)
	}
	if strings.Contains(es[0].Text, "time=") || strings.Contains(es[0].Text, "level=") {
		t.Errorf("the slog prefix should be stripped: %q", es[0].Text)
	}
}

func TestConcurrentLogging(t *testing.T) {
	b := logs.NewBuffer(50)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				b.Add(logs.Info, fmt.Sprintf("g%d-%d", g, i))
				_ = b.Snapshot()
				b.Count(logs.Warn)
			}
		}()
	}
	wg.Wait()
	if b.Len() != 50 {
		t.Errorf("Len = %d", b.Len())
	}
	es := b.Snapshot()
	for i := 1; i < len(es); i++ {
		if es[i].Seq >= es[i-1].Seq {
			t.Fatalf("not strictly newest first at %d: %d then %d", i, es[i-1].Seq, es[i].Seq)
		}
	}
}

func th(tty bool) *theme.Theme { return theme.New(theme.Default(), tty) }

func plain(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = dashtest.Plain(r)
	}
	return out
}

func fill(n int) *logs.Buffer {
	b := logs.NewBuffer(500)
	for i := 1; i <= n; i++ {
		b.Add(logs.Info, fmt.Sprintf("entry %03d", i))
	}
	return b
}

func TestViewerShowsNewestAtTheTop(t *testing.T) {
	b := fill(30)
	v := logs.NewViewer(b)
	rows := plain(v.Rows(th(false), 60, 5))
	for i, want := range []string{"entry 030", "entry 029", "entry 028", "entry 027", "entry 026"} {
		if !strings.Contains(rows[i], want) {
			t.Errorf("row %d = %q, want it to hold %q", i, rows[i], want)
		}
	}
	if !v.Live() || v.Newer() != 0 {
		t.Error("a fresh viewer is live")
	}
	if !strings.Contains(dashtest.Plain(v.Badge(th(false))), "LIVE") {
		t.Error("the live badge should say LIVE")
	}
}

func TestViewerRowsHaveExactDimensions(t *testing.T) {
	b := logs.NewBuffer(50)
	b.Add(logs.Info, strings.Repeat("a very long message ", 20))
	b.Add(logs.Error, "short 日本語 text")
	b.Add(logs.Warn, "")
	for _, tty := range []bool{false, true} {
		v := logs.NewViewer(b)
		for cols := 1; cols <= 90; cols += 3 {
			for rows := 1; rows <= 12; rows++ {
				got := v.Rows(th(tty), cols, rows)
				if len(got) != rows {
					t.Fatalf("tty=%v %dx%d: %d rows", tty, cols, rows, len(got))
				}
				for i, r := range got {
					if w := text.Width(r); w != cols {
						t.Fatalf("tty=%v %dx%d row %d is %d wide", tty, cols, rows, i, w)
					}
				}
			}
		}
	}
	if logs.NewViewer(b).Rows(th(false), 0, 5) != nil || logs.NewViewer(b).Rows(th(false), 5, 0) != nil {
		t.Error("a non-positive size renders nothing")
	}
}

func TestViewerScrollsOlderAndNewer(t *testing.T) {
	b := fill(30)
	v := logs.NewViewer(b)
	first := func() string { return plain(v.Rows(th(false), 60, 3))[0] }

	if !v.HandleKey("down", 3) || !strings.Contains(first(), "entry 029") {
		t.Errorf("down should show an older entry at the top: %q", first())
	}
	if v.Live() || v.Newer() != 1 {
		t.Errorf("live=%v newer=%d after one step", v.Live(), v.Newer())
	}
	v.HandleKey("j", 3)
	if !strings.Contains(first(), "entry 028") {
		t.Errorf("j = %q", first())
	}
	v.HandleKey("up", 3)
	v.HandleKey("k", 3)
	if !v.Live() || !strings.Contains(first(), "entry 030") {
		t.Errorf("scrolling back up should return to live: %q", first())
	}
	v.HandleKey("up", 3) // already at the top: stays live
	if !v.Live() {
		t.Error("cannot scroll above the newest entry")
	}
}

func TestViewerPagesHomeAndEnd(t *testing.T) {
	b := fill(30)
	v := logs.NewViewer(b)
	first := func() string { return plain(v.Rows(th(false), 60, 6))[0] }

	v.HandleKey("pgdown", 6) // a page is rows-1 entries
	if !strings.Contains(first(), "entry 025") {
		t.Errorf("pgdown = %q", first())
	}
	v.HandleKey("end", 6)
	if !strings.Contains(first(), "entry 001") {
		t.Errorf("end = %q, want the oldest", first())
	}
	v.HandleKey("down", 6) // cannot go past the oldest
	if !strings.Contains(first(), "entry 001") {
		t.Errorf("down past the end = %q", first())
	}
	v.HandleKey("pgup", 6)
	if !strings.Contains(first(), "entry 006") {
		t.Errorf("pgup = %q", first())
	}
	v.HandleKey("home", 6)
	if !v.Live() || !strings.Contains(first(), "entry 030") {
		t.Errorf("home = %q", first())
	}
}

func TestViewerHoldsItsPlaceWhileEntriesArrive(t *testing.T) {
	b := fill(30)
	v := logs.NewViewer(b)
	v.HandleKey("down", 5)
	v.HandleKey("down", 5)
	v.HandleKey("down", 5) // top is entry 027
	first := func() string { return plain(v.Rows(th(false), 60, 5))[0] }
	if !strings.Contains(first(), "entry 027") {
		t.Fatalf("setup: %q", first())
	}

	for i := 31; i <= 40; i++ {
		b.Add(logs.Info, fmt.Sprintf("entry %03d", i))
	}
	if !strings.Contains(first(), "entry 027") {
		t.Errorf("new entries must not move the line being read: %q", first())
	}
	if v.Newer() != 13 {
		t.Errorf("Newer = %d, want the 10 new entries plus the 3 scrolled past", v.Newer())
	}
	if !strings.Contains(dashtest.Plain(v.Badge(th(false))), "13 newer") {
		t.Errorf("badge = %q", dashtest.Plain(v.Badge(th(false))))
	}

	v.HandleKey("home", 5) // and Home catches up with the newest
	if !strings.Contains(first(), "entry 040") {
		t.Errorf("home = %q", first())
	}
}

func TestLiveViewFollowsNewEntries(t *testing.T) {
	b := fill(5)
	v := logs.NewViewer(b)
	b.Add(logs.Info, "latest")
	if got := plain(v.Rows(th(false), 60, 3))[0]; !strings.Contains(got, "latest") {
		t.Errorf("a live view shows the newest entry: %q", got)
	}
}

func TestViewerSurvivesEvictionOfItsAnchor(t *testing.T) {
	b := logs.NewBuffer(10)
	for i := 1; i <= 10; i++ {
		b.Add(logs.Info, fmt.Sprintf("entry %03d", i))
	}
	v := logs.NewViewer(b)
	v.HandleKey("end", 4) // anchored on the oldest, entry 001
	for i := 11; i <= 30; i++ {
		b.Add(logs.Info, fmt.Sprintf("entry %03d", i))
	}
	rows := plain(v.Rows(th(false), 60, 4))
	if !strings.Contains(rows[0], "entry 021") {
		t.Errorf("the anchor was evicted; the view should settle on the oldest kept entry, got %q", rows[0])
	}
}

func TestViewerLevelFilter(t *testing.T) {
	b := logs.NewBuffer(20)
	b.Add(logs.Debug, "debug line")
	b.Add(logs.Info, "info line")
	b.Add(logs.Warn, "warn line")
	b.Add(logs.Error, "error line")
	v := logs.NewViewer(b)
	screen := func() string { return strings.Join(plain(v.Rows(th(false), 60, 6)), "\n") }

	if s := screen(); !strings.Contains(s, "debug line") || v.MinLevel() != logs.Debug {
		t.Errorf("by default every level shows:\n%s", s)
	}
	v.HandleKey("f", 6)
	if s := screen(); strings.Contains(s, "debug line") || !strings.Contains(s, "info line") || v.MinLevel() != logs.Info {
		t.Errorf("Info and above:\n%s", s)
	}
	v.HandleKey("f", 6)
	v.HandleKey("f", 6)
	if s := screen(); strings.Contains(s, "warn line") || !strings.Contains(s, "error line") {
		t.Errorf("Error only:\n%s", s)
	}
	v.HandleKey("f", 6) // wraps back to everything
	if v.MinLevel() != logs.Debug {
		t.Errorf("filter should wrap, got %v", v.MinLevel())
	}
}

func TestViewerEmptyStates(t *testing.T) {
	b := logs.NewBuffer(5)
	v := logs.NewViewer(b)
	if s := strings.Join(plain(v.Rows(th(false), 60, 3)), "\n"); !strings.Contains(s, "no log entries yet") {
		t.Errorf("empty buffer:\n%s", s)
	}
	b.Add(logs.Debug, "only debug")
	v.HandleKey("f", 3)
	v.HandleKey("f", 3)
	if s := strings.Join(plain(v.Rows(th(false), 80, 3)), "\n"); !strings.Contains(s, "no entries at WARN or above") {
		t.Errorf("filtered to nothing:\n%s", s)
	}
	v.HandleKey("up", 3) // scrolling with nothing to scroll must not panic
	v.HandleKey("end", 3)
	v.HandleKey("pgdown", 3)
}

func TestViewerIgnoresOtherKeys(t *testing.T) {
	v := logs.NewViewer(fill(3))
	for _, k := range []string{"enter", "esc", "a", "tab", ""} {
		if v.HandleKey(k, 5) {
			t.Errorf("key %q should not be handled", k)
		}
	}
}

func TestViewerWrapsLongEntries(t *testing.T) {
	b := logs.NewBuffer(5)
	b.Add(logs.Error, strings.Repeat("word ", 30))
	rows := plain(logs.NewViewer(b).Rows(th(false), 50, 8))
	if !strings.Contains(rows[0], "ERROR") || !strings.Contains(rows[0], "word") {
		t.Errorf("first row = %q", rows[0])
	}
	if !strings.HasPrefix(rows[1], strings.Repeat(" ", 15)) || !strings.Contains(rows[1], "word") {
		t.Errorf("continuation rows should be indented under the text: %q", rows[1])
	}
}

func TestLogTextCannotInjectEscapes(t *testing.T) {
	b := logs.NewBuffer(5)
	b.Add(logs.Info, "evil \x1b[2J\x1b[31mred\x1b]0;title\x07 \r\x08 text")
	raw := strings.Join(logs.NewViewer(b).Rows(th(false), 80, 3), "\n")
	stripped := dashtest.Plain(raw)
	for _, bad := range []string{"\x1b[2J", "\x1b]0;", "\x07", "\x08"} {
		if strings.Contains(raw, bad) {
			t.Errorf("log text leaked a control sequence %q", bad)
		}
	}
	if !strings.Contains(stripped, "evil") || !strings.Contains(stripped, "text") {
		t.Errorf("the readable part should survive: %q", stripped)
	}
}

func TestTTYModeUsesPlainGlyphs(t *testing.T) {
	b := fill(3)
	v := logs.NewViewer(b)
	v.HandleKey("down", 3)
	for _, r := range dashtest.Plain(v.Badge(th(true))) + strings.Join(plain(v.Rows(th(true), 40, 3)), "") {
		if r > 0x7e {
			t.Errorf("TTY mode drew %q", r)
		}
	}
	if !strings.Contains(dashtest.Plain(logs.NewViewer(b).Badge(th(true))), "* LIVE") {
		t.Error("the TTY live badge uses the ASCII bullet")
	}
}

func TestDivertDoesNotBreakOtherLoggers(t *testing.T) {
	b := logs.NewBuffer(5)
	other := log.New(io.Discard, "", 0)
	restore := b.Divert(log.New(io.Discard, "", 0))
	other.Print("untouched")
	restore()
	if b.Len() != 0 {
		t.Error("only the diverted logger is captured")
	}
}
