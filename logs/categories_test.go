package logs_test

import (
	"bytes"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/text"
)

// categorised returns a buffer with entries in three categories, oldest first:
// net 1, disk 2, net 3, auth 4, disk 5, net 6.
func categorised() *logs.Buffer {
	b := logs.NewBuffer(100)
	b.In("net").Add(logs.Info, "net one")
	b.In("disk").Add(logs.Info, "disk two")
	b.In("net").Add(logs.Warn, "net three")
	b.In("auth").Add(logs.Info, "auth four")
	b.In("disk").Add(logs.Error, "disk five")
	b.In("net").Add(logs.Info, "net six")
	return b
}

func screen(v *logs.Viewer, cols, rows int) string {
	return strings.Join(plain(v.Rows(th(false), cols, rows)), "\n")
}

func TestEntriesRememberTheirCategory(t *testing.T) {
	b := categorised()
	got := map[string]int{}
	for _, e := range b.Snapshot() {
		got[e.Category]++
	}
	if got["net"] != 3 || got["disk"] != 2 || got["auth"] != 1 {
		t.Errorf("counts = %v", got)
	}
}

func TestPlainAddUsesTheDefaultCategory(t *testing.T) {
	b := logs.NewBuffer(10)
	b.Add(logs.Info, "x")
	_, _ = b.Write([]byte("y\n"))
	b.In("").Add(logs.Info, "z")
	for _, e := range b.Snapshot() {
		if e.Category != logs.DefaultCategory {
			t.Errorf("%q is in %q, want %q", e.Text, e.Category, logs.DefaultCategory)
		}
	}
}

func TestCategoriesListInOrderOfFirstAppearance(t *testing.T) {
	b := categorised()
	if got := strings.Join(b.Categories(), ","); got != "net,disk,auth" {
		t.Errorf("categories = %q", got)
	}
	b.Clear()
	if got := strings.Join(b.Categories(), ","); got != "net,disk,auth" {
		t.Errorf("Clear should keep the tabs steady, got %q", got)
	}
}

func TestScopedWriterHandlerAndDivert(t *testing.T) {
	b := logs.NewBuffer(20)
	fmtWriter := b.In("net")
	_, _ = fmtWriter.Write([]byte("[WARN] via writer\n"))
	slog.New(b.In("disk").Handler(nil)).Error("via slog", "k", "v")
	l := log.New(&bytes.Buffer{}, "", 0)
	restore := b.In("auth").Divert(l)
	l.Print("via logger")
	restore()
	l.Print("after restore") // goes back to the old writer

	byText := map[string]logs.Entry{}
	for _, e := range b.Snapshot() {
		byText[strings.Fields(e.Text)[0]+" "+strings.Fields(e.Text)[1]] = e
	}
	check := func(key, cat string, lvl logs.Level) {
		t.Helper()
		e, ok := byText[key]
		if !ok {
			t.Fatalf("no entry %q in %v", key, byText)
		}
		if e.Category != cat || e.Level != lvl {
			t.Errorf("%q: category %q level %v, want %q %v", key, e.Category, e.Level, cat, lvl)
		}
	}
	check("via writer", "net", logs.Warn)
	check("via logger", "auth", logs.Info)
	if b.Len() != 3 {
		t.Errorf("len = %d, want 3 (the restored logger must not write to the buffer)", b.Len())
	}
	for _, e := range b.Snapshot() {
		if strings.Contains(e.Text, "via slog") && (e.Category != "disk" || e.Level != logs.Error) {
			t.Errorf("slog entry = %+v", e)
		}
	}
}

func TestScopedLoggingIsConcurrencySafe(t *testing.T) {
	b := logs.NewBuffer(5000)
	var wg sync.WaitGroup
	for _, c := range []string{"a", "b", "c", "d"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := b.In(c)
			for range 200 {
				s.Add(logs.Info, "x")
			}
		}()
	}
	wg.Wait()
	if b.Len() != 800 || len(b.Categories()) != 4 {
		t.Errorf("len %d, categories %v", b.Len(), b.Categories())
	}
}

func TestNoTabsWithoutCategories(t *testing.T) {
	b := fill(5) // everything in the default category
	v := logs.NewViewer(b)
	rows := plain(v.Rows(th(false), 60, 4))
	if !strings.Contains(rows[0], "entry 005") {
		t.Errorf("the first row should be an entry, not tabs: %q", rows[0])
	}
}

func TestTabsAppearWithACategory(t *testing.T) {
	v := logs.NewViewer(categorised())
	first := plain(v.Rows(th(false), 60, 6))[0]
	for _, want := range []string{"[All]", "net", "disk", "auth"} {
		if !strings.Contains(first, want) {
			t.Errorf("tab strip %q lacks %q", first, want)
		}
	}
}

