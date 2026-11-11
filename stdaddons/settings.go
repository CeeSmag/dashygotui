package stdaddons

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/modal"
)

// The settings in this file are one-line declarations for the most common kind
// of option: a value the user can change, remembered between runs in the
// dashboard's Store under the setting's Key. Each is an [dashygotui.Addon], so it
// goes straight into [dashygotui.Registry.Add]:
//
//	dashboard.Registry.Add(stdaddons.Number[dashboard.State]{
//		Key: "storage.limit_gb", Section: "settings", Number: 10, Label: "Storage Limit",
//		Unit: "GB", Min: 1, Max: 1000, Default: 50,
//	})
//
// Each gives the option a badge with the current value, a pop-up to change it,
// validation, saving, and a confirmation message. OnChange, optional, runs after
// a change is saved: the place to restart or reconfigure something.

// Toggle is an on/off setting. Opening the row flips it.
type Toggle[S any] struct {
	// Key names the setting in the Store and the option. Required and unique.
	Key string
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Order sorts the option within its section. Optional.
	Order int
	// Label is the row text. Required.
	Label string
	// Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option].
	Hotkey        string
	HotkeyDisplay string
	HotkeyHint    string
	// Default is the value until the user changes it. The zero value is off.
	Default bool
	// OnChange is called with the new value after it was saved. Optional.
	OnChange func(ctx *dashygotui.Context[S], on bool) tea.Cmd
}

// Addons implements [dashygotui.Addon].
func (t Toggle[S]) Addons() []dashygotui.Addon[S] {
	value := func(c *dashygotui.Context[S]) bool {
		on := t.Default
		_, _ = c.Store().Get(t.Key, &on) // a missing or unreadable value keeps the default
		return on
	}
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key: t.Key, Section: t.Section, Number: t.Number, Order: t.Order, Label: t.Label,
		Hotkey: t.Hotkey, HotkeyDisplay: t.HotkeyDisplay, HotkeyHint: t.HotkeyHint,
		Badge: func(c *dashygotui.Context[S]) string { return c.OnOff(value(c)) },
		Run: func(c *dashygotui.Context[S]) tea.Cmd {
			on := !value(c)
			if err := c.Store().Set(t.Key, on); err != nil {
				c.FlashResult("", fmt.Errorf("saving %s: %w", t.Label, err))
				return nil
			}
			c.FlashResult(t.Label+" "+map[bool]string{true: "on", false: "off"}[on]+".", nil)
			if t.OnChange != nil {
				return t.OnChange(c, on)
			}
			return nil
		},
	}}
}

// Number is a whole-number setting, edited in a pop-up that validates the range.
type Number[S any] struct {
	// Key names the setting in the Store and the option. Required and unique.
	Key string
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Order sorts the option within its section. Optional.
	Order int
	// Label is the row text. Required.
	Label string
	// Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option].
	Hotkey        string
	HotkeyDisplay string
	HotkeyHint    string
	// Unit follows the value in the badge and messages, such as "GB". Optional.
	Unit string
	// Min and Max bound the value, inclusive. Max must be greater than Min.
	Min, Max int
	// Default is the value until the user changes it. It should be within the
	// bounds.
	Default int
	// Help explains the setting in the pop-up. It defaults to a sentence naming
	// the bounds.
	Help string
	// OnChange is called with the new value after it was saved. Optional.
	OnChange func(ctx *dashygotui.Context[S], n int) tea.Cmd
}

