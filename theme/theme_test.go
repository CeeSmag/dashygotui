package theme_test

import (
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

func TestBuiltinsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	palettes := append(theme.Builtins(), theme.Console())
	for _, p := range palettes {
		t.Run(p.ID, func(t *testing.T) {
			if p.ID == "" || p.ID != strings.ToLower(p.ID) {
				t.Errorf("ID %q must be non-empty lower case", p.ID)
			}
			if seen[p.ID] {
				t.Errorf("duplicate ID %q", p.ID)
			}
			seen[p.ID] = true
			if p.Name == "" || p.Vibe == "" {
				t.Errorf("palette %q needs a Name and a Vibe", p.ID)
			}
			for field, c := range map[string]string{
				"Border": p.Border, "Cyan": p.Cyan, "Green": p.Green, "Mint": p.Mint,
				"Amber": p.Amber, "Orange": p.Orange, "Red": p.Red, "Muted": p.Muted,
				"White": p.White, "Track": p.Track,
			} {
				if c == "" {
					t.Errorf("%s is empty", field)
				}
			}
		})
	}
}

func TestBuiltinsStartWithDefault(t *testing.T) {
	if got := theme.Builtins()[0].ID; got != theme.DefaultID {
		t.Errorf("first builtin = %q, want %q", got, theme.DefaultID)
	}
}

func TestBuiltinsReturnsIndependentCopies(t *testing.T) {
	a := theme.Builtins()
	a[0].Name = "mutated"
	if theme.Builtins()[0].Name == "mutated" {
		t.Error("Builtins shares state between calls")
	}
}