func TestDeclaredCategoriesComeFirstAndShowEarly(t *testing.T) {
	b := logs.NewBuffer(10)
	v := logs.NewViewer(b, logs.WithCategories("storage", "net"))
	first := plain(v.Rows(th(false), 60, 3))[0]
	if strings.Index(first, "storage") >= strings.Index(first, "net") || !strings.Contains(first, "[All]") {
		t.Errorf("strip = %q", first)
	}
	b.In("auth").Add(logs.Info, "late")
	b.In("net").Add(logs.Info, "x")
	first = plain(v.Rows(th(false), 80, 3))[0]
	if strings.Index(first, "net") >= strings.Index(first, "auth") {
		t.Errorf("a discovered category goes after the declared ones: %q", first)
	}
	if got := strings.Join(v.Categories(), ","); got != "storage,net,auth" {
		t.Errorf("categories = %q", got)
	}
}

func TestCategoryKeyCyclesAndWraps(t *testing.T) {
	v := logs.NewViewer(categorised(), logs.WithCategoryKey("c"))
	var seen []string
	for range 5 {
		v.HandleKey("c", 10)
		seen = append(seen, v.Category())
	}
	if got := strings.Join(seen, ","); got != "net,disk,auth,,net" {
		t.Errorf("sequence = %q (an empty name is All)", got)
	}
	// Reverse with the upper-case letter or shift+c.
	v.HandleKey("C", 10)
	if v.Category() != "" {
		t.Errorf("C from net should go to All, got %q", v.Category())
	}
	v.HandleKey("shift+c", 10)
	if v.Category() != "auth" {
		t.Errorf("shift+c from All should wrap to auth, got %q", v.Category())
	}
}

func TestCategoryKeyBeatsScrollingKeys(t *testing.T) {
	v := logs.NewViewer(categorised(), logs.WithCategoryKey("j"))
	v.HandleKey("j", 10)
	if v.Category() != "net" {
		t.Errorf("the category key should win over j, category = %q", v.Category())
	}
}

func TestWithoutAKeyTheTabsDoNotSwitch(t *testing.T) {
	v := logs.NewViewer(categorised())
	for _, k := range []string{"c", "C", "tab", "left", "right"} {
		if v.HandleKey(k, 10) && k != "left" && k != "right" {
			t.Errorf("key %q was used with no category key set", k)
		}
	}
	if v.Category() != "" {
		t.Errorf("category = %q", v.Category())
	}
}

func TestSelectedCategoryFiltersEntries(t *testing.T) {
	v := logs.NewViewer(categorised(), logs.WithCategoryKey("c"))
	all := screen(v, 70, 10)
	for _, w := range []string{"net one", "disk two", "auth four"} {
		if !strings.Contains(all, w) {
			t.Errorf("All should show %q:\n%s", w, all)
		}
	}
	v.HandleKey("c", 10) // net
	s := screen(v, 70, 10)
	if !strings.Contains(s, "[net]") || !strings.Contains(s, "net six") || !strings.Contains(s, "net one") {
		t.Errorf("net tab:\n%s", s)
	}
	if strings.Contains(s, "disk") && strings.Contains(strings.SplitN(s, "\n", 2)[1], "disk") {
		t.Errorf("net view leaked disk entries:\n%s", s)
	}
	if strings.Contains(strings.SplitN(s, "\n", 2)[1], "auth four") {
		t.Errorf("net view leaked auth entries:\n%s", s)
	}
}

func TestOnlyTheSelectedTabIsHighlighted(t *testing.T) {
	v := logs.NewViewer(categorised(), logs.WithCategoryKey("c"))
	v.HandleKey("c", 10) // net
	first := plain(v.Rows(th(false), 80, 4))[0]
	if strings.Count(first, "[") != 1 || !strings.Contains(first, "[net]") {
		t.Errorf("exactly one tab should be marked: %q", first)
	}
	// And it is styled, not just bracketed.
	raw := v.Rows(th(false), 80, 4)[0]
	if !strings.Contains(raw, "\x1b[") {
		t.Errorf("the selected tab should be coloured: %q", raw)
	}
}

func TestEachCategoryKeepsItsOwnScrollPosition(t *testing.T) {
	b := logs.NewBuffer(500)
	for i := 1; i <= 40; i++ {
		b.In("net").Add(logs.Info, "net entry "+pad3(i))
		b.In("disk").Add(logs.Info, "disk entry "+pad3(i))
	}
	v := logs.NewViewer(b, logs.WithCategoryKey("c"))
	v.HandleKey("c", 10) // net
	for range 12 {
		v.HandleKey("down", 10)
	}
	netTop := plain(v.Rows(th(false), 70, 6))[1]
	if v.Live() {
		t.Fatal("net should be scrolled")
	}
	v.HandleKey("c", 10) // disk: still live
	if !v.Live() {
		t.Error("disk has not been scrolled, it should be live")
	}
	if !strings.Contains(plain(v.Rows(th(false), 70, 6))[1], "disk entry 040") {
		t.Errorf("disk should show its newest entry:\n%s", screen(v, 70, 6))
	}
	v.HandleKey("C", 10) // back to net
	if got := plain(v.Rows(th(false), 70, 6))[1]; got != netTop {
		t.Errorf("net should return to where it was:\n got %q\nwant %q", got, netTop)
	}
}