// Addons implements [dashygotui.Addon].
func (n Number[S]) Addons() []dashygotui.Addon[S] {
	switch {
	case n.Min >= n.Max:
		return invalid[S]("stdaddons.Number %q: Min (%d) must be less than Max (%d)", n.Key, n.Min, n.Max)
	case n.Default < n.Min || n.Default > n.Max:
		return invalid[S]("stdaddons.Number %q: Default (%d) is outside Min..Max (%d..%d)", n.Key, n.Default, n.Min, n.Max)
	}
	value := func(c *dashygotui.Context[S]) int {
		v := n.Default
		_, _ = c.Store().Get(n.Key, &v) // a missing or unreadable value keeps the default
		return v
	}
	show := func(v int) string { return strings.TrimSpace(strconv.Itoa(v) + " " + n.Unit) }
	help := n.Help
	if help == "" {
		help = fmt.Sprintf("A whole number from %d to %d.", n.Min, n.Max)
	}
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key: n.Key, Section: n.Section, Number: n.Number, Order: n.Order, Label: n.Label,
		Hotkey: n.Hotkey, HotkeyDisplay: n.HotkeyDisplay, HotkeyHint: n.HotkeyHint,
		Badge: func(c *dashygotui.Context[S]) string { return c.Badge(show(value(c))) },
		Modal: func(c *dashygotui.Context[S]) dashygotui.Modal[S] {
			st := c.Store()
			var saved int
			return &modal.Edit[S]{
				Title: n.Label, Help: help, Current: show(value(c)), Pending: "Saving...",
				Validate: func(v string) error {
					x, err := strconv.Atoi(v)
					if err != nil || x < n.Min || x > n.Max {
						return fmt.Errorf("%q is not a number from %d to %d", v, n.Min, n.Max)
					}
					return nil
				},
				Save: func(_ context.Context, v string) (string, error) {
					x, err := strconv.Atoi(v)
					if err != nil {
						return "", fmt.Errorf("reading %q: %w", v, err)
					}
					if err := st.Set(n.Key, x); err != nil {
						return "", fmt.Errorf("saving %s: %w", n.Label, err)
					}
					saved = x
					return n.Label + " set to " + show(x) + ".", nil
				},
				OnSaved: func(c *dashygotui.Context[S], _ string) tea.Cmd {
					if n.OnChange == nil {
						return nil
					}
					return n.OnChange(c, saved)
				},
			}
		},
	}}
}

// Pick is a setting chosen from a fixed list, shown in a pop-up with the current
// choice marked. The Store keeps the chosen item's Label.
type Pick[S any] struct {
	// Key names the setting in the Store and the option. Required and unique.
	Key string
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Order sorts the option within its section. Optional.
	Order int
	// Label is the row text. Required.
	Label string
	// Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option].
	Hotkey        string
	HotkeyDisplay string
	HotkeyHint    string
	// Items are the choices; at least one, with distinct labels. An item's Detail
	// explains it under its label.
	Items []modal.Item
	// Default is the Label chosen until the user picks. It defaults to the first
	// item.
	Default string
	// Help explains the setting above the list. Optional.
	Help string
	// OnChange is called with the chosen item's Label after it was saved.
	// Optional.
	OnChange func(ctx *dashygotui.Context[S], label string) tea.Cmd
}

// Addons implements [dashygotui.Addon].
func (p Pick[S]) Addons() []dashygotui.Addon[S] {
	if len(p.Items) == 0 {
		return invalid[S]("stdaddons.Pick %q has no Items to pick from", p.Key)
	}
	seen := map[string]bool{}
	for _, it := range p.Items {
		switch {
		case it.Label == "":
			return invalid[S]("stdaddons.Pick %q has an item with no Label", p.Key)
		case seen[it.Label]:
			return invalid[S]("stdaddons.Pick %q has two items labelled %q: the label is what is stored", p.Key, it.Label)
		}
		seen[it.Label] = true
	}
	if p.Default != "" && !seen[p.Default] {
		return invalid[S]("stdaddons.Pick %q: Default %q is not one of its items", p.Key, p.Default)
	}
	current := func(c *dashygotui.Context[S]) int {
		name := p.Default
		_, _ = c.Store().Get(p.Key, &name) // a missing or unreadable value keeps the default
		for i, it := range p.Items {
			if it.Label == name {
				return i
			}
		}
		return 0
	}
	label := func(c *dashygotui.Context[S]) string {
		if len(p.Items) == 0 {
			return ""
		}
		return p.Items[current(c)].Label
	}
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key: p.Key, Section: p.Section, Number: p.Number, Order: p.Order, Label: p.Label,
		Hotkey: p.Hotkey, HotkeyDisplay: p.HotkeyDisplay, HotkeyHint: p.HotkeyHint,
		Badge: func(c *dashygotui.Context[S]) string { return c.Badge(label(c)) },
		Run: func(c *dashygotui.Context[S]) tea.Cmd {
			c.OpenModal(&modal.Choice[S]{
				Title: p.Label, Help: p.Help, Cursor: current(c),
				Badge: func(c *dashygotui.Context[S]) string { return c.Badge("now " + label(c)) },
				Rows: func(c *dashygotui.Context[S]) []modal.Item {
					items := append([]modal.Item(nil), p.Items...)
					items[current(c)].Badge = c.Theme().Green.Render("[current]")
					return items
				},
				FooterSelect: "Save",
				OnSelect: func(c *dashygotui.Context[S], i int) tea.Cmd {
					if err := c.Store().Set(p.Key, p.Items[i].Label); err != nil {
						c.FlashResult("", fmt.Errorf("saving %s: %w", p.Label, err))
						return nil
					}
					c.FlashResult(p.Label+" set to "+p.Items[i].Label+".", nil)
					if p.OnChange != nil {
						return p.OnChange(c, p.Items[i].Label)
					}
					return nil
				},
			})
			return nil
		},
	}}
}

