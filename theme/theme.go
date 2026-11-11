package theme

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/CeeSmag/dashygotui/text"
)

// Theme is the set of styles and glyphs derived from one [Palette]. It is
// immutable once built; to change the look, build a new Theme with [New].
type Theme struct {
	// Palette is the palette the styles were built from.
	Palette Palette

	// ASCII reports TTY mode: the theme was built with the conservative
	// [TTYGlyphs] and graphs use shade blocks instead of Braille. It is the
	// switch for terminals with a small font, such as the Linux console.
	ASCII bool

	// G is the symbol set renderers draw with.
	G Glyphs

	Border lipgloss.Style
	Title  lipgloss.Style
	Cyan   lipgloss.Style
	Key    lipgloss.Style
	Green  lipgloss.Style
	Mint   lipgloss.Style
	Amber  lipgloss.Style
	Orange lipgloss.Style
	Red    lipgloss.Style
	Muted  lipgloss.Style
	White  lipgloss.Style
	Track  lipgloss.Style
	Cursor lipgloss.Style
}

// New builds a Theme from p. With tty set it uses [TTYGlyphs] and shade-block
// graphs, otherwise [UnicodeGlyphs].
func New(p Palette, tty bool) *Theme {
	g := UnicodeGlyphs()
	if tty {
		g = TTYGlyphs()
	}
	t := NewWithGlyphs(p, g)
	t.ASCII = tty
	return t
}

// NewWithGlyphs builds a Theme from p drawing with g. ASCII stays false; set it
// on the result when g is a restricted set and graphs should use shade blocks.
func NewWithGlyphs(p Palette, g Glyphs) *Theme {
	fg := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	return &Theme{
		Palette: p,
		G:       g,
		Border:  fg(p.Border),
		Title:   fg(p.Cyan).Bold(true),
		Cyan:    fg(p.Cyan),
		Key:     fg(p.Cyan).Bold(true),
		Green:   fg(p.Green).Bold(true),
		Mint:    fg(p.Mint),
		Amber:   fg(p.Amber),
		Orange:  fg(p.Orange),
		Red:     fg(p.Red).Bold(true),
		Muted:   fg(p.Muted),
		White:   fg(p.White).Bold(true),
		Track:   fg(p.Track),
		Cursor:  fg(p.White).Bold(true),
	}
}

// Trunc is [text.Trunc] using the theme's ellipsis.
func (t *Theme) Trunc(s string, w int) string { return text.TruncWith(s, w, t.G.Ellipsis) }

// Fit is [text.Fit] using the theme's ellipsis.
func (t *Theme) Fit(s string, w int) string { return text.FitWith(s, w, t.G.Ellipsis) }

// GradientRGB maps v in [0, 1] onto the meter and graph ramp of the palette:
// calm at 0, through the middle and warning stops, to the peak colour at 1.
// Values outside [0, 1] are clamped.
func (t *Theme) GradientRGB(v float64) (r, g, b uint8) {
	p := t.Palette
	switch {
	case v <= 0:
		return p.GradLow[0], p.GradLow[1], p.GradLow[2]
	case v >= 1:
		return p.GradPeak[0], p.GradPeak[1], p.GradPeak[2]
	case v < 0.5:
		return lerp(p.GradLow, p.GradMid, v/0.5)
	case v < 0.8:
		return lerp(p.GradMid, p.GradWarn, (v-0.5)/0.3)
	default:
		return lerp(p.GradWarn, p.GradPeak, (v-0.8)/0.2)
	}
}

// Gradient returns a style coloured by [Theme.GradientRGB].
func (t *Theme) Gradient(v float64) lipgloss.Style {
	r, g, b := t.GradientRGB(v)
	return lipgloss.NewStyle().Foreground(color.RGBA{R: r, G: g, B: b, A: 0xff})
}

