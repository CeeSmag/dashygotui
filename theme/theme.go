package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme is the set of styles derived from one [Palette]. It is immutable once
// built; to change the look, build a new Theme with [New].
type Theme struct {
	// Palette is the palette the styles were built from.
	Palette Palette

	// ASCII selects plain glyphs (shade blocks, ">" and "|/-\") for terminals
	// without Braille or box-drawing arrow support, such as the Linux console.
	ASCII bool

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

// New builds a Theme from p. Set ascii to restrict output to glyphs that
// exist in every console font.
func New(p Palette, ascii bool) *Theme {
	fg := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	return &Theme{
		Palette: p,
		ASCII:   ascii,
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
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	if t.ASCII {
		frames = []string{"|", "/", "-", "\\"}
	}
	return frames[((n%len(frames))+len(frames))%len(frames)]
}

// Pointer is the cursor glyph for the selected row of a menu.
func (t *Theme) Pointer() string {
	if t.ASCII {
		return ">"
	}
	return "❯"
}

// Swatch renders a bracketed six-colour preview of p, for palette pickers.
func Swatch(p Palette) string {
	st := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	const cell = "■"
	return st(p.Border).Render("[") +
		st(p.Border).Render(cell) + st(p.Cyan).Render(cell) + st(p.Green).Render(cell) +
		st(p.Amber).Render(cell) + st(p.Red).Render(cell) + st(p.White).Render(cell) +
		st(p.Border).Render("]")
}

// DetectASCII reports whether the environment looks like a terminal that needs
// plain glyphs: the Linux console (TERM=linux) or an explicit
// DASHYGOTUI_ASCII=1|true. getenv is typically os.Getenv; taking it as a
// parameter keeps the function pure and testable.
func DetectASCII(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv("DASHYGOTUI_ASCII"))) {
	case "1", "true", "yes":
		return true
	}
	return strings.EqualFold(strings.TrimSpace(getenv("TERM")), "linux")
}