// Text is a text setting, edited in a one-line pop-up. Secret masks the input and
// hides the value, for tokens and passwords.
type Text[S any] struct {
	// Key names the setting in the Store and the option. Required and unique.
	Key string
	// Section is the section the option appears in. Required.
	Section string
	// Number is the option's number. Required and unique.
	Number int
	// Order sorts the option within its section. Optional.
	Order int
	// Label is the row text. Required.
	Label string
	// Hotkey, HotkeyDisplay and HotkeyHint are those of [dashygotui.Option].
	Hotkey        string
	HotkeyDisplay string
	HotkeyHint    string
	// Default is the value until the user changes it. Optional.
	Default string
	// Help explains the setting in the pop-up. Optional.
	Help string
	// Secret masks the input, and shows only "set" or "not set" as the badge.
	Secret bool
	// Limit is the longest value accepted, in characters. The default is 200.
	Limit int
	// AllowEmpty lets the user save an empty value, which clears the setting
	// back to Default.
	AllowEmpty bool
	// Validate checks a value before it is saved. Optional.
	Validate func(value string) error
	// OnChange is called with the new value after it was saved. Optional.
	OnChange func(ctx *dashygotui.Context[S], value string) tea.Cmd
}

// Addons implements [dashygotui.Addon].
func (t Text[S]) Addons() []dashygotui.Addon[S] {
	value := func(c *dashygotui.Context[S]) string {
		v := t.Default
		_, _ = c.Store().Get(t.Key, &v) // a missing or unreadable value keeps the default
		return v
	}
	return []dashygotui.Addon[S]{dashygotui.Option[S]{
		Key: t.Key, Section: t.Section, Number: t.Number, Order: t.Order, Label: t.Label,
		Hotkey: t.Hotkey, HotkeyDisplay: t.HotkeyDisplay, HotkeyHint: t.HotkeyHint,
		Badge: func(c *dashygotui.Context[S]) string {
			v := value(c)
			switch {
			case v == "":
				return c.Badge("not set")
			case t.Secret:
				return c.Badge("set")
			}
			return c.Badge(v)
		},
		Modal: func(c *dashygotui.Context[S]) dashygotui.Modal[S] {
			st := c.Store()
			var saved string
			ed := &modal.Edit[S]{
				Title: t.Label, Help: t.Help, Secret: t.Secret, Limit: t.Limit, AllowEmpty: t.AllowEmpty,
				Pending: "Saving...", Validate: t.Validate,
				Save: func(_ context.Context, v string) (string, error) {
					if v == "" {
						v = t.Default
					}
					if err := st.Set(t.Key, v); err != nil {
						return "", fmt.Errorf("saving %s: %w", t.Label, err)
					}
					saved = v
					return t.Label + " saved.", nil
				},
				OnSaved: func(c *dashygotui.Context[S], _ string) tea.Cmd {
					if t.OnChange == nil {
						return nil
					}
					return t.OnChange(c, saved)
				},
			}
			if !t.Secret {
				ed.Current, ed.Initial = value(c), value(c)
			}
			return ed
		},
	}}
}

// invalid returns an addon that reports a mistake in a setting at startup.
func invalid[S any](format string, args ...any) []dashygotui.Addon[S] {
	return []dashygotui.Addon[S]{dashygotui.Invalid[S]{Reason: fmt.Sprintf(format, args...)}}
}