func lerp(a, b [3]uint8, t float64) (uint8, uint8, uint8) {
	mix := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5)
	}
	return mix(a[0], b[0]), mix(a[1], b[1]), mix(a[2], b[2])
}

// Spinner returns the frame of the "live" spinner for tick n. Any n, including
// a negative one, selects a valid frame.
func (t *Theme) Spinner(n int) string {
	frames := t.G.Spinner
	if len(frames) == 0 {
		return ""
	}
	return frames[((n%len(frames))+len(frames))%len(frames)]
}

// Pointer is the cursor glyph for the selected row of a menu.
func (t *Theme) Pointer() string { return t.G.Pointer }

// Swatch renders a bracketed six-colour preview of p, for palette pickers.
func Swatch(p Palette) string { return SwatchWith(p, "■") }

// SwatchWith is [Swatch] drawing each colour with the given one-cell glyph.
func SwatchWith(p Palette, cell string) string {
	st := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	return st(p.Border).Render("[") +
		st(p.Border).Render(cell) + st(p.Cyan).Render(cell) + st(p.Green).Render(cell) +
		st(p.Amber).Render(cell) + st(p.Red).Render(cell) + st(p.White).Render(cell) +
		st(p.Border).Render("]")
}

// DetectTTY reports whether the environment looks like a terminal that needs
// the restricted [TTYGlyphs]: the Linux console (TERM=linux), or an explicit
// DASHYGOTUI_TTY=1|true (DASHYGOTUI_ASCII is accepted too). getenv is typically
// os.Getenv; taking it as a parameter keeps the function pure and testable.
func DetectTTY(getenv func(string) string) bool {
	for _, name := range []string{"DASHYGOTUI_TTY", "DASHYGOTUI_ASCII"} {
		switch strings.ToLower(strings.TrimSpace(getenv(name))) {
		case "1", "true", "yes":
			return true
		}
	}
	return strings.EqualFold(strings.TrimSpace(getenv("TERM")), "linux")
}

// Backdrop returns the colours the terminal itself should use so the whole
// screen matches the palette, and whether the palette sets a background. In TTY
// mode it reports false: the Linux console cannot take them.
func (t *Theme) Backdrop() (bg, fg color.Color, ok bool) {
	if t.ASCII {
		return nil, nil, false
	}
	br, bgr, bb, bok := hexRGB(t.Palette.Background)
	if !bok {
		return nil, nil, false
	}
	bg = color.RGBA{R: br, G: bgr, B: bb, A: 0xff}
	if fr, fgr, fb, fok := hexRGB(t.Palette.Foreground); fok {
		fg = color.RGBA{R: fr, G: fgr, B: fb, A: 0xff}
	}
	return bg, fg, true
}

// Paint gives every cell of the frame the palette's background (and default
// foreground) by writing the colour into the text itself. Use it for terminals
// that ignore the escape sequence which sets the terminal's own background. It
// re-applies the colour after every style reset, so it survives styled text,
// and does not change the width of any line. It returns lines unchanged when
// the palette has no background or the theme is in TTY mode.
func (t *Theme) Paint(lines []string) []string {
	bg, fg, ok := t.Backdrop()
	if !ok {
		return lines
	}
	br, bgr, bb, _ := bg.RGBA()
	seq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", br>>8, bgr>>8, bb>>8)
	if fg != nil {
		fr, fgr, fb, _ := fg.RGBA()
		seq += fmt.Sprintf("\x1b[38;2;%d;%d;%dm", fr>>8, fgr>>8, fb>>8)
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		l = strings.ReplaceAll(l, "\x1b[m", "\x1b[m"+seq)
		l = strings.ReplaceAll(l, "\x1b[0m", "\x1b[0m"+seq)
		out[i] = seq + l + "\x1b[m"
	}
	return out
}

// hexRGB parses "#RRGGBB".
func hexRGB(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimSpace(s)
	if len(s) != 7 || s[0] != '#' {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), true
}