func TestLookup(t *testing.T) {
	palettes := theme.Builtins()
	tests := []struct {
		id     string
		wantID string
		wantOK bool
	}{
		{"nord", "nord", true},
		{"  NORD ", "nord", true},
		{"Dracula", "dracula", true},
		{"missing", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			p, ok := theme.Lookup(palettes, tt.id)
			if ok != tt.wantOK || p.ID != tt.wantID {
				t.Errorf("Lookup(%q) = (%q, %v), want (%q, %v)", tt.id, p.ID, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestGradientRGB(t *testing.T) {
	p := theme.Default()
	th := theme.New(p, false)
	tests := []struct {
		name string
		v    float64
		want [3]uint8
	}{
		{"below zero clamps to low", -5, p.GradLow},
		{"zero is low", 0, p.GradLow},
		{"half is mid", 0.5, p.GradMid},
		{"0.8 is warn", 0.8, p.GradWarn},
		{"one is peak", 1, p.GradPeak},
		{"above one clamps to peak", 7, p.GradPeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b := th.GradientRGB(tt.v)
			if got := [3]uint8{r, g, b}; got != tt.want {
				t.Errorf("GradientRGB(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
	t.Run("interpolates between stops", func(t *testing.T) {
		r, g, b := th.GradientRGB(0.25)
		for i, got := range [3]uint8{r, g, b} {
			want := (int(p.GradLow[i]) + int(p.GradMid[i])) / 2 // midway between the first two stops
			if d := int(got) - want; d < -1 || d > 1 {
				t.Errorf("channel %d = %d, want about %d", i, got, want)
			}
		}
	})
}

func TestSpinnerAlwaysYieldsAFrame(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := theme.New(theme.Default(), ascii)
		for n := -25; n <= 25; n++ {
			if th.Spinner(n) == "" {
				t.Fatalf("ascii=%v Spinner(%d) is empty", ascii, n)
			}
		}
	}
	ascii := theme.New(theme.Default(), true)
	if got := ascii.Spinner(0) + ascii.Spinner(1); got != "|/" {
		t.Errorf("ascii frames = %q, want %q", got, "|/")
	}
}

func TestPointer(t *testing.T) {
	if got := theme.New(theme.Default(), true).Pointer(); got != ">" {
		t.Errorf("ascii pointer = %q", got)
	}
	if got := theme.New(theme.Default(), false).Pointer(); got != "❯" {
		t.Errorf("unicode pointer = %q", got)
	}
}

func TestSwatchContainsBrackets(t *testing.T) {
	s := theme.Swatch(theme.Default())
	if !strings.Contains(s, "[") || !strings.Contains(s, "]") || strings.Count(s, "■") != 6 {
		t.Errorf("Swatch = %q, want bracketed 6-cell preview", s)
	}
}

func TestDetectTTY(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"empty", nil, false},
		{"xterm", map[string]string{"TERM": "xterm-256color"}, false},
		{"linux console", map[string]string{"TERM": "linux"}, true},
		{"linux console, odd case", map[string]string{"TERM": " Linux "}, true},
		{"explicit 1", map[string]string{"DASHYGOTUI_ASCII": "1"}, true},
		{"explicit tty", map[string]string{"DASHYGOTUI_TTY": "yes"}, true},
		{"explicit true", map[string]string{"DASHYGOTUI_ASCII": "TRUE"}, true},
		{"explicit 0", map[string]string{"DASHYGOTUI_ASCII": "0", "TERM": "xterm"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := theme.DetectTTY(env(tt.env)); got != tt.want {
				t.Errorf("DetectASCII = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlyphSetsAreSingleCellAndComplete(t *testing.T) {
	for name, g := range map[string]theme.Glyphs{"unicode": theme.UnicodeGlyphs(), "tty": theme.TTYGlyphs()} {
		t.Run(name, func(t *testing.T) {
			single := map[string]string{
				"CornerTL": g.CornerTL, "CornerTR": g.CornerTR, "CornerBL": g.CornerBL, "CornerBR": g.CornerBR,
				"Pointer": g.Pointer, "Prompt": g.Prompt, "Bullet": g.Bullet, "Dot": g.Dot, "Sep": g.Sep,
				"Check": g.Check, "Cross": g.Cross, "Up": g.Up, "Down": g.Down, "Ellipsis": g.Ellipsis,
				"EchoChar": g.EchoChar, "MeterFill": g.MeterFill, "MeterEmpty": g.MeterEmpty,
				"SliderKnob": g.SliderKnob, "SliderFill": g.SliderFill, "SliderRail": g.SliderRail,
			}
			for field, v := range single {
				if text.Width(v) != 1 {
					t.Errorf("%s = %q is %d cells wide, want exactly 1", field, v, text.Width(v))
				}
			}
			for field, v := range map[string]string{"UpDown": g.UpDown, "LeftRight": g.LeftRight, "Enter": g.Enter, "Both": g.Both} {
				if v == "" {
					t.Errorf("%s is empty", field)
				}
			}
			if len(g.Ramp) != 8 {
				t.Errorf("Ramp has %d entries, want 8", len(g.Ramp))
			}
			for _, r := range g.Ramp {
				if text.Width(r) != 1 {
					t.Errorf("ramp glyph %q is not one cell wide", r)
				}
			}
			if len(g.Spinner) == 0 {
				t.Error("no spinner frames")
			}
			for _, f := range g.Spinner {
				if text.Width(f) != 1 {
					t.Errorf("spinner frame %q is not one cell wide", f)
				}
			}
		})
	}
}

func TestTTYGlyphsStayInsideTheConsoleFont(t *testing.T) {
	g := theme.TTYGlyphs()
	all := strings.Join([]string{
		g.CornerTL, g.CornerTR, g.CornerBL, g.CornerBR, g.Pointer, g.Prompt, g.Bullet, g.Dot, g.Sep,
		g.Check, g.Cross, g.Up, g.Down, g.Both, g.Ellipsis, g.UpDown, g.LeftRight, g.Enter, g.EchoChar,
		g.MeterFill, g.MeterEmpty, g.SliderKnob, g.SliderFill, g.SliderRail,
		strings.Join(g.Spinner, ""), strings.Join(g.Ramp, ""),
	}, "")
	// ASCII plus the straight line-drawing corners.
	for _, r := range all {
		if r > 0x7e && !strings.ContainsRune("┌┐└┘", r) {
			t.Errorf("TTY glyph %q (U+%04X) is outside ASCII and the corner set", r, r)
		}
	}
}

func TestNewChoosesTheGlyphSet(t *testing.T) {
	if th := theme.New(theme.Default(), false); th.G.CornerTL != "╭" || th.ASCII {
		t.Errorf("unicode theme = %+v", th.G)
	}
	if th := theme.New(theme.Default(), true); th.G.CornerTL != "┌" || !th.ASCII {
		t.Errorf("tty theme = %+v", th.G)
	}
	custom := theme.UnicodeGlyphs()
	custom.Pointer = "=>"
	if th := theme.NewWithGlyphs(theme.Default(), custom); th.Pointer() != "=>" {
		t.Error("NewWithGlyphs must use the glyphs it is given")
	}
}

func TestThemeEllipsisFollowsTheGlyphs(t *testing.T) {
	long := "a very long label"
	if got := theme.New(theme.Default(), false).Trunc(long, 6); got != "a ver…" {
		t.Errorf("unicode Trunc = %q", got)
	}
	if got := theme.New(theme.Default(), true).Trunc(long, 6); got != "a ver~" {
		t.Errorf("tty Trunc = %q", got)
	}
	if got := text.Width(theme.New(theme.Default(), true).Fit(long, 9)); got != 9 {
		t.Errorf("Fit width = %d", got)
	}
}

func TestBackdrop(t *testing.T) {
	nord, _ := theme.Lookup(theme.Builtins(), "nord")
	bg, fg, ok := theme.New(nord, false).Backdrop()
	if !ok || bg == nil || fg == nil {
		t.Fatalf("nord should have a backdrop: %v %v %v", bg, fg, ok)
	}
	r, g, b, _ := bg.RGBA()
	if r>>8 != 0x2E || g>>8 != 0x34 || b>>8 != 0x40 {
		t.Errorf("background = #%02X%02X%02X, want #2E3440", r>>8, g>>8, b>>8)
	}
	if _, _, ok := theme.New(plainPalette(), false).Backdrop(); ok {
		t.Error("a palette without a background keeps the terminal's own colours")
	}
	if _, _, ok := theme.New(nord, true).Backdrop(); ok {
		t.Error("TTY mode must not set a background")
	}
	bad := nord
	bad.Background = "blue"
	if _, _, ok := theme.New(bad, false).Backdrop(); ok {
		t.Error("only #RRGGBB is accepted")
	}
}

func TestPaintSurvivesStyleResetsAndKeepsWidth(t *testing.T) {
	nord, _ := theme.Lookup(theme.Builtins(), "nord")
	th := theme.New(nord, false)
	lines := []string{
		th.Green.Render("ok") + " plain " + th.Red.Render("bad") + " tail",
		"unstyled",
		"",
	}
	got := th.Paint(lines)
	if len(got) != len(lines) {
		t.Fatalf("%d lines, want %d", len(got), len(lines))
	}
	const bg = "\x1b[48;2;46;52;64m"
	for i, l := range got {
		if !strings.HasPrefix(l, bg) {
			t.Errorf("line %d does not start with the background: %q", i, l)
		}
		if text.Width(l) != text.Width(lines[i]) {
			t.Errorf("line %d changed width: %d -> %d", i, text.Width(lines[i]), text.Width(l))
		}
		if strings.HasSuffix(l, bg) {
			t.Errorf("line %d ends by re-opening the background, leaking it past the line", i)
		}
	}
	// Every reset in the first line is followed by the background again.
	if n, m := strings.Count(got[0], "\x1b[m"), strings.Count(got[0], "\x1b[m"+bg); n-1 > m {
		t.Errorf("only %d of %d resets re-apply the background: %q", m, n, got[0])
	}
	if same := theme.New(plainPalette(), false).Paint(lines); strings.Join(same, "") != strings.Join(lines, "") {
		t.Error("a palette without a background must leave lines untouched")
	}
}

// plainPalette is the default palette with no background, like an application's
// own palette that leaves the terminal's colours alone.
func plainPalette() theme.Palette {
	p := theme.Default()
	p.Background, p.Foreground = "", ""
	return p
}

func TestEveryPaletteExceptTheConsoleSetsABackground(t *testing.T) {
	for _, p := range theme.Builtins() {
		if p.Background == "" || p.Foreground == "" {
			t.Errorf("%s has no background and foreground: the app would have none out of the box", p.ID)
		}
		if _, _, ok := theme.New(p, false).Backdrop(); !ok {
			t.Errorf("%s: Backdrop rejects its own colours (%q, %q)", p.ID, p.Background, p.Foreground)
		}
		if p.Background == p.Foreground {
			t.Errorf("%s: text would be invisible", p.ID)
		}
	}
	if c := theme.Console(); c.Background != "" || c.Foreground != "" {
		t.Error("the 16-colour console palette cannot take a background")
	}
}

func TestTheDefaultBackgroundIsDark(t *testing.T) {
	bg, fg, ok := theme.New(theme.Default(), false).Backdrop()
	if !ok {
		t.Fatal("the default palette sets a background")
	}
	lum := func(c interface{ RGBA() (r, g, b, a uint32) }) uint32 {
		r, g, b, _ := c.RGBA()
		return (2126*(r>>8) + 7152*(g>>8) + 722*(b>>8)) / 10000
	}
	if lum(bg) > 60 || lum(fg) < 180 {
		t.Errorf("want a dark background and a light foreground, got luminance %d and %d", lum(bg), lum(fg))
	}
}
