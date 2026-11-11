package dashygotui_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/store"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

// ttyAllowed reports whether r may appear in TTY mode: printable ASCII, the
// straight line-drawing characters and corners, and the shade blocks the
// console font has.
func ttyAllowed(r rune) bool {
	return (r >= 0x20 && r <= 0x7e) || strings.ContainsRune("─│┌┐└┘├┤░▒▓█\n", r)
}

func offenders(screen string) []string {
	seen := map[rune]bool{}
	var out []string
	for _, r := range screen {
		if !ttyAllowed(r) && !seen[r] {
			seen[r] = true
			out = append(out, fmt.Sprintf("%q (U+%04X)", r, r))
		}
	}
	return out
}

func TestAutoDetectsTheLinuxConsole(t *testing.T) {
	f := newFixture(t)
	f.cfg.Getenv = env("TERM", "linux")
	var tty bool
	var pal string
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd { tty, pal = c.TTY(), c.PaletteID(); return nil }})
	d := f.driver(120, 40)
	d.Press("tab")
	d.RequireContains("┌", ">") // straight corners and an ASCII pointer
	d.RequireAbsent("╭", "❯", "●")
	d.Press("p")
	if !tty || pal != "console" {
		t.Errorf("tty=%v palette=%q, want the TTY set with the 16-colour palette", tty, pal)
	}
}

func TestAutoStaysUnicodeElsewhere(t *testing.T) {
	f := newFixture(t)
	f.cfg.Getenv = env("TERM", "xterm-256color")
	var tty bool
	var pal string
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd { tty, pal = c.TTY(), c.PaletteID(); return nil }})
	d := f.driver(120, 40)
	d.Press("tab")
	d.RequireContains("╭", "❯")
	d.RequireAbsent("┌─┐")
	d.Press("p")
	if tty || pal != "default" {
		t.Errorf("tty=%v palette=%q", tty, pal)
	}
}

func TestForcedModesIgnoreTheEnvironment(t *testing.T) {
	f := newFixture(t)
	f.cfg.Getenv = env("TERM", "xterm")
	f.cfg.Glyphs = dashygotui.GlyphsTTY
	d := f.driver(120, 40)
	d.Press("tab")
	d.RequireContains("┌")
	d.RequireAbsent("╭")

	g := newFixture(t)
	g.cfg.Getenv = env("TERM", "linux")
	g.cfg.Glyphs = dashygotui.GlyphsUnicode
	u := g.driver(120, 40)
	u.Press("tab")
	u.RequireContains("╭")
}

func TestGlyphModeSwitchesLiveAndIsRemembered(t *testing.T) {
	st := store.NewMemory()
	f := newFixture(t)
	f.cfg.Store = st
	f.cfg.Getenv = env("TERM", "xterm")
	var mode dashygotui.GlyphMode
	var pal string
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "t", Run: func(c *ctx) tea.Cmd { c.SetGlyphMode(dashygotui.GlyphsTTY); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "u", Run: func(c *ctx) tea.Cmd { c.SetGlyphMode(dashygotui.GlyphsUnicode); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd { mode, pal = c.GlyphMode(), c.PaletteID(); return nil }})
	d := f.driver(120, 40)
	d.Press("tab")
	d.RequireContains("╭")

	d.Press("t")
	d.RequireContains("┌")
	d.RequireAbsent("╭")
	d.Press("p")
	if mode != dashygotui.GlyphsTTY || pal != "console" {
		t.Errorf("mode=%v palette=%q after switching to TTY", mode, pal)
	}

	d.Press("u") // back: the palette goes back with it, because none was chosen
	d.RequireContains("╭")
	d.Press("p")
	if mode != dashygotui.GlyphsUnicode || pal != "default" {
		t.Errorf("mode=%v palette=%q after switching back", mode, pal)
	}

	d.Press("t")
	var saved string
	if ok, err := st.Get("dashygotui.glyphs", &saved); err != nil || !ok || saved != "tty" {
		t.Fatalf("stored glyph mode = %q ok=%v err=%v", saved, ok, err)
	}
	again := newFixture(t)
	again.cfg.Store = st
	again.cfg.Getenv = env("TERM", "xterm")
	r := again.driver(120, 40)
	r.Press("tab")
	r.RequireContains("┌")
}

func TestAChosenPaletteSurvivesTheTTYDefault(t *testing.T) {
	f := newFixture(t)
	f.cfg.Palette = "nord"
	f.cfg.Getenv = env("TERM", "linux")
	var pal string
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd { pal = c.PaletteID(); return nil }})
	d := f.driver(120, 40)
	d.Press("p")
	if pal != "nord" {
		t.Errorf("palette = %q, want the configured one even on a TTY", pal)
	}
}

