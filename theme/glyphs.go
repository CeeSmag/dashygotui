package theme

// Glyphs is the set of symbols a dashboard draws with. Two sets ship: the rich
// Unicode set, and a TTY set for terminals whose font has few glyphs, chiefly
// the Linux console. An application may build its own.
//
// Every glyph must be exactly one cell wide.
type Glyphs struct {
	// Corners of panel borders. The straight line-drawing characters
	// (─ │ ┌ ┐ └ ┘ ├ ┤) exist in every console font and are used in both sets.
	CornerTL, CornerTR, CornerBL, CornerBR string

	Pointer  string // the selected row of a menu
	Prompt   string // before an input field
	Bullet   string // a status dot
	Dot      string // a separator inside a value, e.g. "0·0·2·0"
	Sep      string // a separator inside a sentence
	Check    string // success
	Cross    string // failure
	Up, Down string // an upload / download arrow
	Both     string // between two related names
	Ellipsis string // marks cut-off text

	// Key names for footer hints.
	UpDown, LeftRight, Enter string

	// EchoChar masks a password.
	EchoChar string

	// Meters and sliders.
	MeterFill, MeterEmpty              string
	SliderKnob, SliderFill, SliderRail string

	// Spinner frames of the "live" indicator.
	Spinner []string
	// Ramp is eight glyphs of increasing height or weight, for sparklines and
	// bar graphs.
	Ramp []string
}

// UnicodeGlyphs is the default, rich symbol set.
func UnicodeGlyphs() Glyphs {
	return Glyphs{
		CornerTL: "╭", CornerTR: "╮", CornerBL: "╰", CornerBR: "╯",
		Pointer: "❯", Prompt: "›", Bullet: "●", Dot: "·", Sep: "•",
		Check: "✓", Cross: "✗", Up: "▲", Down: "▼", Both: "↔", Ellipsis: "…",
		UpDown: "↑↓", LeftRight: "←→", Enter: "↵",
		EchoChar:  "•",
		MeterFill: "■", MeterEmpty: "■",
		SliderKnob: "●", SliderFill: "■", SliderRail: "─",
		Spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Ramp:    []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"},
	}
}

// TTYGlyphs is the conservative set for the Linux console and other terminals
// with a small font: ASCII plus the straight line-drawing characters, which the
// default console font contains.
func TTYGlyphs() Glyphs {
	return Glyphs{
		CornerTL: "┌", CornerTR: "┐", CornerBL: "└", CornerBR: "┘",
		Pointer: ">", Prompt: ">", Bullet: "*", Dot: ".", Sep: "-",
		Check: "+", Cross: "x", Up: "^", Down: "v", Both: "<>", Ellipsis: "~",
		UpDown: "Up/Dn", LeftRight: "</>", Enter: "Enter",
		EchoChar:  "*",
		MeterFill: "#", MeterEmpty: ".",
		SliderKnob: "O", SliderFill: "#", SliderRail: "-",
		Spinner: []string{"|", "/", "-", "\\"},
		Ramp:    []string{".", ":", "-", "=", "+", "*", "#", "%"},
	}
}
