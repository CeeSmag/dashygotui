// Package theme defines colour palettes and the immutable [Theme] of styles
// derived from one.
//
// Nothing here is global: a [Theme] is a value created from a [Palette] and
// handed to every renderer that colours output, so two dashboards in one
// process (or two tests running in parallel) never interfere.
package theme

import "strings"

// Palette is a named set of colours. The colour strings are anything lipgloss
// accepts: "#RRGGBB" hex or an ANSI colour number such as "14".
type Palette struct {
	ID     string // lower-case unique key, e.g. "nord"
	Name   string // human name, e.g. "Nord"
	Vibe   string // one-line description shown in pickers
	Border string
	Cyan   string
	Green  string
	Mint   string
	Amber  string
	Orange string
	Red    string
	Muted  string
	White  string
	Track  string // empty part of a meter or slider

	// Background and Foreground, when set, are the colours the terminal itself
	// should use ("#RRGGBB" only), so the whole screen matches the palette. The
	// dashboard applies them when the terminal can show them. Leave empty to keep
	// the terminal's own colours.
	Background string
	Foreground string

	// The four RGB stops of the meter and graph ramp, from calm to hot.
	GradLow  [3]uint8
	GradMid  [3]uint8
	GradWarn [3]uint8
	GradPeak [3]uint8
}

// DefaultID is the ID of the palette [Default] returns.
const DefaultID = "default"

// Default is the built-in electric-cyan palette, on a dark slate background.
func Default() Palette {
	return Palette{
		ID:     DefaultID,
		Name:   "Teal",
		Vibe:   "Classic electric cyan frame with Tailwind accents",
		Border: "#00B4D8",
		Cyan:   "#38BDF8",
		Green:  "#22C55E",
		Mint:   "#4ADE80",
		Amber:  "#FBBF24",
		Orange: "#F97316",
		Red:    "#EF4444",
		Muted:  "#94A3B8",
		White:  "#F8FAFC",
		Track:  "#334155",
		// A dark slate that the accents above were chosen against.
		Background: "#0F172A",
		Foreground: "#E2E8F0",
		GradLow:    [3]uint8{56, 189, 248},
		GradMid:    [3]uint8{34, 197, 94},
		GradWarn:   [3]uint8{245, 158, 11},
		GradPeak:   [3]uint8{239, 68, 68},
	}
}

// Console is the native 16-colour ANSI palette, for Linux virtual consoles and
// serial terminals that cannot show true colour. It is not part of [Builtins]
// because a user would not choose it from a picker; use it with
// [DetectASCII].
func Console() Palette {
	return Palette{
		ID:       "console",
		Name:     "Console (16 colours)",
		Vibe:     "Native 16-colour ANSI palette for the Linux console and serial terminals",
		Border:   "6",
		Cyan:     "14",
		Green:    "10",
		Mint:     "2",
		Amber:    "11",
		Orange:   "3",
		Red:      "9",
		Muted:    "8",
		White:    "15",
		Track:    "0",
		GradLow:  [3]uint8{0, 255, 255},
		GradMid:  [3]uint8{0, 255, 0},
		GradWarn: [3]uint8{255, 255, 0},
		GradPeak: [3]uint8{255, 0, 0},
	}
}

// Builtins returns the palettes shipped with the library, in the order a
// picker should list them. The first is [Default].
func Builtins() []Palette {
	return []Palette{
		Default(),
		{
			ID:         "catppuccin",
			Name:       "Catppuccin Mocha",
			Vibe:       "Soothing pastel dark aesthetic with soft accents",
			Border:     "#89B4FA",
			Cyan:       "#89DCEB",
			Green:      "#A6E3A1",
			Mint:       "#94E2D5",
			Amber:      "#F9E2AF",
			Orange:     "#FAB387",
			Red:        "#F38BA8",
			Muted:      "#7F849C",
			White:      "#CDD6F4",
			Track:      "#313244",
			Background: "#1E1E2E",
			Foreground: "#CDD6F4",
			GradLow:    [3]uint8{137, 220, 235},
			GradMid:    [3]uint8{166, 227, 161},
			GradWarn:   [3]uint8{249, 226, 175},
			GradPeak:   [3]uint8{243, 139, 168},
		},
		{
			ID:         "dracula",
			Name:       "Dracula",
			Vibe:       "Vibrant high-contrast gothic dark theme",
			Border:     "#BD93F9",
			Cyan:       "#8BE9FD",
			Green:      "#50FA7B",
			Mint:       "#69FF94",
			Amber:      "#F1FA8C",
			Orange:     "#FFB86C",
			Red:        "#FF5555",
			Muted:      "#6272A4",
			White:      "#F8F8F2",
			Track:      "#44475A",
			Background: "#282A36",
			Foreground: "#F8F8F2",
			GradLow:    [3]uint8{139, 233, 253},
			GradMid:    [3]uint8{80, 250, 123},
			GradWarn:   [3]uint8{241, 250, 140},
			GradPeak:   [3]uint8{255, 85, 85},
		},
		{
			ID:         "nord",
			Name:       "Nord",
			Vibe:       "Arctic minimalist palette of cool frost blues",
			Border:     "#81A1C1",
			Cyan:       "#88C0D0",
			Green:      "#A3BE8C",
			Mint:       "#8FBCBB",
			Amber:      "#EBCB8B",
			Orange:     "#D08770",
			Red:        "#BF616A",
			Muted:      "#616E88",
			White:      "#ECEFF4",
			Track:      "#3B4252",
			Background: "#2E3440",
			Foreground: "#D8DEE9",
			GradLow:    [3]uint8{136, 192, 208},
			GradMid:    [3]uint8{163, 190, 140},
			GradWarn:   [3]uint8{235, 203, 139},
			GradPeak:   [3]uint8{191, 97, 106},
		},
	}
}

// Lookup finds the palette with the given ID in palettes. IDs are compared
// case-insensitively and ignoring surrounding space.
func Lookup(palettes []Palette, id string) (Palette, bool) {
	want := normalizeID(id)
	for _, p := range palettes {
		if normalizeID(p.ID) == want {
			return p, true
		}
	}
	return Palette{}, false
}

func normalizeID(id string) string { return strings.ToLower(strings.TrimSpace(id)) }