func TestTTYModeDrawsOnlyConsoleSafeGlyphs(t *testing.T) {
	f := newFixture(t)
	f.cfg.Glyphs = dashygotui.GlyphsTTY
	f.reg.View(dashygotui.View[state]{ID: "v", Title: "View", Subtitle: "sub", Mode: "VIEW",
		Status:  func(*ctx, dashygotui.Size) []string { return []string{"status row"} },
		Options: func(*ctx) []option { return []option{{Number: 1, Label: "Do It", Hotkey: "d", Run: noop}} }})
	open := func(m dashygotui.Modal[state]) func(c *ctx) tea.Cmd {
		return func(c *ctx) tea.Cmd { c.OpenModal(m); return nil }
	}
	items := []modal.Item{{Label: "One", Badge: "[x]", Detail: "first"}, {Label: "Two", Detail: "second"}}
	long := make([]string, 40)
	for i := range long {
		long[i] = "a long line of notice text number " + fmt.Sprint(i)
	}
	scenarios := map[string]func(d *dashtest.Driver){
		"monitor": func(*dashtest.Driver) {},
		"control": func(d *dashtest.Driver) { d.Press("tab") },
		"theme":   func(d *dashtest.Driver) { d.Press("tab", "tab") },
		"view":    func(d *dashtest.Driver) { d.Press("w") },
		"flash ok": func(d *dashtest.Driver) {
			d.Press("tab", "f")
		},
		"modal edit":    func(d *dashtest.Driver) { d.Press("e") },
		"modal secret":  func(d *dashtest.Driver) { d.Press("s"); d.Type("pw") },
		"modal confirm": func(d *dashtest.Driver) { d.Press("c") },
		"modal typed":   func(d *dashtest.Driver) { d.Press("y") },
		"modal choice":  func(d *dashtest.Driver) { d.Press("h") },
		"modal slider":  func(d *dashtest.Driver) { d.Press("l") },
		"modal notice":  func(d *dashtest.Driver) { d.Press("n") },
	}
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "w", Run: func(c *ctx) tea.Cmd { _ = c.OpenView("v"); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "f", Run: func(c *ctx) tea.Cmd { c.FlashResult("done", nil); c.Flash("hint"); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "e", Run: open(&modal.Edit[state]{Title: "Edit", Help: "help text", Current: "cur", Validate: func(string) error { return fmt.Errorf("bad") }})})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "s", Run: open(&modal.Edit[state]{Title: "Secret", Secret: true})})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "c", Run: open(&modal.Confirm[state]{Title: "Confirm", Lines: []string{"sure?"}})})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "y", Run: open(&modal.Typed[state]{Title: "Typed", Word: "WORD", Prompt: "Type WORD"})})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "h", Run: open(&modal.Choice[state]{Title: "Choice", Items: items})})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "l", Run: open(&modal.Slider[state]{Title: "Slider", Min: 0, Max: 6, Value: 2})})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "n", Run: open(&modal.Notice[state]{Title: "Notice", Lines: long})})

	for name, setup := range scenarios {
		t.Run(name, func(t *testing.T) {
			d := f.driver(120, 40)
			d.Advance(time.Second)
			setup(d)
			if o := offenders(d.Screen()); len(o) > 0 {
				t.Errorf("TTY mode drew glyphs the console font may lack: %s\n%s", strings.Join(o, ", "), d.Screen())
			}
		})
	}
}

func TestTTYModeStillFitsEverySize(t *testing.T) {
	f := newFixture(t)
	f.cfg.Glyphs = dashygotui.GlyphsTTY
	for _, sz := range [][2]int{{1, 1}, {39, 9}, {40, 10}, {60, 12}, {120, 40}} {
		for page := range 3 {
			d := f.driver(120, 40)
			for range page {
				d.Press("tab")
			}
			d.Resize(sz[0], sz[1])
			lines := strings.Split(d.Raw(), "\n")
			if len(lines) != sz[1] {
				t.Fatalf("%dx%d page %d: %d rows", sz[0], sz[1], page, len(lines))
			}
			for i, l := range lines {
				if w := text.Width(l); w != sz[0] {
					t.Fatalf("%dx%d page %d row %d is %d wide", sz[0], sz[1], page, i, w)
				}
			}
		}
	}
}

func TestOnOffBadgeFollowsTheGlyphs(t *testing.T) {
	f := newFixture(t)
	f.cfg.Glyphs = dashygotui.GlyphsTTY
	d := f.driver(120, 40)
	d.Press("tab")
	d.RequireContains("[ON +]")
	d.RequireAbsent("✓")
}

