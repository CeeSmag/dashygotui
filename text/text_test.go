package text_test

import (
	"math"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/text"
)

const (
	red   = "\x1b[31m"
	reset = "\x1b[0m"
)

func TestWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "hello", 5},
		{"ansi is zero width", red + "hello" + reset, 5},
		{"cjk is double width", "日本", 4},
		{"mixed", "a日b", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := text.Width(tt.in); got != tt.want {
				t.Errorf("Width(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestTrunc(t *testing.T) {
	tests := []struct {
		name string
		in   string
		w    int
		want string
	}{
		{"fits", "hello", 5, "hello"},
		{"cut with ellipsis", "hello world", 6, "hello…"},
		{"width one", "hello", 1, "…"},
		{"zero", "hello", 0, ""},
		{"negative", "hello", -3, ""},
		{"wide rune not split", "日本語", 5, "日本…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := text.Trunc(tt.in, tt.w); got != tt.want {
				t.Errorf("Trunc(%q, %d) = %q, want %q", tt.in, tt.w, got, tt.want)
			}
		})
	}
}

func TestFitIsExactlyWide(t *testing.T) {
	inputs := []string{"", "a", "hello world", red + "styled text here" + reset, "日本語のテキスト", "😀 emoji 😀"}
	for _, in := range inputs {
		for w := 0; w <= 30; w++ {
			got := text.Fit(in, w)
			if text.Width(got) != w {
				t.Errorf("Fit(%q, %d) has width %d, want %d", in, w, text.Width(got), w)
			}
		}
	}
}

func TestPad(t *testing.T) {
	tests := []struct {
		name string
		in   string
		w    int
		want string
	}{
		{"pads", "ab", 5, "ab   "},
		{"exact", "abc", 3, "abc"},
		{"wider is untouched", "abcdef", 3, "abcdef"},
		{"counts cells not bytes", "日", 4, "日  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := text.Pad(tt.in, tt.w); got != tt.want {
				t.Errorf("Pad(%q, %d) = %q, want %q", tt.in, tt.w, got, tt.want)
			}
		})
	}
}

func TestCenter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		w    int
		want string
	}{
		{"even", "ab", 6, "  ab  "},
		{"odd leaves extra on the right", "ab", 5, " ab  "},
		{"too wide is truncated", "abcdefgh", 4, "abc…"},
		{"zero", "ab", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := text.Center(tt.in, tt.w); got != tt.want {
				t.Errorf("Center(%q, %d) = %q, want %q", tt.in, tt.w, got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	t.Run("non-positive width", func(t *testing.T) {
		if got := text.Wrap("abc", 0); got != nil {
			t.Errorf("Wrap(_, 0) = %v, want nil", got)
		}
	})
	t.Run("every line fits", func(t *testing.T) {
		in := "the quick brown fox jumps over the lazy dog " + strings.Repeat("x", 40)
		for w := 1; w <= 30; w++ {
			for _, line := range text.Wrap(in, w) {
				if text.Width(line) > w {
					t.Errorf("Wrap(w=%d) produced %q (width %d)", w, line, text.Width(line))
				}
			}
		}
	})
	t.Run("breaks at spaces", func(t *testing.T) {
		got := text.Wrap("one two three", 7)
		want := []string{"one two", "three"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("Wrap = %q, want %q", got, want)
		}
	})
}

func TestSlice(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		left, right int
		want        string
	}{
		{"middle", "abcdef", 1, 4, "bcd"},
		{"empty range", "abcdef", 3, 3, ""},
		{"reversed range", "abcdef", 4, 1, ""},
		{"past the end", "abc", 1, 99, "bc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := text.Slice(tt.in, tt.left, tt.right); got != tt.want {
				t.Errorf("Slice(%q, %d, %d) = %q, want %q", tt.in, tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestOneLine(t *testing.T) {
	if got, want := text.OneLine("a\n b\t\tc \r\n d "), "a b c d"; got != want {
		t.Errorf("OneLine = %q, want %q", got, want)
	}
}

func TestBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{-5, "0 B"},
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1 << 20, "1.0 MB"},
		{5 << 30, "5.0 GB"},
		{math.MaxInt64, "8192.0 PB"},
	}
	for _, tt := range tests {
		if got := text.Bytes(tt.in); got != tt.want {
			t.Errorf("Bytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRate(t *testing.T) {
	tests := []struct {
		in          float64
		long, short string
	}{
		{math.NaN(), "0 B/s", "0/s"},
		{-1, "0 B/s", "0/s"},
		{999, "999 B/s", "999/s"},
		{1500, "1.5 KB/s", "1.5K/s"},
		{12_400_000, "12.4 MB/s", "12M/s"},
		{5e6, "5.0 MB/s", "5M/s"},
	}
	for _, tt := range tests {
		if got := text.Rate(tt.in); got != tt.long {
			t.Errorf("Rate(%v) = %q, want %q", tt.in, got, tt.long)
		}
		if got := text.RateShort(tt.in); got != tt.short {
			t.Errorf("RateShort(%v) = %q, want %q", tt.in, got, tt.short)
		}
	}
}
