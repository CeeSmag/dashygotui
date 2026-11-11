package theme_test

import (
	"strings"
	"testing"

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

func TestDetectASCII(t *testing.T) {
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
		{"explicit true", map[string]string{"DASHYGOTUI_ASCII": "TRUE"}, true},
		{"explicit 0", map[string]string{"DASHYGOTUI_ASCII": "0", "TERM": "xterm"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := theme.DetectASCII(env(tt.env)); got != tt.want {
				t.Errorf("DetectASCII = %v, want %v", got, tt.want)
			}
		})
	}
}