// --- background theming ---

func backdropOf(t *testing.T, d *dashtest.Driver) (bg, fg bool) {
	t.Helper()
	v := d.Model().View()
	return v.BackgroundColor != nil, v.ForegroundColor != nil
}

func TestPaletteBackgroundIsSetOnTheTerminal(t *testing.T) {
	f := newFixture(t)
	f.cfg.Palette = "nord"
	f.cfg.Getenv = env("TERM", "xterm")
	d := f.driver(120, 40)
	bg, fg := backdropOf(t, d)
	if !bg || !fg {
		t.Errorf("nord should set the terminal's background and foreground: bg=%v fg=%v", bg, fg)
	}
	r, g, b, _ := d.Model().View().BackgroundColor.RGBA()
	if r>>8 != 0x2E || g>>8 != 0x34 || b>>8 != 0x40 {
		t.Errorf("background = #%02X%02X%02X, want nord's #2E3440", r>>8, g>>8, b>>8)
	}
	if strings.Contains(d.Raw(), "\x1b[48;2;46;52;64m") {
		t.Error("by default the cells are not painted; the terminal's own background is used")
	}
}

func TestBackgroundFollowsThePalette(t *testing.T) {
	f := newFixture(t)
	f.cfg.Getenv = env("TERM", "xterm")
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "d", Run: func(c *ctx) tea.Cmd { c.PreviewPalette("dracula"); return nil }})
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "t", Run: func(c *ctx) tea.Cmd { c.PreviewPalette("default"); return nil }})
	d := f.driver(120, 40)
	if bg, _ := backdropOf(t, d); bg {
		t.Error("the default palette keeps the terminal's colours")
	}
	d.Press("d")
	if bg, _ := backdropOf(t, d); !bg {
		t.Error("switching to dracula should set a background")
	}
	d.Press("t")
	if bg, _ := backdropOf(t, d); bg {
		t.Error("switching back should release it")
	}
}

func TestBackgroundOffAndTTYNeverSetIt(t *testing.T) {
	off := newFixture(t)
	off.cfg.Palette = "nord"
	off.cfg.Background = dashygotui.BackgroundOff
	if bg, fg := backdropOf(t, off.driver(120, 40)); bg || fg {
		t.Error("BackgroundOff must not set colours")
	}
	tty := newFixture(t)
	tty.cfg.Palette = "nord"
	tty.cfg.Glyphs = dashygotui.GlyphsTTY
	if bg, fg := backdropOf(t, tty.driver(120, 40)); bg || fg {
		t.Error("the Linux console cannot take a background")
	}
}

func TestPaintModeColoursEveryCell(t *testing.T) {
	f := newFixture(t)
	f.cfg.Palette = "nord"
	f.cfg.Background = dashygotui.BackgroundPaint
	f.cfg.Getenv = env("TERM", "xterm")
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "o", Run: func(c *ctx) tea.Cmd { c.OpenModal(&stubModal{}); return nil }})
	d := f.driver(120, 40)
	const bg = "\x1b[48;2;46;52;64m"
	for i, l := range strings.Split(d.Raw(), "\n") {
		if !strings.HasPrefix(l, bg) {
			t.Fatalf("row %d does not start with the background", i)
		}
		if w := text.Width(l); w != 120 {
			t.Fatalf("row %d is %d wide after painting", i, w)
		}
	}
	d.Press("tab")
	if !strings.Contains(d.Raw(), bg+"\x1b[38;2;216;222;233m") {
		t.Error("the foreground should be applied with the background")
	}
	// And it works over a modal, so the overlay is painted too.
	m := f.driver(120, 40)
	m.Press("o")
	m.RequireContains("Really do it?")
	for i, l := range strings.Split(m.Raw(), "\n") {
		if !strings.HasPrefix(l, bg) {
			t.Fatalf("modal frame row %d does not start with the background", i)
		}
	}
}

func TestConsolePaletteIsOffered(t *testing.T) {
	f := newFixture(t)
	var ids []string
	f.reg.Shortcut(dashygotui.Shortcut[state]{Key: "p", Run: func(c *ctx) tea.Cmd {
		for _, p := range c.Palettes() {
			ids = append(ids, p.ID)
		}
		return nil
	}})
	d := f.driver(120, 40)
	d.Press("p")
	if got := strings.Join(ids, ","); got != "default,catppuccin,dracula,nord,"+theme.Console().ID {
		t.Errorf("palettes = %s", got)
	}
}