func pad3(i int) string {
	return string([]byte{byte('0' + i/100), byte('0' + i/10%10), byte('0' + i%10)})
}

func TestInactiveTabsCountNewEntries(t *testing.T) {
	b := logs.NewBuffer(100)
	b.In("net").Add(logs.Info, "n1")
	b.In("disk").Add(logs.Info, "d1")
	v := logs.NewViewer(b, logs.WithCategoryKey("c"))
	strip := func() string { return plain(v.Rows(th(false), 80, 3))[0] }

	if s := strip(); !strings.Contains(s, "net (1)") || !strings.Contains(s, "disk (1)") {
		t.Fatalf("unseen entries should be counted: %q", s)
	}
	v.HandleKey("c", 10) // net: read
	b.In("net").Add(logs.Info, "n2")
	b.In("disk").Add(logs.Info, "d2")
	if s := strip(); strings.Contains(s, "net (") || !strings.Contains(s, "disk (2)") {
		t.Fatalf("the open tab shows no count; disk has 2 unread: %q", s)
	}
	v.HandleKey("c", 10) // disk: read
	b.In("net").Add(logs.Info, "n3")
	if s := strip(); !strings.Contains(s, "net (1)") || strings.Contains(s, "disk (") {
		t.Fatalf("only what arrived after leaving net counts: %q", s)
	}
}

func TestLevelFilterAppliesInsideACategory(t *testing.T) {
	v := logs.NewViewer(categorised(), logs.WithCategoryKey("c"))
	v.HandleKey("c", 10)
	v.HandleKey("c", 10) // disk: info + error
	v.HandleKey("f", 10) // info
	v.HandleKey("f", 10) // warn
	v.HandleKey("f", 10) // error
	s := screen(v, 70, 8)
	if !strings.Contains(s, "disk five") || strings.Contains(s, "disk two") {
		t.Errorf("only the error should remain in disk:\n%s", s)
	}
	v.HandleKey("c", 10) // auth has no errors
	if s := screen(v, 70, 8); !strings.Contains(s, "no entries in auth at ERROR or above") {
		t.Errorf("empty-state message:\n%s", s)
	}
}

func TestEmptyCategoryMessage(t *testing.T) {
	b := logs.NewBuffer(10)
	b.In("net").Add(logs.Info, "x")
	v := logs.NewViewer(b, logs.WithCategories("net", "disk"), logs.WithCategoryKey("c"))
	v.HandleKey("c", 5)
	v.HandleKey("c", 5) // disk, which is declared but empty
	if s := screen(v, 60, 4); !strings.Contains(s, "no entries in disk") {
		t.Errorf("screen:\n%s", s)
	}
}

func TestSetCategoryIgnoresUnknownNames(t *testing.T) {
	v := logs.NewViewer(categorised())
	v.SetCategory("nope")
	if v.Category() != "" {
		t.Errorf("category = %q", v.Category())
	}
	v.SetCategory("disk")
	if v.Category() != "disk" {
		t.Errorf("category = %q", v.Category())
	}
	v.SetCategory("")
	if v.Category() != "" {
		t.Errorf("category = %q", v.Category())
	}
}

func TestTabStripAlwaysFitsAndKeepsTheSelectedTab(t *testing.T) {
	b := logs.NewBuffer(100)
	names := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf"}
	for _, n := range names {
		b.In(n).Add(logs.Info, "x")
	}
	v := logs.NewViewer(b, logs.WithCategoryKey("c"))
	for range len(names) + 1 {
		v.HandleKey("c", 5)
		for _, cols := range []int{1, 8, 20, 30, 80} {
			rows := v.Rows(th(false), cols, 4)
			if len(rows) != 4 {
				t.Fatalf("%d rows at %d cols", len(rows), cols)
			}
			for _, r := range rows {
				if w := text.Width(r); w != cols {
					t.Fatalf("a row is %d wide at %d cols: %q", w, cols, r)
				}
			}
		}
		if c := v.Category(); c != "" {
			if first := plain(v.Rows(th(false), 30, 4))[0]; !strings.Contains(first, "["+c+"]") {
				t.Errorf("the selected tab %q must stay visible at 30 cols: %q", c, first)
			}
		}
	}
}

func TestTabsUsePlainGlyphsOnATTY(t *testing.T) {
	b := logs.NewBuffer(100)
	for i, n := range []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf"} {
		b.In(n).Add(logs.Info, strings.Repeat("x", i+1))
	}
	v := logs.NewViewer(b, logs.WithCategoryKey("c"))
	for range 7 {
		v.HandleKey("c", 5)
	}
	for _, l := range plain(v.Rows(th(true), 24, 4)) {
		for _, r := range l {
			if r > 0x7e {
				t.Fatalf("TTY mode drew %q in %q", r, l)
			}
		}
	}
}
